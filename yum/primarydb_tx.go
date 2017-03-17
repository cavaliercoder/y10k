package yum

import (
	"database/sql"
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
	"strings"
	"sync"
)

const (
	sqlInsertPackage = `INSERT INTO packages(
 name
 , arch
 , epoch
 , version
 , release
 , summary
 , description
 , url
 , time_file
 , size_package
 , size_installed
 , size_archive
 , location_href
 , pkgId
 , checksum_type
 , time_build
 , rpm_license
 , rpm_vendor
 , rpm_group
 , rpm_buildhost
 , rpm_sourcerpm
 , rpm_header_start
 , rpm_header_end
 , rpm_packager
) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	sqlInsertPackageFile = `INSERT INTO files(name, type, pkgKey) VALUES (?, ?, ?);`
)

// primaryDBTx is a SQLite transaction opened on the primary database.
type primaryDBTx struct {
	*sync.Mutex

	tx            *sql.Tx
	insertPackage *sql.Stmt
	insertFile    *sql.Stmt
}

func NewPrimaryDBTx(tx *sql.Tx, mu *sync.Mutex) (Tx, error) {
	c := &primaryDBTx{mu, tx, nil, nil}

	c.Lock()
	defer c.Unlock()

	// prepare insert package query
	if stmt, err := tx.Prepare(sqlInsertPackage); err != nil {
		return nil, err
	} else {
		c.insertPackage = stmt
	}

	// prepare insert file query
	if stmt, err := tx.Prepare(sqlInsertPackageFile); err != nil {
		return nil, err
	} else {
		c.insertFile = stmt
	}

	return c, nil
}

func (c *primaryDBTx) Commit() error {
	c.Lock()
	defer c.Unlock()
	return c.tx.Commit()
}

func (c *primaryDBTx) Rollback() error {
	c.Lock()
	defer c.Unlock()
	return c.tx.Rollback()
}

func (c *primaryDBTx) AddPackage(packages ...*rpm.PackageFile) error {
	for _, p := range packages {
		sum, err := p.Checksum()
		if err != nil {
			return err
		}

		href := filepath.Base(p.Path())

		c.Lock()
		defer c.Unlock()

		res, err := c.insertPackage.Exec(
			p.Name(),
			p.Architecture(),
			p.Epoch(),
			p.Version(),
			p.Release(),
			p.Summary(),
			p.Description(),
			p.URL(),
			p.FileTime().Unix(),
			p.FileSize(),
			p.Size(),
			p.ArchiveSize(),
			href,
			sum,
			p.ChecksumType(),
			p.BuildTime().Unix(),
			p.License(),
			p.Vendor(),
			strings.Join(p.Groups(), "\n"),
			p.BuildHost(),
			p.SourceRPM(),
			p.HeaderStart(),
			p.HeaderEnd(),
			p.Packager())

		if err != nil {
			return err
		}

		i, err := res.LastInsertId()
		if err != nil {
			return err
		}

		// insert files
		files := p.Files()
		for _, f := range files {
			_, err := c.insertFile.Exec(f.Name(), "file", i)

			if err != nil {
				return fmt.Errorf("error inserting file %v for %v: %v", f.Name(), p, err)
			}
		}
	}

	return nil
}
