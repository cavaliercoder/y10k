package main

import (
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	"github.com/cavaliercoder/go-rpm/yum"
	"github.com/pivotal-golang/bytefmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"xi2.org/x/xz"
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
func (c *Repo) CacheLocal(path string) error {
	Dprintf("Caching %v to %s...\n", c, path)

	// create cache folder
	if err := c.mkCacheDir(path); err != nil {
		return err
	}

	// cache metadata file
	repomd, err := c.cacheMetadata(path)
	if err != nil {
		return err
	}

	// detect primary db
	var primarydb *yum.RepoDatabase = nil
	for _, db := range repomd.Databases {
		if db.Type == "primary_db" {
			primarydb = &db
			break
		}
	}

	if primarydb == nil {
		return fmt.Errorf("No primary database found for repo %v", c)
	}

	// download primary database
	primarydb_path, err := c.downloadDatabase(path, primarydb)
	if err != nil {
		return err
	}

	// decompress primary database
	primarydb_path, err = c.decompressDatabase(path, primarydb_path, primarydb)
	if err != nil {
		return err
	}

	return nil
}

// mkCacheDir creates directories required for caching, with all missing parent
// directories.
func (c *Repo) mkCacheDir(path string) error {
	if err := os.MkdirAll(path, 0750); err != nil && os.IsNotExist(err) {
		return fmt.Errorf("Error creating cache directory %s: %v", path, err)
	}

	return nil
}

// cacheMetadata downloads a repository's repomd.xml file to the given cache
// directory.
func (c *Repo) cacheMetadata(cachedir string) (*yum.RepoMetadata, error) {
	// TODO: add support for repository mirror lists

	// TODO: prevent double forward-slash in URL joins
	repomd_url := fmt.Sprintf("%s/repodata/repomd.xml", c.BaseURL)
	repomd_path := filepath.Join(cachedir, "repomd.xml")

	// open repo metadata from URL
	// TODO: Add support for non HTTP repositories
	Dprintf("Downloading repo metadata from %s...\n", repomd_url)
	resp, err := http.Get(repomd_url)
	if err != nil {
		return nil, fmt.Errorf("Error retrieving repo metadata from URL: %v", err)
	}
	defer resp.Body.Close()

	// read repometadata into byte buffer
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading repo metadata: %v", err)
	}

	// decode repo metadata into struct
	repomd, err := yum.ReadRepoMetadata(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("Error decoding repo metadata: %v", err)
	}

	// read existing cache
	update_mdcache := false
	f, err := os.Open(repomd_path)
	if err == nil {
		defer f.Close()

		// decode existing cache
		cache_repomd, err := yum.ReadRepoMetadata(f)
		if err != nil {
			return nil, fmt.Errorf("Error decoding cached repo metadata: %v", err)
		}

		// update cache if online version is newer
		if repomd.Revision > cache_repomd.Revision {
			Dprintf("Cached metadata revision %d requires an update to revision %d\n", cache_repomd.Revision, repomd.Revision)
			update_mdcache = true
		} else {
			Dprintf("Cached metadata already at upstream revision %d\n", cache_repomd.Revision)
			update_mdcache = false
		}
	} else if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("Error reading precached repo metadata: %v", err)
	} else {
		update_mdcache = true
	}

	// cache metadata locally
	if update_mdcache {
		Dprintf("Caching repo metadata to %s...\n", repomd_path)
		if err = ioutil.WriteFile(repomd_path, b, 0640); err != nil {
			return nil, fmt.Errorf("Error caching repo metadata: %v", err)
		}
	}

	return repomd, nil
}

