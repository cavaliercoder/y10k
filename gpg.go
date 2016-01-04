package main

import (
	"fmt"
	"github.com/cavaliercoder/go-rpm"
	"golang.org/x/crypto/openpgp"
	"strings"
)

// OpenKeyRing returns the GPG keyring for the given gpgkey file.
func OpenKeyRing(path string) (openpgp.KeyRing, error) {
	// check gpgkey is specified
	if path == "" {
		return nil, fmt.Errorf("gpgkey not specified")
	}

	// trim file:// prefix
	if strings.HasPrefix(strings.ToLower(path), "file://") {
		path = path[7:]
	}

	keyring, err := rpm.KeyRingFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error reading GPG key: %v", err)
	}

	return keyring, nil
}
