package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

type Checksummer interface {
	Checksum(r io.Reader) (string, error)
	ChecksumFile(src string) (string, error)
}

type checksummer struct {
	h hash.Hash
}

func (c *checksummer) Checksum(r io.Reader) (string, error) {
	c.h.Reset()

	if _, err := io.Copy(c.h, r); err != nil {
		return "", err
	}

	checksum := hex.EncodeToString(c.h.Sum(nil))
	return checksum, nil
}

func (c *checksummer) ChecksumFile(src string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()

	return c.Checksum(f)
}

func NewSha256() Checksummer {
	return &checksummer{sha256.New()}
}
