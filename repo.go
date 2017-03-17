package main

import (
	"./yum"
	"code.cloudfoundry.org/bytefmt"
	"encoding/hex"
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	"github.com/cavaliercoder/grab"
	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sync"
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
//
// Analogous to the `reposync` command.
func (c *Repo) Sync(cachedir, packagedir string) error {
	var err error

	Printf("Syncronizing %v to %v...\n", c, packagedir)

	// load gpg keys
	var keyring openpgp.KeyRing
	if c.GPGCheck {
		keyring, err = OpenKeyRing(c.GPGKey)
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
	packages = FilterPackages(c, packages)
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
		for _, fi := range files {
			// find file for package
			if fi.Name() == package_filename {
				// check file size
				if fi.Size() == p.PackageSize() {
					// validate checksum
					sum, err := p.Checksum()
					if err != nil {
						Errorf(err, "Failed to compute checksum for package %v", p)
						break
					}

					err = yum.ValidateFileChecksum(package_path, sum, p.ChecksumType())
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

				} else if fi.Size() > p.PackageSize() {
					// existing file is too large (smaller is okay)
					Errorf(err, "Existing file is larger (%s) than expected (%s) for package %v", bytefmt.ByteSize(uint64(fi.Size())), bytefmt.ByteSize(uint64(p.PackageSize())), p)
					break
				} else {
					Printf("Existing file is incomplete for package %v\n", p)
				}
			}
		}

		// TODO: filter packages according to Yumfile rules

		if !found {
			missing = append(missing, p)
			totalsize += uint64(p.PackageSize())
		}
	}

	if len(missing) == 0 {
		Dprintf("No packages scheduled for download\n")
	} else {
		Printf("Downloading %d new packages (%s)\n", len(missing), bytefmt.ByteSize(totalsize))
	}

	// schedule download jobs
	reqs := make([]*grab.Request, 0)
	for i, p := range missing {
		req, err := grab.NewRequest(urljoin(c.BaseURL, p.LocationHref()))
		if err != nil {
			Errorf(err, "Error requesting package %v", p)
		} else {
			req.Label = fmt.Sprintf("[ %d / %d ] %v", i+1, len(missing), p)
			req.Filename = filepath.Join(packagedir, filepath.Base(p.LocationHref()))
			req.Size = uint64(p.PackageSize())
			sum, err := p.Checksum()
			if err != nil {
				Errorf(err, "Error reading checksum for package %v", p)
			} else {
				b, err := hex.DecodeString(sum)
				if err != nil {
					Errorf(err, "Error decoding checksum for package %p", p)
				} else {
					req.SetChecksum(p.ChecksumType(), b)
					reqs = append(reqs, req)
				}
			}
		}
	}

	// download missing packages
	responses := download(reqs, DownloadThreads)

	// handle each finished package
	for resp := range responses {
		if resp.Error != nil {
			Errorf(resp.Error, "Error downloading %s", resp.Request.Label)
		} else {
			// gpg check
			// TODO: create more gpgcheck threads
			if c.GPGCheck {
				// open downloaded package for reading
				f, err := os.Open(resp.Filename)
				if err != nil {
					Errorf(err, "Error reading %s for GPG check", resp.Request.Label)
				} else {
					defer f.Close()

					// gpg check
					_, err = rpm.GPGCheck(f, keyring)
					if err != nil {
						Errorf(err, "GPG check validation failed for %s", resp.Request.Label)

						// delete bad package
						if err := os.Remove(resp.Filename); err != nil {
							Errorf(err, "Error deleting %v", resp.Request.Label)
						}
					}
				}
			}
		}
	}

	// createrepo
	return c.UpdateDB(packagedir)
}

// UpdateDB creates or updates all databases and metadata for a package
// repository.
//
// Analogous to the `createrepo` command.
//
// `/repodata` is always appended to the given path and packages are considered
// to be relative to this path.
func (c *Repo) UpdateDB(path string) error {
	// init repo file structure
	repo := yum.NewRepo(path)
	if err := repo.Bootstrap(); err != nil {
		return errors.Wrap(err, "error bootstrapping repository")
	}

	// enumerate package dir
	glob := filepath.Join(path, "/*.rpm")
	files, err := filepath.Glob(glob)
	if err != nil {
		return errors.Wrapf(err, "error globbing %v", glob)
	}
	Dprintf("Found %v packages matching %v\n", len(files), glob)

	// TODO: filter and GPG check packages here

	// start workers
	ch := make(chan *rpm.PackageFile)
	workerCount := runtime.NumCPU()
	wg := &sync.WaitGroup{}
	wg.Add(workerCount)

	Dprintf("Starting %v workers\n", workerCount)
	for i := 0; i < workerCount; i++ {
		go func(worker int) {
			defer wg.Done()

			for p := range ch {
				Dprintf("[worker %v] added %v\n", worker, p)
				if err := repo.AddPackage(p); err != nil {
					Errorf(err, "[worker %v] error adding package %v", worker, p)
					// TODO: signal parent that an error occurred
				}
			}
		}(i + 1)
	}

	// send packages to ch
	for _, f := range files {
		p, err := rpm.OpenPackageFile(f)
		if err != nil {
			return errors.Wrapf(err, "failed to open package %v", f)
		}
		ch <- p
	}

	close(ch)
	wg.Wait()

	if err := repo.Publish(); err != nil {
		return errors.Wrap(err, "error publishing repository databases")
	}

	Printf("Updated local database for %v in %v\n", c, path)

	return nil
}
