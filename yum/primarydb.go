package yum

import (
	"database/sql"
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"strings"
)

// TODO: Add support for XML primary dbs

// Queries to create primary_db schema
const (
	sqlCreateTables = `CREATE TABLE db_info (dbversion INTEGER, checksum TEXT);
CREATE TABLE packages ( pkgKey INTEGER PRIMARY KEY, pkgId TEXT, name TEXT, arch TEXT, version TEXT, epoch TEXT, release TEXT, summary TEXT, description TEXT, url TEXT, time_file INTEGER, time_build INTEGER, rpm_license TEXT, rpm_vendor TEXT, rpm_group TEXT, rpm_buildhost TEXT, rpm_sourcerpm TEXT, rpm_header_start INTEGER, rpm_header_end INTEGER, rpm_packager TEXT, size_package INTEGER, size_installed INTEGER, size_archive INTEGER, location_href TEXT, location_base TEXT, checksum_type TEXT);
CREATE TABLE files ( name TEXT, type TEXT, pkgKey INTEGER);
CREATE TABLE requires ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER , pre BOOLEAN DEFAULT FALSE);
CREATE TABLE provides ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER );
CREATE TABLE conflicts ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER );
CREATE TABLE obsoletes ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER );`

	sqlCreateTriggers = `CREATE TRIGGER removals AFTER DELETE ON packages  BEGIN    DELETE FROM files WHERE pkgKey = old.pkgKey;    DELETE FROM requires WHERE pkgKey = old.pkgKey;    DELETE FROM provides WHERE pkgKey = old.pkgKey;    DELETE FROM conflicts WHERE pkgKey = old.pkgKey;    DELETE FROM obsoletes WHERE pkgKey = old.pkgKey;  END;`

	sqlCreateIndexes = `CREATE INDEX packagename ON packages (name);
CREATE INDEX packageId ON packages (pkgId);
CREATE INDEX filenames ON files (name);
CREATE INDEX pkgfiles ON files (pkgKey);
CREATE INDEX pkgrequires on requires (pkgKey);
CREATE INDEX requiresname ON requires (name);
CREATE INDEX pkgprovides on provides (pkgKey);
CREATE INDEX providesname ON provides (name);
CREATE INDEX pkgconflicts on conflicts (pkgKey);
CREATE INDEX pkgobsoletes on obsoletes (pkgKey);`
)

const sqlSelectPackages = `SELECT
 pkgKey
 , name
 , arch
 , epoch
 , version
 , release
 , size_package
 , size_installed
 , size_archive
 , location_href
 , pkgId
 , checksum_type
 , time_build
FROM packages;`

