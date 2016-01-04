package main

import (
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	"github.com/cavaliercoder/go-rpm/yum"
	"github.com/pivotal-golang/bytefmt"
	"golang.org/x/crypto/openpgp"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Repo is a package repository defined in a Yumfile
type Repo struct {
	ID             string
	Name           string
	Architecture   string
	BaseURL        string
	CachePath      string
	Checksum       string
	DeleteRemoved  bool
	GPGCheck       bool
	GPGKey         string
	Groupfile      string
	IncludeSources bool
	LocalPath      string
	MirrorURL      string
	NewOnly        bool
	MaxDate        time.Time
	MinDate        time.Time
	YumfileLineNo  int
	YumfilePath    string
}

// NewRepo initializes a new Repo struct and returns a pointer to it.
func NewRepo() *Repo {
	return &Repo{}
}

func (c Repo) String() string {
	return c.ID
}

// Validate checks the syntax of a repo defined in a Yumfile and returns an
// on the first syntax error encountered. If no errors are found, nil is
// returned.
func (c *Repo) Validate() error {
	if c.ID == "" {
		return NewErrorf("Upstream repository has no ID specified (in %s:%d)", c.YumfilePath, c.YumfileLineNo)
	}

	if c.MirrorURL == "" && c.BaseURL == "" {
		return NewErrorf("Upstream repository for '%s' has no mirror list or base URL (in %s:%d)", c.ID, c.YumfilePath, c.YumfileLineNo)
	}

	return nil
}

// CacheLocal caches a copy of a Repo's metadata and databases to the given
// cache directory. If the Repo is already cached, the cache is validated and
// updated if the source repository has been updated.
func (c *Repo) CacheLocal(path string) (*RepoCache, error) {
	Dprintf("Caching %v to %s...\n", c, path)

	// connect to cache
	cache, err := NewCache(path)
	if err != nil {
		return nil, err
	}

	// get cache for this repo
	repocache, err := cache.NewRepoCache(c)
	if err != nil {
		return nil, err
	}

	// update cache
	if err := repocache.Update(); err != nil {
		return nil, err
	}

	return repocache, nil
}

// Sync syncronizes a local package repository with an upstream repository using
// filter rules defined for the repository in its parent Yumfile. All repository
// metadata is cached in the given cache directory.
func (c *Repo) Sync(cachedir, packagedir string) error {
	var err error

	// load gpg keys
	var keyring openpgp.KeyRing
	if c.GPGCheck {
		keyring, err = c.keyring()
		if err != nil {
			return err
		}
	}

	// cache repo metadata locally to TmpYumCachePath
	repocache, err := c.CacheLocal(cachedir)
	if err != nil {
		return fmt.Errorf("Failed to cache metadata for repo %v: %v", c, err)
	}

	// get primary db from cache
	primarydb, err := repocache.PrimaryDB()
	if err != nil {
		return err
	}

	// create package directory
	if err := os.MkdirAll(packagedir, 0750); err != nil && !os.IsExist(err) {
		return fmt.Errorf("Error creating local package path %s: %v", packagedir, err)
	}

	// list existing files
	files, err := ioutil.ReadDir(packagedir)
	if err != nil {
		return fmt.Errorf("Error reading packages")
	}

	// load packages from primary_db
	Dprintf("Loading package metadata from primary_db...\n")
	packages, err := primarydb.Packages()
	if err != nil {
		return fmt.Errorf("Error reading packages from primary_db: %v", err)
	}

	// filter list
	packages = c.FilterPackages(packages)
	Dprintf("Found %d packages in primary_db\n", len(packages))

	// build a list of missing packages
	Dprintf("Checking for existing packages in %s...\n", packagedir)
	missing := make([]yum.PackageEntry, 0)
	var totalsize uint64 = 0
	for _, p := range packages {
		package_filename := filepath.Base(p.LocationHref())
		package_path := filepath.Join(packagedir, filepath.Base(p.LocationHref()))

		// search local files
		found := false
		for _, filename := range files {
			if filename.Name() == package_filename {

				// validate checksum
				err = yum.ValidateFileChecksum(package_path, p.Checksum(), p.ChecksumType())
				if err == yum.ErrChecksumMismatch {
					Errorf(err, "Existing file failed checksum validation for package %v", p)
					break

				} else if err != nil {
					Errorf(err, "Error validating checksum for package %v", p)
					break

				}

				// valid package found
				found = true
				break
			}
		}

		// TODO: filter packages according to Yumfile rules

		if !found {
			missing = append(missing, p)
			totalsize += uint64(p.PackageSize())
		}
	}

	Dprintf("Scheduled %d packages for download (%s)\n", len(missing), bytefmt.ByteSize(totalsize))

	// schedule download jobs
	jobs := make([]DownloadJob, len(missing))
	for i, p := range missing {
		// create download job
		jobs[i] = DownloadJob{
			Label:        p.String(),
			URL:          urljoin(c.BaseURL, p.LocationHref()),
			Size:         uint64(p.PackageSize()),
			Path:         filepath.Join(packagedir, filepath.Base(p.LocationHref())),
			Checksum:     p.Checksum(),
			ChecksumType: p.ChecksumType(),
		}
	}

	// download missing packages
	complete := make(chan DownloadJob, 0)
	go Download(jobs, complete)

	// handle each finished package
	// TODO: create more gpgcheck threads
	for job := range complete {
		if job.Error != nil {
			Errorf(job.Error, "Error downloading %s", job.Label)
		} else {
			// open downloaded package for reading
			f, err := os.Open(job.Path)
			if err != nil {
				Errorf(err, "Error reading %s for GPG check", job.Label)
			} else {
				defer f.Close()

				// gpg check
				_, err = rpm.GPGCheck(f, keyring)
				if err != nil {
					Errorf(err, "GPG check validation failed for %s", job.Label)

					// delete bad package
					if err := os.Remove(job.Path); err != nil {
						Errorf(err, "Error deleting %v", job.Label)
					}
				}
			}
		}
	}

	return nil
}

// FilterPackages returns a list of packages filtered according the repo's
// settings.
func (c *Repo) FilterPackages(packages yum.PackageEntries) yum.PackageEntries {
	newest := make(map[string]*yum.PackageEntry, 0)

	// calculate which packages are the latest
	if c.NewOnly {
		for i, p := range packages {
			// index on name and architecture
			id := fmt.Sprint("%s.%s", p.Name(), p.Architecture())

			// lookup previous index
			if n, ok := newest[id]; ok {
				// compare version with previous index
				if 1 == rpm.VersionCompare(rpm.Package(&p), rpm.Package(n)) {
					newest[id] = &packages[i]
				}
			} else {
				// add new index for this package
				newest[id] = &packages[i]
			}
		}

		// replace packages with only the latest packages
		i := 0
		packages = make(yum.PackageEntries, len(newest))
		for _, p := range newest {
			packages[i] = *p
			i++
		}
	}

	// filter the package list
	filtered := make(yum.PackageEntries, 0)
	for _, p := range packages {
		include := true

		// filter by architecture
		if c.Architecture != "" {
			if p.Architecture() != c.Architecture {
				include = false
			}
		}

		// filter by minimum build date
		if !c.MinDate.IsZero() {
			if p.BuildTime().Before(c.MinDate) {
				include = false
			}
		}

		// filter by maximum build date
		if !c.MaxDate.IsZero() {
			if p.BuildTime().After(c.MaxDate) {
				include = false
			}
		}

		// append to output
		if include {
			filtered = append(filtered, p)
		} else {
			Dprintf("Exluding: %v\n", p)
		}
	}

	return filtered
}

// keyring returns the GPG keyring for the given gpgkey file.
func (c *Repo) keyring() (openpgp.KeyRing, error) {
	gpgkey := c.GPGKey

	// check gpgkey is specified
	if gpgkey == "" {
		return nil, fmt.Errorf("gpgkey not specified")
	}

	// trim file:// prefix
	if strings.HasPrefix(strings.ToLower(gpgkey), "file://") {
		gpgkey = gpgkey[7:]
	}

	keyring, err := rpm.KeyRingFromFile(gpgkey)
	if err != nil {
		return nil, fmt.Errorf("Error reading GPG key: %v", err)
	}

	return keyring, nil
}
