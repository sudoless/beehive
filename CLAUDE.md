# CLAUDE.md

`go.sdls.io/beehive` — a zero-allocation HTTP router. Go 1.27.

## Commands

```sh
go test -race ./...                      # full suite
go test -race -run TestName ./pkg/...    # one test
go test -bench=. -benchmem ./...         # benchmarks
make lint                                # golangci-lint, config in devz/
make lint-src                            # same, skipping tests
```

The Makefile is mostly imported boilerplate (`info`, `tag-*`, `mk-update`); only the targets under
the `CUSTOM` line are this project's. There is no `make test`, `make fmt` or `make dev-deps`.

## The suite is intentionally red

`develop` carries failing tests that record confirmed defects (SUW-81). **Do not delete, skip, or
weaken them to get a green run.** Each asserts the behaviour the maintainer decided on; making one
pass means fixing the production code. `go test ./... 2>&1 | grep FAIL` shows the current set.

## Architecture

Request flow: `Router.ServeHTTP` → per-method `Radix.Get(r.URL.Path)` → `[]HandlerFunc` chain →
first non-nil `Responder` wins and its `Respond` writes the response.

**`HandlerFunc`** (`pkg/beehive/handler.go`) — `func(*Context) Responder`. Handlers and middleware
share this signature. Return `nil` to continue the chain, a `Responder` to short-circuit. `ctx.Next()`
runs the rest of the chain inline, so middleware can wrap it.

**`Responder`** (`pkg/beehive/responder.go`) — `Respond(*Context)` writes, `StatusCode(*Context)`
reports what was written *afterwards*. The two must agree. Implementations live in
`pkg/beehive-responder/`.

**`Context`** (`pkg/beehive/context.go`) — pooled via `sync.Pool`, wraps the `http.ResponseWriter`,
`*http.Request` and a `context.Context`, and implements `context.Context` itself. `ctx.After(f)`
registers cleanup that runs after the chain but **before** `Router.After`.

**Router hooks** — `Context` (per-request context factory), `WhenNotFound`, `Recover`, `After`.
`NewRouter` installs the first three.

**Trie** (`internal/trie/`) — generic radix tree, one `Radix[[]HandlerFunc]` per method. A path
ending in `*` registers a wildcard prefix; each node carries a `.wildcard` pointer to its nearest
ancestor wildcard, so `propagateWildcard` must run after any structural change that introduces one
(the split case in `add`). Exact matches beat wildcards.

**Groups** (`pkg/beehive/group.go`) — `Grouper` accumulates a path prefix and middleware without
touching the trie; only `Handle`/`HandleAny` reach it. `Router` implements `Grouper`, so it is the
root of every chain. Note `internal/trie/group.go` is a separate, unused prototype.

## Invariants

- **`Radix.Get` and `Router.ServeHTTP` allocate nothing.** Enforced by `TestRadix_Get_0alloc` and
  `TestRouter_ServeHTTP_0alloc`. Check these before touching the request path.
- `internal/unsafe.StringToBytes` gives a zero-copy view of a string — read-only, never mutate it.
- `pkg/beehive-*` packages are independent add-ons; none may become a dependency of `pkg/beehive`.

## Conventions

Tests live beside their subject (`foo.go` → `foo_test.go`), are table-driven with `t.Run`, and call
`t.Parallel()` unless they touch a port, a shared counter, or `AllocsPerRun`.
