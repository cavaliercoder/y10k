package yum

import (
	"github.com/cavaliercoder/go-rpm"
)

// A RepoTX is a "super-transaction" that applies all its actions to a list of
// underlying database transactions.
type RepoTx []Tx

// Commit commits all underlying database transactions.
func (c RepoTx) Commit() error {
	for _, tx := range c {
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	return nil
}

// Rollback aborts all underlying database transactions.
func (c RepoTx) Rollback() error {
	for _, tx := range c {
		if err := tx.Rollback(); err != nil {
			return err
		}
	}

	return nil
}

// Add new RPM packages to all underlying database.
func (c RepoTx) AddPackage(pkgs ...*rpm.PackageFile) error {
	for _, tx := range c {
		if err := tx.AddPackage(pkgs...); err != nil {
			return err
		}
	}

	return nil
}
