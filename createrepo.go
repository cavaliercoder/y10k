package main

import (
	"github.com/cavaliercoder/go-rpm"
	"github.com/cavaliercoder/y10k/yum"
	"os"
	"path/filepath"
)

// PrimaryDatabaseWriter writes packages to a Primary Database. It is the
// callers responsibility to close the channel when all packages have been sent.
type PrimaryDatabaseWriter chan<- *rpm.PackageFile

func (w PrimaryDatabaseWriter) Write(p *rpm.PackageFile) {
	w <- p
}

func (w PrimaryDatabaseWriter) Close() {
	close(w)
}

// createrepo create the required databases and metadata for a package
// repository.
//
// `/repodata` is always appended to the given path.
func createrepo(path string) (PrimaryDatabaseWriter, error) {
	// create repodata directory
	dbPath := filepath.Join(path, "/gen")
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, err
	}

	// create primary db
	pdbPath := filepath.Join(dbPath, "/primary_db.sqlite")
	db, err := yum.CreatePrimaryDB(pdbPath)
	if err != nil {
		return nil, err
	}

	// create package channel
	w := make(chan *rpm.PackageFile, 0)
	go func(w chan *rpm.PackageFile) {
		for p := range w {
			if err := db.InsertPackage(p); err != nil {
				Errorf(err, "Failed to insert %v", p)
			}
		}

		// TODO: finalise database
	}(w)

	return w, nil
}
