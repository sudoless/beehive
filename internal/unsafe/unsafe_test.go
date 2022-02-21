package unsafe

import (
	"testing"
)

func Test_string_to_bytes_to_string(t *testing.T) {
	myPath := "/foo/bar/baz/fiz"
	myPathBytes := StringToBytes(myPath)

	if len(myPath) != len(myPathBytes) {
		t.Fatalf("stringToBytes(%s) returned %d bytes, expected %d", myPath, len(myPathBytes), len(myPath))
	}

	segmentString := myPath[1:4]
	if segmentString != "foo" {
		t.Fatalf("expected 'foo' got '%s'", segmentString)
	}

	segmentUnsafe := BytesToString(myPathBytes[1:4])
	if segmentUnsafe != "foo" {
		t.Fatalf("expected 'foo' got '%s'", segmentUnsafe)
	}
}
