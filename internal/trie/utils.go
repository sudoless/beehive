package trie

func commonPrefix(b0, b1 []byte) int {
	idx := 0
	limit := min(len(b0), len(b1))

	for idx < limit && b0[idx] == b1[idx] {
		idx++
	}

	return idx
}
