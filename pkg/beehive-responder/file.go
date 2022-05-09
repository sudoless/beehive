package beehive_responder

import (
	"context"
	"net/http"
	"strconv"
)

type File struct {
	NoCookies

	Name string
	Data []byte
	Code int
}

func (f *File) Headers(_ context.Context, _ *http.Request, h http.Header) {
	h["Content-Description"] = []string{"File Transfer"}
	h["Content-Transfer-Encoding"] = []string{"binary"}
	h["Content-Disposition"] = []string{`attachment; filename="` + f.Name + `"`}
	h["Content-Type"] = []string{"application/octet-stream"}
	h["X-Filename"] = []string{f.Name}
	h["X-Filesize"] = []string{strconv.Itoa(len(f.Data))}
	h["Cache-Control"] = []string{"no-cache"}
}

func (f *File) StatusCode(_ context.Context, _ *http.Request) int {
	if f.Code == 0 {
		return http.StatusOK
	}
	return f.Code
}

func (f *File) Body(_ context.Context, _ *http.Request) []byte {
	return f.Data
}
