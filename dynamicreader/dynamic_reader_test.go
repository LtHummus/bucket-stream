package dynamicreader

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestDynamicReader(t *testing.T) {
	idx := 0
	dr := NewDynamicReader(func() (io.ReadCloser, error) {
		if idx == 0 {
			idx++
			return io.NopCloser(strings.NewReader("a")), nil
		} else if idx == 1 {
			idx++
			return io.NopCloser(strings.NewReader("b")), nil
		} else {
			return nil, nil
		}
	})

	var b bytes.Buffer
	_, err := io.Copy(&b, dr)
	if err != nil {
		t.Fatalf("error on io.Copy: %s", err)
	}

	if res := b.String(); res != "ab" {
		t.Fatalf("did not get expected result. expected %s, got %s", "ab", res)
	}
}
