package unsafe

import "unsafe"

func StringToBytes(s string) []byte {
	// #nosec G103
	return *(*[]byte)(unsafe.Pointer(
		&struct {
			string
			Cap int
		}{s, len(s)},
	))
}

func BytesToString(b []byte) string {
	// #nosec G103
	return *(*string)(unsafe.Pointer(&b))
}
