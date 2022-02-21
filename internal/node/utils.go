package node

func commonPrefix(s0, s1 string) int {
	idx := 0
	max := len(s1)
	if len(s0) < max {
		max = len(s0)
	}

	for idx < max && s0[idx] == s1[idx] {
		idx++
	}

	return idx
}

func (trie *Trie) Paths() []string {
	if len(trie.children) == 0 {
		if trie.isWildcard {
			return []string{trie.path + "$"}
		}

		return []string{trie.path}
	}
	var paths []string
	for _, child := range trie.children {
		childPaths := child.Paths()
		for idx := range childPaths {
			childPaths[idx] = trie.path + childPaths[idx]
		}
		paths = append(paths, childPaths...)
	}

	if trie.isWildcard {
		paths = append(paths, trie.path+"$")
	}

	return paths
}

func (trie *Trie) PathsHandlers() map[string]struct{ Handlers interface{} } {
	m := make(map[string]struct{ Handlers interface{} })

	if trie.data != nil {
		v := struct{ Handlers interface{} }{trie.data}
		if trie.isWildcard {
			m[trie.path+"$"] = v
		} else {
			m[trie.path] = v
		}
	}
	if len(trie.children) == 0 {
		return m
	}

	for _, child := range trie.children {
		cm := child.PathsHandlers()
		for k, v := range cm {
			m[trie.path+k] = v
		}
	}

	if trie.isWildcard {
		m[trie.path+"$"] = struct{ Handlers interface{} }{trie.data}
	}

	return m
}
