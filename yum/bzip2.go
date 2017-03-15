package yum

import (
	"github.com/dsnet/compress/bzip2"
	"io"
)

func Bzip2Compress(w io.Writer, r io.Reader) error {
	conf := &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	}

	compress, err := bzip2.NewWriter(w, conf)
	if err != nil {
		return err
	}

	_, err = io.Copy(compress, r)
	return err
}
