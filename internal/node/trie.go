package node

type Trie struct {
	data       interface{}
	path       string
	children   []*Trie
	lookup     []byte
	isWildcard bool
}

func (trie *Trie) Add(path string, data interface{}) error {
	if trie.path == "" || trie.path == path {
		return trie.set(path, data)
	}

	current := trie
	commonIdx := commonPrefix(path, current.path)

	isWildcard := path[len(path)-1] == '*'

	for ; commonIdx == len(current.path); commonIdx = commonPrefix(path, current.path) {
		if commonIdx == len(path) {
			return current.set("", data)
		}

		if isWildcard && commonIdx == len(path)-1 {
			current.isWildcard = true
			return current.set("", data)
		}

		path = path[commonIdx:]
		lookupIdx := current.findChildIndex(path[0])
		if lookupIdx == -1 {
			return current.createNew(path, data)
		}

		current = current.children[lookupIdx]
	}

	current.createSplit(commonIdx, path, data)
	return nil
}

func (trie *Trie) Get(path string) (current *Trie, err error) {
	var wildcard *Trie

	current = trie
	currentPrefix := current.path

	for {
		if current.isWildcard {
			wildcard = current
		}

		if path == currentPrefix {
			return current, nil
		}

		if len(current.children) == 0 {
			if wildcard != nil && commonPrefix(path, currentPrefix) == len(currentPrefix) {
				return wildcard, nil
			}

			return current, ErrNodeNotFound
		}

		if len(path) <= len(currentPrefix) {
			return current, ErrNodeNotFound
		}

		idx := current.findChildIndex(path[len(currentPrefix)])
		if idx == -1 {
			if wildcard != nil {
				return wildcard, nil
			}

			return current, ErrNodeNotFound
		}

		if path[:len(currentPrefix)] == currentPrefix {
			path = path[len(currentPrefix):]
			current = current.children[idx]
			currentPrefix = current.path
		}
	}
}

func (trie *Trie) findChildIndex(b byte) int {
	for idx, lookup := range trie.lookup {
		if b == lookup {
			return idx
		}
	}

	return -1
}

func (trie *Trie) set(path string, data interface{}) error {
	if path != "" {
		if path[len(path)-1] == '*' {
			trie.isWildcard = true
			path = path[:len(path)-1]
		}

		trie.path = path
	}

	if data != nil {
		if trie.data != nil {
			return ErrNodeAlreadyExists
		}

		trie.data = data
	}

	return nil
}

func (trie *Trie) createNew(path string, data interface{}) error {
	child := &Trie{}
	_ = child.set(path, data)

	trie.children = append(trie.children, child)
	trie.lookup = append(trie.lookup, path[0])

	return nil
}

func (trie *Trie) createSplit(commonIdx int, path string, data interface{}) {
	self := &Trie{}
	*self = *trie

	isWildcard := false
	if path[len(path)-1] == '*' {
		path = path[:len(path)-1]
		isWildcard = true
	}

	self.path = self.path[commonIdx:]

	trie.path = trie.path[:commonIdx]
	trie.data = nil
	trie.isWildcard = false

	if commonIdx == len(path) {
		trie.children = []*Trie{self}
		trie.lookup = []byte{self.path[0]}
		trie.isWildcard = isWildcard

		_ = trie.set("", data)
	} else {
		child := &Trie{}
		child.isWildcard = isWildcard

		_ = child.set(path[commonIdx:], data)

		trie.children = []*Trie{self, child}
		trie.lookup = []byte{self.path[0], path[commonIdx]}
	}
}

func (trie *Trie) Data() interface{} {
	return trie.data
}
