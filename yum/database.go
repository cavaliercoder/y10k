package yum

import (
	"github.com/cavaliercoder/go-rpm"
)

// DB is one of multiple possible databases in a Yum repository.
type DB interface {
	// Name is the type of database as it appears in repomd.xml (E.g.
	// "primary_db")
	Name() string

	// File is file path of the uncompressed database file.
	Path() string

	// Begin starts a transaction.
	Begin() (Tx, error)

	// Close closes the database, releasing any open resources. It should also
	// repackage any assets for distribution (such as gzipping a modified XML
	// document).
	Close() error
}

// Tx is an in-progress database transaction.
// A transaction must end with a call to Commit or Rollback.
type Tx interface {
	// Commit commits the transaction.
	Commit() error

	// Rollback aborts the transaction.
	Rollback() error

	// Add new RPM packages to the database.
	AddPackage(...*rpm.PackageFile) error
}
