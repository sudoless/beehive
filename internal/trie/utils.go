package trie

func commonPrefix(b0, b1 []byte) int {
	idx := 0
	max := len(b1)
	if len(b0) < max {
		max = len(b0)
	}

	for idx < max && b0[idx] == b1[idx] {
		idx++
	}

	return idx
}
