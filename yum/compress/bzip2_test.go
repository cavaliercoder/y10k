package compress

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestBzip2Compress(t *testing.T) {
	// generate 4MB random data
	in := make([]byte, 4194304)
	inlen, _ := rand.Read(in)

	// compress it to a byte buffer
	buf := &bytes.Buffer{}
	bzip2Compress(buf, bytes.NewReader(in))
	out := buf.Bytes()
	outlen := len(out)

	// check
	if outlen == 0 || outlen >= inlen {
		t.Errorf("unexpected compressed size: %d", outlen)
	}

	t.Logf("%d bytes compressed down to %d bytes (%d%%)", inlen, outlen, 100-int(float64(outlen)/float64(inlen)*100))
}
