package compress

import (
	"bytes"
	"crypto/sha256"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"testing"
)

func TestCompressToTemp(t *testing.T) {
	var src string
	var sum1 [32]byte
	var sum2 [32]byte

	// generate random file
	p := make([]byte, 4194304)
	rand.Read(p)
	sum1 = sha256.Sum256(p)
	if f, err := ioutil.TempFile("", "yum-test-"); err != nil {
		panic(err)
	} else {
		defer f.Close()

		src = f.Name()
		if _, err := f.Write(p); err != nil {
			panic(err)
		}
	}

	// "compress" file
	cmp := CompressorFunc(io.Copy)
	dst, err := cmp.CompressToTemp(src)
	if err != nil {
		panic(err)
	}

	// inspect output
	if f, err := os.Open(dst); err != nil {
		panic(err)
	} else {
		defer f.Close()

		if fi, err := f.Stat(); err != nil {
			panic(err)
		} else {
			if fi.Size() == 0 {
				t.Errorf("compressed file is zero length")
			}
		}

		if out, err := ioutil.ReadAll(f); err != nil {
			panic(err)
		} else {
			sum2 = sha256.Sum256(out)
		}
	}

	// compare
	if 0 != bytes.Compare(sum1[:], sum2[:]) {
		t.Errorf("checksum fail")
	}

	os.Remove(src)
	os.Remove(dst)
}
