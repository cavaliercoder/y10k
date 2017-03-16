package yum

import (
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	"github.com/cavaliercoder/y10k/yum/compress"
	"github.com/cavaliercoder/y10k/yum/crypto"
	"os"
	"path/filepath"
	"time"
)

type Repo struct {
	// BasePath is a directory where the package repository file structure will
	// be written.
	BasePath string

	// list of active databases
	dbs map[string]DB
}

func NewRepo(path string) *Repo {
	return &Repo{
		BasePath: path,
		dbs:      make(map[string]DB, 0),
	}
}

// Bootstrap creates the configured repository file structure and databases.
func (c *Repo) Bootstrap() error {
	// create repodata directory
	rdata := filepath.Join(c.BasePath, "/repodata")
	gen := filepath.Join(rdata, "/gen")
	if err := os.MkdirAll(gen, 0755); err != nil {
		return err
	}

	// create primary db
	pdbPath := filepath.Join(c.BasePath, "/repodata/gen/primary_db.sqlite")
	// TODO: reopen existing database
	pdb, err := NewPrimaryDB(pdbPath)
	if err != nil {
		return fmt.Errorf("Error creating primary database: %v", err)
	}
	c.dbs["primary_db"] = pdb

	return nil
}

// Publish compresses all databases, updates the metadata doc and deletes all
// source files in `repodata/gen/`.
func (c *Repo) Publish() error {
	// build metadata document
	repomd := &RepoMetadata{
		Revision:  int(time.Now().Unix()),
		Databases: make([]RepoDatabase, 0),
	}

	// add primary db
	for _, db := range c.dbs {
		m, err := c.publishDB(db)
		if err != nil {
			return err
		}
		repomd.Databases = append(repomd.Databases, *m)
	}

	// write XML document
	repomdPath := filepath.Join(c.BasePath, "/repodata/repomd.xml")
	w, err := os.OpenFile(repomdPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer w.Close()

	if err := repomd.Write(w); err != nil {
		return err
	}

	// delete repodata/gen/
	/* TODO
	gen := filepath.Join(c.BasePath, "/repodata/gen/")
	if err := os.RemoveAll(gen); err != nil {
		return fmt.Errorf("Error cleaning up repodata: %v", err)
	}
	*/

	return nil
}

func (c *Repo) publishDB(db DB) (*RepoDatabase, error) {
	// finalize the database
	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("Error closing %v: %v", db, err)
	}

	// TODO: select the correct compressor
	cmp := compress.NewBzip2Compressor()

	// compress to temp file and return its path
	tmp, err := cmp.CompressToTemp(db.Path())
	if err != nil {
		return nil, err
	}

	// TODO: select correct checksum algorithm
	h := crypto.NewSha256()

	// checksum the file
	sum, err := h.ChecksumFile(tmp)
	if err != nil {
		return nil, err
	}

	// TODO: select the correct db suffix

	// move the compressed file into place
	name := fmt.Sprintf("%s-primary.sqlite.bz2", sum)
	rel := filepath.Join("repodata", name)
	path := filepath.Join(c.BasePath, rel)
	if err := os.Rename(tmp, path); err != nil {
		return nil, err
	}

	// create metadata
	m := &RepoDatabase{
		Type:            db.Name(),
		DatabaseVersion: 10,
		Location:        RepoDatabaseLocation{rel},
	}

	// stat uncompressed file
	if f, err := os.Open(db.Path()); err != nil {
		return nil, err
	} else {
		defer f.Close()

		if fi, err := f.Stat(); err != nil {
			return nil, err
		} else {
			m.Timestamp = fi.ModTime().Unix()
			m.OpenSize = int(fi.Size())
		}

		if sum, err := h.Checksum(f); err != nil {
			return nil, err
		} else {
			m.OpenChecksum = RepoDatabaseChecksum{"sha256", sum}
		}
	}

	// stat compressed file
	if f, err := os.Open(path); err != nil {
		return nil, err
	} else {
		defer f.Close()

		if fi, err := f.Stat(); err != nil {
			return nil, err
		} else {
			m.Size = int(fi.Size())
		}

		if sum, err := h.Checksum(f); err != nil {
			return nil, err
		} else {
			m.Checksum = RepoDatabaseChecksum{"sha256", sum}
		}
	}

	return m, nil
}

// AddPackage adds an RPM package to all active databases in the repository.
func (c *Repo) AddPackage(p *rpm.PackageFile) error {
	for key, db := range c.dbs {
		// TODO: reuse transactions
		tx, err := db.Begin()
		if err != nil {
			return err
		}

		if err := tx.AddPackage(p); err != nil {
			return fmt.Errorf("Error adding package %v to %v: %v", p, key, err)
		}

		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Repo) PrimaryDB() *PrimaryDB {
	return c.dbs["primary_db"].(*PrimaryDB)
}
