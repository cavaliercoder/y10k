package compress

import (
	"github.com/dsnet/compress/bzip2"
	"io"
)

func bzip2Compress(w io.Writer, r io.Reader) (int64, error) {
	conf := &bzip2.WriterConfig{
		Level: bzip2.BestCompression,
	}

	compress, err := bzip2.NewWriter(w, conf)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(compress, r)

	// compressor must be explicitely closed or the result is an empty file
	if err := compress.Close(); err != nil {
		return n, err
	}

	return n, err
}

func NewBzip2Compressor() Compressor {
	return CompressorFunc(bzip2Compress)
}