// downloadDatabase downloads and caches the given repository database (E.g.
// primary_db or filelists_db) to the given cache directory.
func (c *Repo) downloadDatabase(cachedir string, db *yum.RepoDatabase) (string, error) {
	// parse db paths
	db_url := fmt.Sprintf("%s/%s", c.BaseURL, db.Location.Href)
	db_path := filepath.Join(cachedir, filepath.Base(db.Location.Href))

	// check cached database
	update_db := false
	f, err := os.Open(db_path)
	if err == nil {
		err := db.Checksum.Check(f)
		if err == yum.ErrChecksumMismatch {
			// checksum mismatch
			update_db = true
			Dprintf("Cached %v database requires an update\n", db)
		} else if err == nil {
			Dprintf("Cached %v database is up to date\n", db)
		} else {
			return "", fmt.Errorf("Error reading cached %v database: %v", db, err)
		}

	} else if os.IsNotExist(err) {
		// db is not cached yet
		update_db = true
	} else {
		return "", fmt.Errorf("Error opening cached %v database: %v", db, err)
	}

	// download database
	if update_db {
		Dprintf("Downloading %v database from %s...\n", db, db_url)
		resp, err := http.Get(db_url)
		if err != nil {
			return "", fmt.Errorf("Error downloading %v database: %v", db, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("Bad response code downloading %v database: %s", db, resp.Status)
		}

		// open output file for writing
		Dprintf("Caching %v database to %s...\n", db, db_path)
		f, err := os.Create(db_path)
		if err != nil {
			return "", fmt.Errorf("Error creating cache file for %v database: %v", db, err)
		}
		defer f.Close()

		// download
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return "", fmt.Errorf("Error downloading %v database: %v", db, err)
		}
		resp.Body.Close()
		f.Close()

		// validate checksum
		if err := db.Checksum.CheckFile(db_path); err == yum.ErrChecksumMismatch {
			return "", fmt.Errorf("Database %v was download but failed checksum validation", db)
		} else if err != nil {
			return "", fmt.Errorf("Error opening downloaded %v database: %v", db, err)
		}
	}

	return db_path, nil
}

// decompressDatabase decompresses a locally cached, compressed repository
// database into the gen/ subdirectory of the given cache directory.
func (c *Repo) decompressDatabase(cachedir string, path string, db *yum.RepoDatabase) (string, error) {
	basepath := filepath.Join(cachedir, "gen")
	dpath := ""

	// create cache folder
	if err := c.mkCacheDir(basepath); err != nil {
		return "", err
	}

	// determine output path
	switch db.DatabaseVersion {
	case 0: // XML files
		dpath = filepath.Join(basepath, fmt.Sprintf("%s.xml", db.Type))

	case 10: // bzip2'd sqlite file
		dpath = filepath.Join(basepath, fmt.Sprintf("%s.sqlite", db.Type))

	default:
		return "", fmt.Errorf("Unsupported database version for %v: %d", db, db.DatabaseVersion)
	}

	// open the archive for decompression
	r, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("Error opening compressed %v database: %v", db, err)
	}
	defer r.Close()

	// select decompression type
	var z io.Reader
	if strings.HasSuffix(path, ".bz2") {
		z = bzip2.NewReader(r)

	} else if strings.HasSuffix(path, ".xz") {
		z, err = xz.NewReader(r, 0)
		if err != nil {
			return "", fmt.Errorf("Error initializing xz decompression: %v", err)
		}

	} else if strings.HasSuffix(path, ".gz") {
		z, err = gzip.NewReader(r)
		if err != nil {
			return "", fmt.Errorf("Error initializing gzip decompression: %v", err)
		}

	} else {
		return "", fmt.Errorf("Unsupported compression format for %v database: %s", db, path)
	}

	// open output file
	w, err := os.Create(dpath)
	if err != nil {
		return "", fmt.Errorf("Error creating output file for %v database: %v", db, err)
	}
	defer w.Close()

	// decompress
	_, err = io.Copy(w, z)
	if err != nil {
		return "", fmt.Errorf("Error decompressing %v database: %v", db, err)
	}
	w.Close()

	// validate checksum
	if err := db.OpenChecksum.CheckFile(dpath); err == yum.ErrChecksumMismatch {
		os.Remove(dpath)
		return "", fmt.Errorf("Decompressed %v database failed checksum validation", db)
	} else if err != nil {
		return "", fmt.Errorf("Error validating checksum for %v database: %v", db, err)
	}

	return dpath, nil
}

// Sync syncronizes a local package repository with an upstream repository using
// filter rules defined for the repository in its parent Yumfile. All repository
// metadata is cached in the given cache directory.
func (c *Repo) Sync(cachedir, packagedir string) error {

	// cache repo metadata locally to TmpYumCachePath
	if err := c.CacheLocal(cachedir); err != nil {
		return fmt.Errorf("Failed to cache metadata for repo %v: %v", c, err)
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

	// open cached primary_db
	primarydb_path := filepath.Join(cachedir, "gen/primary_db.sqlite")
	primarydb, err := yum.OpenPrimaryDB(primarydb_path)
	if err != nil {
		return fmt.Errorf("Error opening primary_db: %v", err)
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
			URL:          fmt.Sprintf("%s/%s", c.BaseURL, p.LocationHref()),
			Path:         filepath.Join(packagedir, filepath.Base(p.LocationHref())),
			Checksum:     p.Checksum(),
			ChecksumType: p.ChecksumType(),
		}
	}

	// download missing packages
	Download(jobs)

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
		}
	}

	return filtered
}
