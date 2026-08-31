# CLAUDE.md

## Commands

```sh
go test -race ./...                      # full suite
go test -bench=. -benchmem ./...         # benchmarks
make lint                                # golangci-lint, config in devz/
make lint-src                            # same, skipping tests
```

The Makefile is imported boilerplate; only the targets below the `CUSTOM` line belong to this
project. There is no `make test`, `make fmt` or `make dev-deps`.

## The suite is intentionally red

Failing tests record confirmed defects (SUW-81). **Never delete, skip or weaken one to get a green
run** — each asserts behaviour that was decided on, so making it pass means fixing the production
code. Delete this section once none are left.

## Invariants

- **`Radix.Get` and `Router.ServeHTTP` allocate nothing.** `TestRadix_Get_0alloc` and
  `TestRouter_ServeHTTP_0alloc` enforce it. Check them before touching the request path.
- `pkg/beehive` depends on nothing but the standard library, and no `pkg/beehive-*` add-on may
  become a dependency of it.
- A `Responder`'s `StatusCode` reports what `Respond` wrote. The two disagreeing is a bug.

## Conventions

Tests sit beside their subject (`foo.go` → `foo_test.go`), are table-driven with `t.Run`, and call
`t.Parallel()` unless they bind a port, share a counter, or measure allocations.
