package node

import "errors"

var (
	// ErrNodeNotFound is returned when the internal router lookup cannot walk the given path.
	ErrNodeNotFound = errors.New("node not found")

	// ErrNodeAlreadyExists is returned when trying to re-define an existing node.
	ErrNodeAlreadyExists = errors.New("node already exists")
)
