package compress

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
)

var EmptyFileErr = errors.New("destination file is empty")

// Compressor compresses a stream.
type Compressor interface {
	Compress(w io.Writer, r io.Reader) (int64, error)
	CompressToTemp(src string) (string, error)
}

// A CompressorFunc reads data from an stream, compresses it and writes it to
// another stream.
type CompressorFunc func(io.Writer, io.Reader) (int64, error)

func (fn CompressorFunc) Compress(w io.Writer, r io.Reader) (int64, error) {
	return fn(w, r)
}

// CompressToTemp compresses the given file to a temporary file and returns the
// path of the temporary file or an error.
func (fn CompressorFunc) CompressToTemp(src string) (string, error) {
	// open source file for reading
	if f, err := os.Open(src); err != nil {
		return "", err
	} else {
		defer f.Close()

		// open temp file for writing
		if tmp, err := ioutil.TempFile("", "yum-"); err != nil {
			return "", err
		} else {
			defer tmp.Close()

			// compress
			if n, err := fn(tmp, f); err != nil {
				return "", err
			} else if n == 0 {
				return "", EmptyFileErr
			}

			return tmp.Name(), nil
		}
	}
}
