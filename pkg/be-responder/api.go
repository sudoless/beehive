package be_responder

import (
	"context"
	"encoding/json"
	"net/http"
)

type Api struct {
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Job     string      `json:"job"`
	Code    int         `json:"status_code,omitempty"`
	Success bool        `json:"success"`

	NoCookies
}

func (a *Api) Headers(_ context.Context, _ *http.Request, h http.Header) {
	h["Content-Type"] = []string{"application/json; charset=utf-8"}
	h["X-Content-Type-Options"] = []string{"nosniff"}
	h["Cache-Control"] = []string{"no-cache"}
	h["Xq-Job"] = []string{a.Job}
}

func (a *Api) StatusCode(_ context.Context, _ *http.Request) int {
	return a.Code
}

func (a *Api) Body(_ context.Context, _ *http.Request) []byte {
	data, err := json.Marshal(a)
	if err != nil {
		a.Success = false
		a.Code = http.StatusInternalServerError

		return nil
	}

	return data
}
