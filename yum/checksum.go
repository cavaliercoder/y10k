package yum

import (
	"fmt"
	"github.com/cavaliercoder/y10k/yum/crypto"
	"io"
	"os"
)

// ErrChecksumMismatch indicates that the checksum value of two items does not
// match.
var ErrChecksumMismatch = fmt.Errorf("Checksum mismatch")

// RepoDatabaseChecksum is the XML element of a repo metadata file which
// describes the checksum required to validate a repository database.
type RepoDatabaseChecksum struct {
	Type string `xml:"type,attr"`
	Hash string `xml:",chardata"`
}

// Check creates a checksum of the given io.Reader content and compares it the
// the expected checksum value. If the checksums match, nil is returned. If the
// checksums do not match, ErrChecksumMismatch is returned. If any other error
// occurs, the error is returned.
func (c *RepoDatabaseChecksum) Check(r io.Reader) error {
	return ValidateChecksum(r, c.Hash, c.Type)
}

// CheckFile creates a checksum of the given file content and compares it the
// the expected checksum value. If the checksums match, nil is returned. If the
// checksums do not match, ErrChecksumMismatch is returned. If any other error
// occurs, the error is returned.
func (c *RepoDatabaseChecksum) CheckFile(name string) error {
	return ValidateFileChecksum(name, c.Hash, c.Type)
}

// ValidateChecksum creates a checksum of the given io.Reader content and
// compares it the the given checksum value. If the checksums match, nil is
// returned. If the checksums do not match, ErrChecksumMismatch is returned. If
// any other error occurs, the error is returned.
func ValidateChecksum(r io.Reader, checksum string, checksum_type string) error {
	// get checksum value based by type
	actual := ""
	var err error
	switch checksum_type {
	case "sha256":
		actual, err = crypto.NewSha256().Checksum(r)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("Unsupported checksum type: %s", checksum_type)
	}

	// check against expected value
	if checksum != actual {
		return ErrChecksumMismatch
	}

	return nil
}

// ValidateChecksum creates a checksum of the given file content and compares it
// the the given checksum value. If the checksums match, nil is returned. If the
// checksums do not match, ErrChecksumMismatch is returned. If any other error
// occurs, the error is returned.
func ValidateFileChecksum(name string, checksum string, checksum_type string) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}

	defer f.Close()

	return ValidateChecksum(f, checksum, checksum_type)
}
