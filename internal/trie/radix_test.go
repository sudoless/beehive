package trie

import (
	"testing"
)

func testRadixAdd(t *testing.T, paths []string) {
	radix := &Radix{}

	avg := testing.AllocsPerRun(1, func() {
		for idx, path := range paths {
			radix.Add(path, idx)
		}
	})
	t.Logf("allocs: %f", avg)

	m := radix.root.leafs()

	mPaths := make(map[string]any, len(m))
	for idx, path := range paths {
		mPaths[path] = idx
	}

	if len(m) != len(mPaths) {
		t.Errorf("expected %d leafs, got %d", len(paths), len(m))
	}

	for path, idx := range mPaths {
		if m[path] != idx {
			t.Errorf("expected %s to be %d, got %d", path, idx, m[path])
		}
	}
}

func TestRadix_Add(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		paths := []string{"test", "team", "toast", "slow", "water", "slower", "tester"}
		testRadixAdd(t, paths)
	})

	t.Run("romane", func(t *testing.T) {
		paths := []string{"romane", "romanus", "romulus", "rubens", "ruber", "rubicon", "rubicundus"}
		testRadixAdd(t, paths)
	})

	t.Run("duplicate", func(t *testing.T) {
		paths := []string{"test", "test", "team", "slow", "water", "slower", "slow", "toast", "tester", "water"}
		testRadixAdd(t, paths)
	})
}

func BenchmarkRadix_Add(b *testing.B) {
	paths := [...]string{
		"/foo/bar/baz",
		"/foo/bar/buz",
		"/foo/bar/bed",
		"/foo/bar",
		"/foo/bar/bug",
		"/foo/biz/fiz",
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/α",
		"/β",
	}

	b.ReportAllocs()
	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
		radix := &Radix{}
		for idx, path := range paths {
			radix.Add(path, idx)
		}
	}
}

func TestRadix_Get(t *testing.T) {
	t.Parallel()

	paths := []string{"test", "team", "toast", "slow", "water", "slower", "tester"}

	radix := &Radix{}
	for idx, path := range paths {
		radix.Add(path, idx)
	}

	search := map[string]any{
		"test":              0,
		"team":              1,
		"toast":             2,
		"slow":              3,
		"water":             4,
		"slower":            5,
		"tester":            6,
		"slowest":           nil,
		"slo":               nil,
		"t":                 nil,
		"te":                nil,
		"tes":               nil,
		"":                  nil,
		" ":                 nil,
		"largeststringhere": nil,
		"tes_er":            nil,
	}

	for path, mustFind := range search {
		value, _ := radix.Get(path)
		if value != mustFind {
			t.Errorf("expected %s to be %v, got %v", path, mustFind, value)
		}
	}
}

func TestRadix_Get_samePrefix(t *testing.T) {
	t.Parallel()

	paths := []string{"/test", "/team", "/toast", "/slow", "/water", "/slower", "/tester"}

	radix := &Radix{}
	for idx, path := range paths {
		radix.Add(path, idx)
	}

	search := map[string]any{
		"/test":              0,
		"/team":              1,
		"/toast":             2,
		"/slow":              3,
		"/water":             4,
		"/slower":            5,
		"/tester":            6,
		"test":               nil,
		"team":               nil,
		"toast":              nil,
		"slow":               nil,
		"water":              nil,
		"slower":             nil,
		"tester":             nil,
		"/slowest":           nil,
		"/slo":               nil,
		"/t":                 nil,
		"/te":                nil,
		"/tes":               nil,
		"/":                  nil,
		"/ ":                 nil,
		"/largeststringhere": nil,
		"/tes_er":            nil,
		"slowest":            nil,
		"slo":                nil,
		"t":                  nil,
		"te":                 nil,
		"tes":                nil,
		"":                   nil,
		" ":                  nil,
		"largeststringhere":  nil,
		"tes_er":             nil,
	}

	for path, mustFind := range search {
		value, _ := radix.Get(path)
		if value != mustFind {
			t.Errorf("expected %s to be %v, got %v", path, mustFind, value)
		}
	}
}

func TestRadix_Get_empty(t *testing.T) {
	t.Parallel()

	radix := &Radix{}
	_, found := radix.Get("")
	if found {
		t.Errorf("expected not to find empty")
	}

	radix.Add("", 0)

	_, found = radix.Get("")
	if found {
		t.Errorf("expected not to find empty")
	}
}

