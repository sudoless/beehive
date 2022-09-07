# BeeHive 🐝🐝🐝

BeeHive is a highly opinionated performant HTTP router with a
series of middleware and utilities for production ready robust systems.

Less is more. As such the router has 0 dependencies and all middleware
in the main `beehive` package only use core features and interfaces where
other packages can be used up to the users' discretion.

## Features

- 🍯 Sweet and _simple_
- [0 dependencies](go.mod)
- 0 memory allocation routing
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

```
name                   time/op

pkg:go.sdls.io/beehive/internal/trie goos:darwin goarch:arm64
Radix_Add-10           1.70µs ± 1%
Radix_Get-10            200ns ± 2%
Radix_wildcard_Get-10  13.7ns ± 1%

pkg:go.sdls.io/beehive/pkg/beehive goos:darwin goarch:arm64
Router_ServeHTTP-10    33.6ns ± 1%

pkg:go.sdls.io/beehive/pkg/beehive-query goos:darwin goarch:arm64
_ValuesParser-10        156ns ± 3%
```

```
name                   alloc/op

pkg:go.sdls.io/beehive/internal/trie goos:darwin goarch:arm64
Radix_Add-10           3.50kB ± 0%
Radix_Get-10            0.00B     
Radix_wildcard_Get-10   0.00B
     
pkg:go.sdls.io/beehive/pkg/beehive goos:darwin goarch:arm64
Router_ServeHTTP-10     0.00B
     
pkg:go.sdls.io/beehive/pkg/beehive-query goos:darwin goarch:arm64
_ValuesParser-10        48.0B ± 0%
```

```
name                   allocs/op

pkg:go.sdls.io/beehive/internal/trie goos:darwin goarch:arm64
Radix_Add-10             68.0 ± 0%
Radix_Get-10             0.00     
Radix_wildcard_Get-10    0.00
     
pkg:go.sdls.io/beehive/pkg/beehive goos:darwin goarch:arm64
Router_ServeHTTP-10      0.00
     
pkg:go.sdls.io/beehive/pkg/beehive-query goos:darwin goarch:arm64
_ValuesParser-10         1.00 ± 0%
```
