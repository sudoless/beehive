package trie

import (
	"bytes"

	"go.sdls.io/beehive/internal/unsafe"
)

type radixNode[T any] struct {
	data        T
	dataIsValid bool
	path        []byte
	pathFull    []byte
	children    []*radixNode[T]
	lookup      []byte
	isWildcard  bool
	wildcard    *radixNode[T]
}

func (node *radixNode[T]) propagateWildcard(wildcard *radixNode[T]) {
	if wildcard != nil {
		node.wildcard = wildcard
	}

	if node.isWildcard {
		wildcard = node
	}

	for _, child := range node.children {
		child.propagateWildcard(wildcard)
	}
}

func (node *radixNode[T]) add(path []byte, data T) {
	current := node
	isWildcard := path[len(path)-1] == '*'
	if isWildcard {
		path = path[:len(path)-1]
	}
	pathFull := path[:] //nolint:gocritic
	commonIdx := commonPrefix(path, current.path)

	for ; commonIdx == len(current.path); commonIdx = commonPrefix(path, current.path) {
		if commonIdx == len(path) {
			current.data = data
			current.dataIsValid = true
			current.pathFull = pathFull
			if isWildcard && !current.isWildcard {
				current.isWildcard = isWildcard
				current.propagateWildcard(nil)
			}
			return
		}

		path = path[commonIdx:]
		lookupIdx := bytes.IndexByte(current.lookup, path[0])
		if lookupIdx != -1 {
			current = current.children[lookupIdx]
			continue
		}

		current.lookup = append(current.lookup, path[0])
		child := &radixNode[T]{
			path:        path,
			pathFull:    pathFull,
			data:        data,
			dataIsValid: true,
			isWildcard:  isWildcard,
		}
		current.children = append(current.children, child)

		if !isWildcard {
			if current.isWildcard {
				child.wildcard = current
			} else {
				child.wildcard = current.wildcard
			}
		} else {
			child.wildcard = current
		}
		return
	}

	self := &radixNode[T]{}
	*self = *current
	self.path = current.path[commonIdx:]

	current.pathFull = nil
	current.path = current.path[:commonIdx]
	if len(current.path) == 0 {
		current.path = nil
	}
	current.dataIsValid = false
	current.isWildcard = false

	if commonIdx == len(path) {
		current.children = []*radixNode[T]{self}
		current.lookup = []byte{self.path[0]}
		current.data = data
		current.dataIsValid = true
		current.pathFull = pathFull
		current.isWildcard = isWildcard
	} else {
		child := &radixNode[T]{
			data:        data,
			dataIsValid: true,
			path:        path[commonIdx:],
			pathFull:    pathFull,
			isWildcard:  isWildcard,
			wildcard:    current.wildcard,
		}

		current.lookup = []byte{self.path[0], path[commonIdx]}
		current.children = []*radixNode[T]{self, child}
	}

	if self.isWildcard {
		current.propagateWildcard(nil)
	}
}

func (node *radixNode[T]) get(path []byte) (T, bool) {
	var zero T

	if len(path) == 0 || node == nil {
		return zero, false
	}

	current := node
	ptr := commonPrefix(path, current.path)
	pathLen := len(path)

	for {
		if ptr >= pathLen {
			if bytes.Equal(path, current.pathFull) {
				return current.data, current.dataIsValid
			}

			break
		}

		lookupIdx := -1
		for idx := range current.lookup {
			if current.lookup[idx] == path[ptr] {
				lookupIdx = idx
				break
			}
		}

		if lookupIdx == -1 {
			break
		}

		current = current.children[lookupIdx]
		ptr += len(current.path)
	}

	wildcard := current.wildcard
	if current.isWildcard {
		wildcard = current
	}

	for wildcard != nil {
		if len(wildcard.pathFull) > pathLen {
			wildcard = wildcard.wildcard
			continue
		}

		if !bytes.Equal(wildcard.pathFull, path[:len(wildcard.pathFull)]) {
			wildcard = wildcard.wildcard
			continue
		}

		return wildcard.data, wildcard.dataIsValid
	}

	return zero, false
}

func (node *radixNode[T]) leafs() map[string]T {
	m := make(map[string]T)

	if len(node.children) != 0 {
		for _, child := range node.children {
			innerM := child.leafs()
			for k, v := range innerM {
				m[k] = v
			}
		}
	}

	if node.dataIsValid {
		m[string(node.pathFull)] = node.data
	}

	return m
}

type Radix[T any] struct {
	root *radixNode[T]
}

func (radix *Radix[T]) Add(path string, data T) {
	if path == "" {
		return
	}

	isWildcard := path[len(path)-1] == '*'
	if radix.root == nil {
		if isWildcard {
			path = path[:len(path)-1]
		}

		radix.root = &radixNode[T]{
			path:        []byte(path),
			pathFull:    []byte(path),
			data:        data,
			dataIsValid: true,
			isWildcard:  isWildcard,
		}

		return
	}

	radix.root.add([]byte(path), data)
}

func (radix Radix[T]) Get(path string) (data T, found bool) {
	return radix.root.get(unsafe.StringToBytes(path))
}
