# BeeHive 🐝🐝🐝

BeeHive is a highly opinionated performant HTTP router with a
series of middleware and utilities for production ready robust systems.

Less is more. As such the router has 0 dependencies and all middleware
in the main `beehive` package only use core features and interfaces where
other packages can be used up to the users' discretion.

[![Go Report Card](https://goreportcard.com/badge/go.sdls.io/beehive)](https://goreportcard.com/report/go.sdls.io/beehive)

## Features

- 🍯 Sweet and _simple_
- [0 dependencies](go.mod)
- 0 allocation route lookup, and one small allocation per request: its `Context`, which is never reused, so it
  stays a valid `context.Context` for anything still holding it after the request
- Route grouping, prefixing
- Wildcard matching
- Middleware and handler chaining
- _Fast and performant_

## Usage

Fetch it using the latest Golang preferred way.

### Example

To see side by side comparison of BeeHive and other Go routers, check out the individual "Rosetta" docs:
- [Chi Rosetta](docs/rosetta_chi.md), and the original [project](https://github.com/go-chi/chi)
- [Echo Rosetta](docs/rosetta_echo.md), and the original [project](https://github.com/labstack/echo)
- [Gin Rosetta](docs/rosetta_gin.md), and the original [project](https://github.com/gin-gonic/gin)

For some classic examples:

```go
// better example coming soon™
```

## Version

Everything until v1.0.0 will be considered unstable and the API may introduce breaking changes in any minor or patch
release. The v1.0.0 release will be considered stable and the API will not introduce any breaking changes. Any new
features will be introduced on a need only basis.

Any addition that can be implemented outside the package should be.

Any addition that uses dependencies must be implemented in a separate package.


## Benchmarks

`go test -run='^$' -bench=. -benchmem -count=10 ./...`, summarised with `benchstat`, on linux/arm64 with 6 cores and
Go 1.27:

| Benchmark | time/op | B/op | allocs/op |
|---|---|---|---|
| `Radix_Add` | 2.11µs ± 7% | 3.41KiB | 68 |
| `Radix_Get` | 195ns ± 3% | 0 | 0 |
| `Radix_wildcard_Get` | 14.3ns ± 2% | 0 | 0 |
| `Router_ServeHTTP/plain` | 55.7ns ± 2% | 48 | 1 |
| `Router_ServeHTTP/plain_parallel` | 36.5ns ± 8% | 48 | 1 |
| `Router_ServeHTTP/with_after_callback` | 65.0ns ± 5% | 48 | 1 |
| `Router_ServeHTTP/with_after_callback_parallel` | 51.0ns ± 8% | 48 | 1 |
| `Router_ServeHTTP/middleware_chain` | 96.6ns ± 7% | 96 | 2 |
| `Router_ServeHTTP/middleware_chain_parallel` | 70.0ns ± 6% | 96 | 2 |
| `_ValuesParser/beehive` | 77.0ns ± 1% | 0 | 0 |
| `_ValuesParser/net/url` | 270ns ± 4% | 448 | 5 |

The one allocation of every `Router_ServeHTTP` case is the request's 48-byte `Context`; the middleware chain's second
is the `context.WithValue` one of its middleware makes.
