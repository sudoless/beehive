package beehive_responder

import (
	"encoding/json"

	"go.sdls.io/beehive/pkg/beehive"
)

// JSON implements the beehive.Responder interface by calling json.Marshal on the given Object when Body is first
// called. Any subsequent calls to Body will return the same data.
type JSON struct {
	Object any
	Code   int

	data []byte
}

// test that JSON implements the beehive.Responder interface.
var _ beehive.Responder = &JSON{}

func (j *JSON) StatusCode(_ *beehive.Context) int {
	return j.Code
}

func (j *JSON) Respond(ctx *beehive.Context) {
	w := ctx.ResponseWriter
	w.WriteHeader(j.Code)

	if j.data != nil {
		_, _ = w.Write(j.data)
		return
	}

	h := ctx.ResponseWriter.Header()
	h.Set("Content-Type", "application/json")

	data, err := json.Marshal(j.Object)
	if err != nil {
		panic(err)
	}

	j.data = data
	_, _ = w.Write(data)
}