const sqlInsertPackage = `INSERT INTO packages(
 name
 , arch
 , epoch
 , version
 , release
 , summary
 , description
 , url
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
 , size_package
) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

// PrimaryDatabase is an SQLite database which contains package data for a
// yum package repository.
type PrimaryDatabase struct {
	dbpath string
}

// CreatePrimaryDB initializes a new and empty primary_db SQLite database on
// disk. Any existing path is deleted.
func CreatePrimaryDB(path string) (*PrimaryDatabase, error) {
	// create database file
	os.Remove(path)
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("Error creating Primary DB: %v", err)
	}
	defer db.Close()

	// create database tables
	_, err = db.Exec(sqlCreateTables)
	if err != nil {
		return nil, fmt.Errorf("Error creating Primary DB tables: %v", err)
	}

	// create database indexes
	_, err = db.Exec(sqlCreateIndexes)
	if err != nil {
		return nil, fmt.Errorf("Error creating Primary DB indexes: %v", err)
	}

	// create database triggers
	_, err = db.Exec(sqlCreateTriggers)
	if err != nil {
		return nil, fmt.Errorf("Error creating Primary DB triggers: %v", err)
	}

	return OpenPrimaryDB(path)
}

// OpenPrimaryDB opens a primary_db SQLite database from file and return a
// pointer to the resulting struct.
func OpenPrimaryDB(path string) (*PrimaryDatabase, error) {
	// open database file
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// TODO: Validate primary_db on open, maybe with the db_info table

	return &PrimaryDatabase{
		dbpath: path,
	}, nil
}

func (c *PrimaryDatabase) InsertPackage(packages ...*rpm.PackageFile) error {
	// open database file
	db, err := sql.Open("sqlite3", c.dbpath)
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare(sqlInsertPackage)
	if err != nil {
		return err
	}

	defer stmt.Close()

	for _, p := range packages {
		sum, err := p.Checksum()
		if err != nil {
			return err
		}

		href := filepath.Base(p.Path())
		_, err = stmt.Exec(
			p.Name(),
			p.Architecture(),
			p.Epoch(),
			p.Version(),
			p.Release(),
			p.Summary(),
			p.Description(),
			p.URL(),
			0,
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
			p.Packager(),
			p.FileSize())

		fmt.Printf("%v\n", p.FileSize())
		if err != nil {
			return err
		}
	}

	return nil
}

// Packages returns all packages listed in the primary_db.
func (c *PrimaryDatabase) Packages() (PackageEntries, error) {
	// open database file
	db, err := sql.Open("sqlite3", c.dbpath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// select packages
	rows, err := db.Query(sqlSelectPackages)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// parse each row as a package
	packages := make(PackageEntries, 0)
	for rows.Next() {
		p := PackageEntry{
			db: c,
		}

		// scan the values into the slice
		if err = rows.Scan(&p.key, &p.name, &p.architecture, &p.epoch, &p.version, &p.release, &p.package_size, &p.install_size, &p.archive_size, &p.locationhref, &p.checksum, &p.checksum_type, &p.time_build); err != nil {
			return nil, fmt.Errorf("Error scanning packages: %v", err)
		}

		packages = append(packages, p)
	}

	return packages, nil
}

// DependenciesByPackage returns all package dependencies of the given type for
// the given package key. The dependency type may be one of 'requires',
// 'provides', 'conflicts' or 'obsoletes'.
func (c *PrimaryDatabase) DependenciesByPackage(pkgKey int, typ string) (rpm.Dependencies, error) {
	q := fmt.Sprintf("SELECT name, flags, epoch, version, release FROM %s WHERE pkgKey = %d", typ, pkgKey)

	// open database file
	db, err := sql.Open("sqlite3", c.dbpath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// select packages
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// parse results
	deps := make(rpm.Dependencies, 0)
	for rows.Next() {
		var flgs, name, version, release string
		var epoch, iflgs int

		if err = rows.Scan(&name, &flgs, &epoch, &version, &release); err != nil {
			return nil, fmt.Errorf("Error reading dependencies: %v", err)
		}

		switch flgs {
		case "EQ":
			iflgs = rpm.DepFlagEqual

		case "LT":
			iflgs = rpm.DepFlagLesser

		case "LE":
			iflgs = rpm.DepFlagLesserOrEqual

		case "GE":
			iflgs = rpm.DepFlagGreaterOrEqual

		case "GT":
			iflgs = rpm.DepFlagGreater
		}

		deps = append(deps, rpm.NewDependency(iflgs, name, epoch, version, release))
	}

	return deps, nil
}

// FilesByPackage returns all known files included in the package of the given
// package key.
func (c *PrimaryDatabase) FilesByPackage(pkgKey int) ([]string, error) {
	q := fmt.Sprintf("SELECT name FROM files WHERE pkgKey = %d", pkgKey)

	// open database file
	db, err := sql.Open("sqlite3", c.dbpath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// select packages
	rows, err := db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// parse results
	files := make([]string, 0)
	for rows.Next() {
		var file string
		if err := rows.Scan(&file); err != nil {
			return nil, fmt.Errorf("Error reading files: %v", err)
		}

		files = append(files, file)
	}

	return files, nil
}
