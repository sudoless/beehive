# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
# Run all tests (requires gotestsum, uses race detector)
make test

# Run a single test
go test -race -run TestName ./path/to/pkg/...

# Run benchmarks
make bench
# or directly:
go test -bench=. -benchmem -benchtime=10s ./...

# Lint
make lint
# (runs golangci-lint)

# Format (uses gofumpt, stricter than gofmt)
make fmt

# Install dev dependencies (golangci-lint, gofumpt, gotestsum, etc.)
make dev-deps
```

## Architecture

### Core Data Flow
HTTP request → `Router.ServeHTTP` → looks up `radix.Get(r.URL.Path)` per-method trie → calls `[]HandlerFunc` chain → each returns a `Responder` or nil (continue).

### Key Types
- **`HandlerFunc`** (`pkg/beehive/handler.go`): `func(ctx *Context) Responder` — both handlers and middleware use this signature. Return `nil` to continue the chain, return a `Responder` to short-circuit.
- **`Responder`** (`pkg/beehive/responder.go`): interface with `Respond(*Context)`. Concrete types (JSON, status codes, etc.) live in `pkg/beehive-responder/`.
- **`Context`** (`pkg/beehive/context.go`): per-request struct pooled via `sync.Pool`, wraps `http.ResponseWriter`, `*http.Request`, and `context.Context`.

### Routing / Trie (`internal/trie/`)
- Generic radix (compressed prefix) trie: `Radix[T]` stores any `T` per route.
- One `Radix[[]HandlerFunc]` per HTTP method in `Router.methods`.
- `Radix.Add(path, data)` — path ending in `*` is a wildcard prefix match.
- `Radix.Get(path)` — zero allocations; uses `unsafe.StringToBytes` to avoid string→[]byte copy.
- Wildcard fallback: each node carries a `.wildcard` pointer to the nearest ancestor wildcard. `propagateWildcard` must be called after any structural change that introduces a new wildcard node (split case in `add()`).

### Route Groups (`pkg/beehive/group.go`, `internal/trie/group.go`)
- `Router` implements `Grouper[HandlerFunc]`.
- `virtualGroup` accumulates prefix + middleware without touching the trie; only `Finalize` calls `trie.Add`.
- `trieGroup` is the root grouper that owns the trie pointer (`*Radix[[]T]`).

### Middleware Packages (`pkg/beehive-*/`)
Each sub-package is independent and provides middleware/utilities:
- `beehive-cors` — CORS headers
- `beehive-query` — zero-alloc query string parsing
- `beehive-rate` — rate limiting
- `beehive-responder` — common responders (JSON, redirect, etc.)
- `beehive-auth`, `beehive-proxy`, `beehive-pprof` — auth, reverse proxy, pprof

### Performance Invariants
- `Radix.Get` and `Router.ServeHTTP` must remain **zero-allocation**. The `TestRadix_Get_0alloc` and router-level alloc tests enforce this.
- The `internal/unsafe` package provides `StringToBytes` for zero-copy string-to-slice conversion — use it only for read-only byte slice needs.
