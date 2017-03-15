package yum

import (
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	"os"
	"path/filepath"
	"time"
)

type Repo struct {
	// BasePath is a directory where the package repository file structure will
	// be written.
	BasePath string

	// list of active databases
	dbs map[string]PackageDatabase
}

func NewRepo(path string) *Repo {
	return &Repo{
		BasePath: path,
		dbs:      make(map[string]PackageDatabase, 0),
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
	pdb.Bzip2Path = filepath.Join(c.BasePath, "/repodata")
	c.dbs["primary_db"] = pdb

	return nil
}

// AddPackage adds an RPM package to all active databases in the repository.
func (c *Repo) AddPackage(p *rpm.PackageFile) error {
	for key, db := range c.dbs {
		if err := db.AddPackage(p); err != nil {
			return fmt.Errorf("Error adding package %v to %v: %v", p, key, err)
		}
	}

	return nil
}

// UpdateMetadata write the repodata/repomd.xml file.
func (c *Repo) UpdateMetadata() error {
	repomd := &RepoMetadata{
		Revision:  int(time.Now().Unix()),
		Databases: make([]RepoDatabase, 0),
	}

	// add primary db
	if pdb := c.PrimaryDB(); pdb != nil {
		m, err := c.PrimaryDB().Metadata()
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

	return nil
}

func (c *Repo) Close() error {
	// close databases
	for key, db := range c.dbs {
		if err := db.Close(); err != nil {
			return fmt.Errorf("Error closing %v: %v", key, err)
		}
	}

	// update metadata document
	if err := c.UpdateMetadata(); err != nil {
		return fmt.Errorf("Error writing repo metadata: %v", err)
	}

	return nil
}

func (c *Repo) PrimaryDB() *PrimaryDB {
	return c.dbs["primary_db"].(*PrimaryDB)
}
