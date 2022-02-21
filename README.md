# BeeHive 🐝🐝🐝

**Be**eHive is a highly opinionated performant HTTP router with a
series of middleware and utilities for production ready robust systems.

Less is more. As such the router has 0 dependencies and all middleware
in the main `beehive` package only use core features and interfaces where
other packages can be used up to the users' discretion.

## Features

- 0 dependencies
- 0 memory allocation routing
- Route grouping, prefixing
- Wildcard matching
- Middleware and handler chaining
- Fast and performant

## Usage

Fetch it using the latest Golang preferred way.

### Example

```go
package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sudoless/beehive/pkg/be"
	beCors "github.com/sudoless/beehive/pkg/be-cors"
)

func handleEcho(_ context.Context, r *http.Request) be.Responder {
	path := r.URL.Path

	return &be.DefaultResponder{
		Message: []byte(r.Method + " " + path),
		Status:  http.StatusOK,
	}
}

func handleLog(ctx context.Context, r *http.Request) be.Responder {
	start := time.Now()

	res := be.Next(ctx, r)
	if res == nil {
		return nil
	}

	log.Printf("[%d] %s: %s | %s",
		res.StatusCode(ctx, r),
		r.Method,
		r.URL.String(),
		time.Since(start).String())

	return res
}

func main() {
	router := be.NewDefaultRouter()

	corsConfig := &beCors.Config{
		AllowHosts:       []string{"dashboard.example.com"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Foo"},
		AllowCredentials: true,
		MaxAge:           0,
	}

	api := router.Group("/api", handleLog)
	api.Handle("GET", "/foo/bar", handleEcho)

	corsGroup := corsConfig.Apply(api.Group("/cors"))
	{
		corsGroup.
			Handle("GET", "/foo", handleEcho).
			HandleAny([]string{"GET", "POST"}, "/foo/bar", handleEcho).
			Handle("GET", "/fiz", handleEcho).
			Handle("GET", "/", handleEcho)

		corsGroup.Handle("GET", "/abc", handleEcho)
		corsGroup.Handle("PUT", "/abc", handleEcho)
	}

	router.Handle("GET", "/routes", func(_ context.Context, _ *http.Request) be.Responder {
		routes := strings.Join(router.DebugRoutes(), "\n")
		return &be.DefaultResponder{
			Message: []byte(routes),
			Status:  http.StatusOK,
		}
	})

	server := http.Server{
		Addr:           "127.0.0.1:8888",
		Handler:        router,
		ReadTimeout:    time.Second * 5,
		IdleTimeout:    time.Second * 10,
		MaxHeaderBytes: 1 << 16,
	}

	log.Println("starting server")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}
```

## Version

Everything until v1.0.0 will be considered unstable and the API may introduce breaking changes in any minor or patch
release. The v1.0.0 release will be considered stable and the API will not introduce any breaking changes. Any new
features will be introduced on a need only basis.

Any addition that can be implemented outside the package should be.

Any addition that uses dependencies must be implemented in a separate package.
