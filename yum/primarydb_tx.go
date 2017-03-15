package yum

import (
	"database/sql"
	"github.com/cavaliercoder/go-rpm"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
	"strings"
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

	sqlInsertPackageFiles = `INSERT INTO files(name, type, pkgKey) VALUES (?, ?, ?);`
)

// PrimaryDBTx is a database transaction opened on the primary database.
type PrimaryDBTx struct {
	tx *sql.Tx
}

func (c *PrimaryDBTx) Commit() error {
	return c.tx.Commit()
}

func (c *PrimaryDBTx) InsertPackage(packages ...*rpm.PackageFile) error {
	// insert package
	stmt, err := c.tx.Prepare(sqlInsertPackage)
	if err != nil {
		return err
	}

	defer stmt.Close()

	// insert files
	stmtFiles, err := c.tx.Prepare(sqlInsertPackageFiles)
	if err != nil {
		return err
	}

	defer stmtFiles.Close()

	for _, p := range packages {
		sum, err := p.Checksum()
		if err != nil {
			return err
		}

		href := filepath.Base(p.Path())
		res, err := stmt.Exec(
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
			stmtFiles.Exec(f, "file", i)
		}
	}

	return nil
}
