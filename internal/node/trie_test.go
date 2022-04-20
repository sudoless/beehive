package node

import (
	"strings"
	"testing"
)

func testHandlerDummy() {}

func Test_node_add_get(t *testing.T) {
	t.Parallel()

	var handlers []any

	paths := []string{
		"/foo/bar/baz",
		"/foo/bar/buz",
		"/foo/bar/bed",
		"/foo/bar",
		"/foo/bar/bug",
		"/foo/biz/fiz",
		"/f/",
		"/f",
		"/fo",
		"/foo",
		"/f/o",
		"/qux",
		"/qux/foo",
		"/qux/foo/bar",
	}

	var n Trie

	m := make(map[string]int)
	for _, path := range paths {
		m[path] = 1

		err := n.Add(path, handlers)
		if err != nil {
			t.Errorf("add(%s) returned error %s", path, err)
		}
	}

	for _, path := range paths {
		_, err := n.Get(path)
		if err != nil {
			t.Errorf("%s: %s", path, err)
		}
	}

	outPaths := n.Paths()
	for _, path := range outPaths {
		if _, ok := m[path]; !ok {
			t.Errorf("path %s not found", path)
		}
	}
}

func Test_node_get_0alloc(t *testing.T) {
	var handlers []any

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

	var n Trie
	for _, path := range paths {
		_ = n.Add(path, handlers)
	}

	alloc := testing.AllocsPerRun(100, func() {
		for _, path := range paths {
			_, _ = n.Get(path)
		}
	})

	if alloc != 0 {
		t.Errorf("alloc = %v, want 0", alloc)
	}
}

func Test_node_get_0alloc_concat(t *testing.T) {
	var handlers []any

	methods := []string{
		"GET",
		"POST",
		"PUT",
	}

	var n Trie
	for _, method := range methods {
		_ = n.Add("/"+method+"/foobar", handlers)
	}

	for _, method := range methods {
		alloc := testing.AllocsPerRun(100, func() {
			_, _ = n.Get("/" + method + "/foobar")
		})

		if alloc != 0 {
			t.Errorf("alloc = %v, want 0", alloc)
		}
	}
}

func Test_node_get_not_found(t *testing.T) {
	t.Parallel()

	t.Run("not found, no children", func(t *testing.T) {
		n := Trie{}

		_, err := n.Get("/foo/bar")
		if err == nil {
			t.Error("expected error")
		}

		if err != ErrNodeNotFound {
			t.Errorf("expected ErrNodeNotFound, got %v", err)
		}
	})
	t.Run("not found, path len", func(t *testing.T) {
		n := Trie{}

		_ = n.Add("/foo/bar", nil)
		_ = n.Add("/foo/biz", nil)

		_, err := n.Get("/foo")
		if err == nil {
			t.Error("expected error")
		}

		if err != ErrNodeNotFound {
			t.Errorf("expected ErrNodeNotFound, got %v", err)
		}
	})
	t.Run("not found, matching index", func(t *testing.T) {
		n := Trie{}

		_ = n.Add("/foo/123", nil)
		_ = n.Add("/foo/456", nil)
		_ = n.Add("/foo/789", nil)

		_, err := n.Get("/foo/000")
		if err == nil {
			t.Error("expected error")
		}

		if err != ErrNodeNotFound {
			t.Errorf("expected ErrNodeNotFound, got %v", err)
		}
	})
}

func Benchmark_node_add(b *testing.B) {
	var handlers []any

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
		var n Trie
		for _, path := range paths {
			_ = n.Add(path, handlers)
		}
	}
}

func Benchmark_node_get(b *testing.B) {
	var handlers []any

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

	var n Trie
	for _, path := range paths {
		_ = n.Add(path, handlers)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
		for _, path := range paths {
			_, _ = n.Get(path)
		}
	}

	b.StopTimer()
}

func Test_node_add_special(t *testing.T) {
	t.Parallel()

	t.Run("split", func(t *testing.T) {
		n := &Trie{}

		_ = n.Add("/foo/bar/baz/qux", []any{testHandlerDummy})
		_ = n.Add("/foo/qux", []any{testHandlerDummy})

		if len(n.children) != 2 {
			t.Fatalf("expected 2 children, got %d", len(n.children))
		}

		if n.path != "/foo/" {
			t.Errorf("expected '/foo/' path for root node, got '%s'", n.path)
		}

		if n.children[0].path != "bar/baz/qux" {
			t.Errorf("expected 'bar/baz/qux' path for child 0, got '%s'", n.children[0].path)
		}

		if n.children[1].path != "qux" {
			t.Errorf("expected 'qux' path for child 1, got '%s'", n.children[1].path)
		}
	})

	t.Run("spawn", func(t *testing.T) {
		n := &Trie{}

		_ = n.Add("/foo/bar", []any{testHandlerDummy})
		_ = n.Add("/foo/bar/qux", []any{testHandlerDummy})
	})
}

