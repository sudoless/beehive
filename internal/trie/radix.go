package trie

import (
	"bytes"

	"go.sdls.io/beehive/internal/unsafe"
)

type radixNode struct {
	data     any
	path     []byte
	pathFull []byte
	children []*radixNode
	lookup   []byte
}

func (node *radixNode) add(path []byte, data any) {
	pathFull := path[:]
	current := node
	commonIdx := commonPrefix(path, current.path)

	for ; commonIdx == len(current.path); commonIdx = commonPrefix(path, current.path) {
		if commonIdx == len(path) {
			current.data = data
			current.pathFull = pathFull
			return
		}

		path = path[commonIdx:]
		lookupIdx := bytes.IndexByte(current.lookup, path[0])
		if lookupIdx == -1 {
			current.lookup = append(current.lookup, path[0])
			current.children = append(current.children, &radixNode{
				path:     path,
				pathFull: pathFull,
				data:     data,
			})
			return
		}

		current = current.children[lookupIdx]
	}

	self := &radixNode{
		data:     current.data,
		path:     current.path[commonIdx:],
		pathFull: current.pathFull,
		children: current.children,
		lookup:   current.lookup,
	}

	current.pathFull = nil
	current.path = current.path[:commonIdx]
	if len(current.path) == 0 {
		current.path = nil
	}
	current.data = nil

	if commonIdx == len(path) {
		current.children = []*radixNode{self}
		current.lookup = []byte{self.path[0]}
		current.data = data
		current.pathFull = pathFull
	} else {
		child := &radixNode{
			data:     data,
			path:     path[commonIdx:],
			pathFull: pathFull,
		}

		current.lookup = []byte{self.path[0], path[commonIdx]}
		current.children = []*radixNode{self, child}
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

			return nil, false
		}

		lookupIdx := -1
		for idx := range current.lookup {
			if current.lookup[idx] == path[ptr] {
				lookupIdx = idx
				break
			}
		}

		if lookupIdx == -1 {
			return nil, false
		}

		current = current.children[lookupIdx]
		ptr += len(current.path)
	}
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

	pathBytes := []byte(path)[:]
	if radix.root == nil {
		radix.root = &radixNode{
			path:     pathBytes,
			pathFull: pathBytes,
			data:     data,
		}
	}

	radix.root.add(pathBytes, data)
}

func (radix Radix) Get(path string) (any, bool) {
	return radix.root.get(unsafe.StringToBytes(path))
}
