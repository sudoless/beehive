package beehive_responder

import (
	"encoding/json"

	"github.com/sudoless/beehive/pkg/beehive"
)

// JSON implements the beehive.Responder interface by calling json.Marshal on the given Object when Body is first called.
// Any subsequent calls to Body will return the same data.
type JSON struct {
	Object any
	Code   int

	data []byte
}

func (j *JSON) StatusCode(_ *beehive.Context) int {
	return j.Code
}

func (j *JSON) Body(ctx *beehive.Context) []byte {
	if j.data != nil {
		return j.data
	}

	h := ctx.ResponseWriter.Header()
	h.Set("Content-Type", "application/json")

	data, err := json.Marshal(j.Object)
	if err != nil {
		panic(err)
	}

	j.data = data
	return data
}
