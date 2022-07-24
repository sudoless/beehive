package trie

import (
	"bytes"

	"go.sdls.io/beehive/internal/unsafe"
)

type radixNode struct {
	data       any
	path       []byte
	pathFull   []byte
	children   []*radixNode
	lookup     []byte
	isWildcard bool
	wildcard   *radixNode
}

func (node *radixNode) propagateWildcard(wildcard *radixNode) {
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

func (node *radixNode) add(path []byte, data any) {
	current := node
	isWildcard := path[len(path)-1] == '*'
	if isWildcard {
		path = path[:len(path)-1]
	}
	pathFull := path[:]
	commonIdx := commonPrefix(path, current.path)

	for ; commonIdx == len(current.path); commonIdx = commonPrefix(path, current.path) {
		if commonIdx == len(path) {
			current.data = data
			current.pathFull = pathFull
			if isWildcard && !current.isWildcard {
				current.isWildcard = isWildcard
				current.propagateWildcard(nil)
			}
			return
		}

		path = path[commonIdx:]
		lookupIdx := bytes.IndexByte(current.lookup, path[0])
		if lookupIdx == -1 {
			current.lookup = append(current.lookup, path[0])
			child := &radixNode{
				path:       path,
				pathFull:   pathFull,
				data:       data,
				isWildcard: isWildcard,
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

		current = current.children[lookupIdx]
	}

	self := &radixNode{}
	*self = *current
	self.path = current.path[commonIdx:]

	current.pathFull = nil
	current.path = current.path[:commonIdx]
	if len(current.path) == 0 {
		current.path = nil
	}
	current.data = nil
	current.isWildcard = false

	if commonIdx == len(path) {
		current.children = []*radixNode{self}
		current.lookup = []byte{self.path[0]}
		current.data = data
		current.pathFull = pathFull
		current.isWildcard = isWildcard
	} else {
		child := &radixNode{
			data:       data,
			path:       path[commonIdx:],
			pathFull:   pathFull,
			isWildcard: isWildcard,
			wildcard:   current.wildcard,
		}

		current.lookup = []byte{self.path[0], path[commonIdx]}
		current.children = []*radixNode{self, child}
	}

	if self.isWildcard {
		current.propagateWildcard(nil)
	}
}

func (node *radixNode) get(path []byte) (any, bool) {
	if len(path) == 0 || node == nil {
		return nil, false
	}

	current := node
	ptr := commonPrefix(path, current.path)
	pathLen := len(path)

	for {
		if ptr >= pathLen {
			if bytes.Equal(path, current.pathFull) {
				return current.data, true
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

		return wildcard.data, true
	}

	return nil, false
}

func (node *radixNode) leafs() map[string]any {
	m := make(map[string]any)

	if len(node.children) != 0 {
		for _, child := range node.children {
			innerM := child.leafs()
			for k, v := range innerM {
				m[k] = v
			}
		}
	}

	if node.data != nil {
		m[string(node.pathFull)] = node.data
	}

	return m
}

type Radix struct {
	root *radixNode
}

func (radix *Radix) Add(path string, data any) {
	if len(path) == 0 {
		return
	}

	isWildcard := path[len(path)-1] == '*'
	if radix.root == nil {
		if isWildcard {
			path = path[:len(path)-1]
		}

		radix.root = &radixNode{
			path:       []byte(path),
			pathFull:   []byte(path),
			data:       data,
			isWildcard: isWildcard,
		}

		return
	}

	radix.root.add([]byte(path)[:], data)
}

func (radix Radix) Get(path string) (any, bool) {
	return radix.root.get(unsafe.StringToBytes(path))
}
