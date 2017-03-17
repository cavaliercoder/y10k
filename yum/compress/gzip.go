package compress

import (
	"compress/gzip"
	"io"
)

func gzipCompress(w io.Writer, r io.Reader) (int64, error) {
	compress := gzip.NewWriter(w)
	return io.Copy(compress, r)
}

func NewGzipCompressor() Compressor {
	return CompressorFunc(gzipCompress)
}
