# BeeHive 🐝🐝🐝

**Be**eHive is a highly opinionated performant HTTP router with a
series of middleware and utilities for production ready robust systems.

Less is more. As such the router has 0 dependencies and all middleware
in the main `beehive` package only use core features and interfaces where
other packages can be used up to the users' discretion.

## Features

- 🍯 Sweet and _simple_
- 0 dependencies
- 0 memory allocation routing
- Route grouping, prefixing
- Wildcard matching
- Middleware and handler chaining
- _Fast and performant_

## Usage

Fetch it using the latest Golang preferred way.

### Example

```go
// better example coming soon™
```

## Version

Everything until v1.0.0 will be considered unstable and the API may introduce breaking changes in any minor or patch
release. The v1.0.0 release will be considered stable and the API will not introduce any breaking changes. Any new
features will be introduced on a need only basis.

Any addition that can be implemented outside the package should be.

Any addition that uses dependencies must be implemented in a separate package.
