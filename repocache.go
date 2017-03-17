package main

import (
	"./yum"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"xi2.org/x/xz"
)

type RepoCache struct {
	Repo *Repo
	Path string
}

func (c *RepoCache) Update() error {
	// cache metadata file
	repomd, err := c.updateMetadata()
	if err != nil {
		return err
	}

	for _, db := range repomd.Databases {
		if _, err := c.downloadDatabase(&db); err != nil {
			return err
		}

		if db.IsCompressed() {
			if _, err = c.decompressDatabase(&db); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *RepoCache) PrimaryDB() (*yum.PrimaryDB, error) {
	path := filepath.Join(c.Path, "gen/primary_db.sqlite")
	return yum.OpenPrimaryDB(path)
}

// cacheMetadata downloads a repository's repomd.xml file to the given cache
// directory.
func (c *RepoCache) updateMetadata() (*yum.RepoMetadata, error) {
	// TODO: add support for repository mirror lists
	repomd_url := urljoin(c.Repo.BaseURL, "/repodata/repomd.xml")
	repomd_path := filepath.Join(c.Path, "repomd.xml")

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
		if err = os.MkdirAll(c.Path, 0750); err != nil {
			return nil, err
		}

		if err = ioutil.WriteFile(repomd_path, b, 0640); err != nil {
			return nil, err
		}
	}

	return repomd, nil
}

// downloadDatabase downloads and caches the given repository database (E.g.
// primary_db or filelists_db) to the given cache directory.
func (c *RepoCache) downloadDatabase(db *yum.RepoDatabase) (string, error) {
	// parse db paths
	db_url := urljoin(c.Repo.BaseURL, db.Location.Href)
	db_path := filepath.Join(c.Path, filepath.Base(db.Location.Href))

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
func (c *RepoCache) decompressDatabase(db *yum.RepoDatabase) (string, error) {
	basepath := filepath.Join(c.Path, "gen")
	path := filepath.Join(c.Path, filepath.Base(db.Location.Href))
	dpath := ""

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
