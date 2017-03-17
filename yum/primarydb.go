package yum

import (
	"database/sql"
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"sync"
)

// Script to create primary_db schema
const sqlPrimaryDBSchema = `
CREATE TABLE db_info (dbversion INTEGER, checksum TEXT);
CREATE TABLE packages (
	pkgKey INTEGER PRIMARY KEY
	, pkgId TEXT
	, name TEXT
	, arch TEXT
	, version TEXT
	, epoch TEXT
	, release TEXT
	, summary TEXT
	, description TEXT
	, url TEXT
	, time_file INTEGER
	, time_build INTEGER
	, rpm_license TEXT
	, rpm_vendor TEXT
	, rpm_group TEXT
	, rpm_buildhost TEXT
	, rpm_sourcerpm TEXT
	, rpm_header_start INTEGER
	, rpm_header_end INTEGER
	, rpm_packager TEXT
	, size_package INTEGER
	, size_installed INTEGER
	, size_archive INTEGER
	, location_href TEXT
	, location_base TEXT
	, checksum_type TEXT
);
CREATE TABLE files ( name TEXT, type TEXT, pkgKey INTEGER);
CREATE TABLE requires ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER , pre BOOLEAN DEFAULT FALSE);
CREATE TABLE provides ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER );
CREATE TABLE conflicts ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER );
CREATE TABLE obsoletes ( name TEXT, flags TEXT, epoch TEXT, version TEXT, release TEXT, pkgKey INTEGER );

CREATE INDEX packagename ON packages (name);
CREATE INDEX packageId ON packages (pkgId);
CREATE INDEX filenames ON files (name);
CREATE INDEX pkgfiles ON files (pkgKey);
CREATE INDEX pkgrequires on requires (pkgKey);
CREATE INDEX requiresname ON requires (name);
CREATE INDEX pkgprovides on provides (pkgKey);
CREATE INDEX providesname ON provides (name);
CREATE INDEX pkgconflicts on conflicts (pkgKey);
CREATE INDEX pkgobsoletes on obsoletes (pkgKey);

CREATE TRIGGER removals AFTER DELETE ON packages
BEGIN
	DELETE FROM files WHERE pkgKey = old.pkgKey;
	DELETE FROM requires WHERE pkgKey = old.pkgKey;
	DELETE FROM provides WHERE pkgKey = old.pkgKey;
	DELETE FROM conflicts WHERE pkgKey = old.pkgKey;
	DELETE FROM obsoletes WHERE pkgKey = old.pkgKey;
END;`

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

// PrimaryDB is an SQLite database which contains package data for a
// yum package repository.
type PrimaryDB struct {
	sync.Mutex

	db   *sql.DB
	path string
}

// CreatePrimaryDB initializes a new and empty primary_db SQLite database on
// disk. Any existing path is deleted.
func NewPrimaryDB(path string) (*PrimaryDB, error) {
	// create database file
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	pdb, err := OpenPrimaryDB(path)
	if err != nil {
		return nil, err
	}

	// create database tables
	if _, err = pdb.db.Exec(sqlPrimaryDBSchema); err != nil {
		return nil, fmt.Errorf("error provisioning Primary DB schema: %v", err)
	}

	return pdb, nil
}

// OpenPrimaryDB opens a primary_db SQLite database from file and return a
// pointer to the resulting struct.
func OpenPrimaryDB(path string) (*PrimaryDB, error) {
	// open database file
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// TODO: Validate primary_db on open, maybe with the db_info table

	return &PrimaryDB{
		db:   db,
		path: path,
	}, nil
}

// String implements Stringer
func (c *PrimaryDB) String() string {
	return c.Name()
}

func (c *PrimaryDB) Name() string {
	return "primary_db"
}

func (c *PrimaryDB) Path() string {
	return c.path
}

func (c *PrimaryDB) Close() error {
	c.Lock()
	defer c.Unlock()
	return c.db.Close()
}

func (c *PrimaryDB) Begin() (Tx, error) {
	c.Lock()
	tx, err := c.db.Begin()
	if err != nil {
		c.Unlock()
		return nil, err
	}

	c.Unlock() // before lock in NewPrimaryDBTx
	return NewPrimaryDBTx(tx, &c.Mutex)
}

// Packages returns all packages listed in the primary_db.
func (c *PrimaryDB) Packages() (PackageEntries, error) {
	c.Lock()
	defer c.Unlock()

	// select packages
	rows, err := c.db.Query(sqlSelectPackages)
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
func (c *PrimaryDB) DependenciesByPackage(pkgKey int, typ string) (rpm.Dependencies, error) {
	q := fmt.Sprintf("SELECT name, flags, epoch, version, release FROM %s WHERE pkgKey = %d", typ, pkgKey)

	c.Lock()
	defer c.Unlock()

	// select packages
	rows, err := c.db.Query(q)
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
func (c *PrimaryDB) FilesByPackage(pkgKey int) ([]string, error) {
	q := fmt.Sprintf("SELECT name FROM files WHERE pkgKey = %d", pkgKey)

	c.Lock()
	defer c.Unlock()

	// select packages
	rows, err := c.db.Query(q)
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
