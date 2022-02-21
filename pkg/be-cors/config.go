package be_cors

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sudoless/beehive/pkg/be"
	ws_responder "github.com/sudoless/beehive/pkg/be-responder"
)

type Config struct {
	AllowHosts []string

	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           time.Duration
}

func (c *Config) Apply(group be.Grouper) be.Grouper {
	group.Handle(http.MethodOptions, "*", c.HandlerFunc(true))
	return group.Group("", c.HandlerFunc(false))
}

func (c *Config) Allow(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	host := u.Hostname()

	for _, h := range c.AllowHosts {
		if h == host {
			return true
		}
	}

	return false
}

func (c *Config) HandlerFunc(preFlight bool) be.HandlerFunc {
	var responderAllow be.Responder
	if preFlight {
		responderAllow = &ws_responder.Status{Code: http.StatusNoContent}
	}

	responderForbidden := &be.DefaultResponder{
		Message: []byte("cors forbidden"),
		Status:  http.StatusForbidden,
	}

	headerAllowMethods := []string{strings.Join(c.AllowMethods, ", ")}
	headerAllowHeaders := []string{strings.Join(c.AllowHeaders, ", ")}

	var headerMaxAge []string
	if c.MaxAge > 0 {
		headerMaxAge = []string{strconv.Itoa(int(c.MaxAge.Seconds()))}
	}

	var headerAllowCredentials []string
	if c.AllowCredentials {
		headerAllowCredentials = []string{"true"}
	}

	return func(ctx context.Context, r *http.Request) be.Responder {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return nil
		}
		if !c.Allow(origin) {
			return responderForbidden
		}

		w := be.ResponseWriter(ctx)
		h := w.Header()

		h.Add("Vary", "Origin")

		h["Access-Control-Allow-Origin"] = []string{origin}
		h["Access-Control-Allow-Methods"] = headerAllowMethods
		h["Access-Control-Allow-Headers"] = headerAllowHeaders

		if len(headerMaxAge) > 0 {
			h["Access-Control-Max-Age"] = headerMaxAge
		}
		if len(headerAllowCredentials) > 0 {
			h["Access-Control-Allow-Credentials"] = headerAllowCredentials
		}

		return responderAllow
	}
}
