package mail

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestBase64WrapBoundaries(t *testing.T) {
	for _, n := range []int{0, 1, 2, 3, 56, 57, 58, 114, 115, 65536} {
		data := bytes.Repeat([]byte("x"), n)
		got := string(base64Wrap(data))
		decoded, err := base64.StdEncoding.DecodeString(got)
		if err != nil || !bytes.Equal(decoded, data) {
			t.Fatalf("size %d: %v", n, err)
		}
		if n > 0 && !strings.HasSuffix(got, "\r\n") {
			t.Fatalf("size %d: missing newline", n)
		}
		for _, line := range strings.Split(got, "\r\n") {
			if len(line) > 76 {
				t.Fatalf("size %d: line length %d", n, len(line))
			}
		}
	}
}

var wrappedResult []byte

func BenchmarkBase64Attachment(b *testing.B) {
	data := bytes.Repeat([]byte("x"), 65536)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		wrappedResult = base64Wrap(data)
	}
}