func TestRadix_Get_0alloc(t *testing.T) {
	paths := [...]string{
		"/foo/bar/baz",
		"/foo/bar/buz",
		"/foo/bar/bed",
		"/foo/bar",
		"/foo/bar/bug",
		"/foo/biz/fiz",
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/α",
		"/β",
	}

	radix := &Radix{}
	for idx, path := range paths {
		radix.Add(path, idx)
	}

	alloc := testing.AllocsPerRun(100, func() {
		for _, path := range paths {
			_, found := radix.Get(path)
			if !found {
				t.Errorf("expected to find %s", path)
			}
		}
	})

	if alloc != 0 {
		t.Errorf("alloc = %v, want 0", alloc)
	}

	if t.Failed() {
		for k, v := range radix.root.leafs() {
			t.Logf("%s: %v", k, v)
		}
	}
}

func TestRadix_Get_newRadix(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("expected not to panic")
		}
	}()

	radix := &Radix{}
	radix.Get("/foo/bar")
}

func TestRadix_Get_special(t *testing.T) {
	t.Parallel()

	paths := []string{"GET/api/health"}

	radix := &Radix{}
	for idx, path := range paths {
		radix.Add(path, idx)
	}

	search := map[string]any{
		"POST/api/hive": nil,
	}

	for path, mustFind := range search {
		value, _ := radix.Get(path)
		if value != mustFind {
			t.Errorf("expected %s to be %v, got %v", path, mustFind, value)
		}
	}
}

func BenchmarkRadix_Get(b *testing.B) {
	paths := [...]string{
		"/foo/bar/baz",
		"/foo/bar/buz",
		"/foo/bar/bed",
		"/foo/bar",
		"/foo/bar/bug",
		"/foo/biz/fiz",
		"/hi",
		"/contact",
		"/co",
		"/c",
		"/a",
		"/ab",
		"/doc/",
		"/doc/go_faq.html",
		"/doc/go1.html",
		"/α",
		"/β",
	}

	radix := &Radix{}
	for idx, path := range paths {
		radix.Add(path, idx)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
		for _, path := range paths {
			_, _ = radix.Get(path)
		}
	}
}

func TestRadix_wildcard(t *testing.T) {
	t.Parallel()

	paths := []string{
		"/foo/baz",
		"/foo/*",
		"/foo/baz/fiz",
		"/foo/bar",
		"/foo",
		"/foo/bar/qux/*",
		"/111",
		"/1*",
		"/2/*",
		"/2/3/4*",
		"/2/3*",
	}

	pathsNoHandlers := []string{
		"/3",
		"/3*",
		"/11/2*",
		"/11/2345",
		"/11/23456",
	}

	radix := &Radix{}
	for idx, path := range paths {
		radix.Add(path, idx)
	}
	for idx, path := range pathsNoHandlers {
		radix.Add(path, idx*1000)
	}

	search := []string{
		"/foo/baz",
		"/foo/1",
		"/foo/2",
		"/foo/baz/fiz",
		"/foo/bar",
		"/foo",
		"/foo/bar/qux/1",
		"/foo/bar/qux/2",
		"/111",
		"/123",
		"/123456789",
		"/2/1",
		"/2/2",
		"/2/3/456",
		"/2/3456",
	}

	for _, path := range search {
		t.Run(path, func(t *testing.T) {
			_, found := radix.Get(path)
			if !found {
				t.Errorf("expected to find %s", path)
			}
		})
	}

	//m := n.PathsHandlers()
	//for _, path := range paths {
	//	path = strings.ReplaceAll(path, "*", "$")
	//	if _, ok := m[path]; !ok {
	//		t.Errorf("expected %s path", path)
	//	}
	//}
}

func TestRadix_wildcard_special(t *testing.T) {
	t.Parallel()

	t.Run("/*", func(t *testing.T) {
		radix := &Radix{}

		paths := []string{
			"/*",
			"/foo",
			"/fo*",
			"/f",
		}

		for idx, path := range paths {
			radix.Add(path, idx)
		}

		search := []string{
			"/foobar",
			"/foo",
			"/f",
			"/foo123",
			"/123",
		}

		for _, path := range search {
			_, found := radix.Get(path)
			if !found {
				t.Errorf("expected to find %s", path)
			}
		}

		out := radix.root.leafs()
		for path := range out {
			t.Log(path)
		}
	})
	t.Run("*", func(t *testing.T) {
		radix := &Radix{}
		paths := []string{
			"*",
		}

		for idx, path := range paths {
			radix.Add(path, idx)
		}

		search := []string{
			"/anything",
			"/",
			"/goes",
			"/test",
			"/123",
			"test",
		}

		for _, path := range search {
			_, found := radix.Get(path)
			if !found {
				t.Errorf("expected to find %s", path)
			}
		}
	})
	t.Run("/test* /test search", func(t *testing.T) {
		radix := &Radix{}

		paths := []string{
			"/foo*",
		}

		for idx, path := range paths {
			radix.Add(path, idx)
		}

		search := []string{
			"/foo",
			"/foobar",
		}

		for _, path := range search {
			_, found := radix.Get(path)
			if !found {
				t.Errorf("expected to find %s", path)
			}
		}
	})
}