func Test_node_add_duplicate(t *testing.T) {
	t.Parallel()

	n := &Trie{}
	paths := []string{
		"/foo/bar",
		"/",
		"/foo/bar/baz",
		"/abc/def",
		"/foo",
		"/abc",
		"/foo/abc",
		"/fo",
		"/a",
		"/foo/bar/baz/biz",
		"/foo/ba",
		"/a/b/",
		"/foo/b",
		"/foo/",
		"/a/b/c",
		"/a/b/cd",
		"/a/bc/",
		"/a/b/c/d/e/f/g",
		"/b/c/d/e/f/",
		"/b/a/d/e/f",
	}

	for _, path := range paths {
		if err := n.Add(path, []any{testHandlerDummy}); err != nil {
			t.Fatal(err)
		}
	}

	m := n.PathsHandlers()
	if len(m) != len(paths) {
		t.Fatalf("expected %d paths, got %d", len(paths), len(m))
	}

	for k, v := range m {
		l := len(v.Handlers.([]any))
		if l != 1 {
			t.Errorf("%s: expected 1 handler, got %d", k, l)
		}
	}

	for _, path := range paths {
		if err := n.Add(path, []any{testHandlerDummy}); err == nil {
			t.Errorf("%s: expected error, got nil", path)
		}
	}
}

func Test_node_add_wildcard(t *testing.T) {
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

	n := &Trie{}
	for _, path := range paths {
		if err := n.Add(path, []any{testHandlerDummy}); err != nil {
			t.Fatalf("failed adding '%s', %v", path, err)
		}
	}
	for _, path := range pathsNoHandlers {
		if err := n.Add(path, nil); err != nil {
			t.Fatalf("failed adding '%s', %v", path, err)
		}
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
		found, err := n.Get(path)
		if err != nil {
			t.Errorf("error walking to node '%s', %v", path, err)
		}
		_ = found
	}

	m := n.PathsHandlers()
	for _, path := range paths {
		path = strings.ReplaceAll(path, "*", "$")
		if _, ok := m[path]; !ok {
			t.Errorf("expected %s path", path)
		}
	}

}

func Test_node_add_wildcard_edge_case(t *testing.T) {
	t.Parallel()

	handler := []any{testHandlerDummy}

	t.Run("/*", func(t *testing.T) {
		n := &Trie{}

		paths := []string{
			"/*",
			"/foo",
			"/fo*",
			"/f",
		}

		for _, path := range paths {
			if err := n.Add(path, handler); err != nil {
				t.Fatalf("failed adding '%s', %v", path, err)
			}
		}

		search := []string{
			"/foobar",
			"/foo",
			"/f",
			"/foo123",
			"/123",
		}

		for _, path := range search {
			found, err := n.Get(path)
			if err != nil {
				t.Errorf("error walking to node '%s', %v", path, err)
			}
			_ = found
		}

		out := n.Paths()
		for _, path := range out {
			t.Log(path)
		}
	})
	t.Run("*", func(t *testing.T) {
		n := &Trie{}

		paths := []string{
			"*",
		}

		for _, path := range paths {
			if err := n.Add(path, handler); err != nil {
				t.Fatalf("failed adding '%s', %v", path, err)
			}
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
			found, err := n.Get(path)
			if err != nil {
				t.Errorf("error walking to node '%s', %v", path, err)
			}
			_ = found
		}
	})
	t.Run("/test* /test search", func(t *testing.T) {
		n := &Trie{}

		paths := []string{
			"/foo*",
		}

		for _, path := range paths {
			if err := n.Add(path, handler); err != nil {
				t.Fatalf("failed adding '%s', %v", path, err)
			}
		}

		search := []string{
			"/foo",
			"/foobar",
		}

		for _, path := range search {
			found, err := n.Get(path)
			if err != nil {
				t.Errorf("error walking to node '%s', %v", path, err)
			}
			_ = found
		}
	})
}
