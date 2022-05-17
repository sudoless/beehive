package beehive_query

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sudoless/beehive/pkg/beehive"
)

func TestValues_parse(t *testing.T) {
	t.Parallel()

	q := &Values{
		dict: map[string]int{
			"":    0,
			"foo": 1,
			"bar": 2,
			"baz": 3,
		},
		values: make([]string, 4),
	}

	q.parse("foo=123&bar=456&baz=789")
	expected := []string{"", "123", "456", "789"}

	if !reflect.DeepEqual(q.values, expected) {
		t.Errorf("expected %v, got %v", expected, q.values)
	}
}

func Test_ValuesParser(t *testing.T) {
	t.Parallel()

	fields := []string{"foo", "bar", "baz"}

	router := beehive.NewDefaultRouter()
	router.Context = func(_ *http.Request) context.Context {
		return context.Background()
	}

	router.Handle(http.MethodGet, "/foo/bar", Parser(fields),
		func(ctx *beehive.Context) beehive.Responder {
			message := &bytes.Buffer{}
			values := ContextValues(ctx)

			for _, value := range values.values {
				message.WriteString(value)
				message.WriteRune('\n')
			}

			return &beehive.DefaultResponder{
				Message: message.Bytes(),
				Status:  http.StatusOK,
			}
		})

	queries := map[string][]string{
		"":                                {"", "", "", ""},
		"foo=123":                         {"", "123", "", ""},
		"foo=123&bar=456":                 {"", "123", "456", ""},
		"foo=123&bar=456&baz=789":         {"", "123", "456", "789"},
		"foo=123&bar=456&baz=789&":        {"", "123", "456", "789"},
		"fiz=123&biz=456&buz=789":         {"", "", "", ""},
		"foo=123&bar=456&baz=789&foo=123": {"", "123", "456", "789"},
		"foo=123&bar=456&baz=789&foo=000": {"", "000", "456", "789"},
		"foo=123&&&bar=456&baz=789":       {"", "123", "", "789"},
		"foobarbaz=123456789":             {"", "", "", ""},
		"foo==123&bar===456&baz====789":   {"", "=123", "==456", "===789"},
		"foo=123&bar=&456&baz=&&789":      {"", "123", "", ""},
		"foo=1&bar=2;&baz=3":              {"", "1", "2;", "3"},
		"foo=123&foo=456&foo=789":         {"", "789", "", ""},
		"bar=%3Ckey%3A+0x90%3E":           {"", "", "%3Ckey%3A+0x90%3E", ""},
		"foo=1&;":                         {"", "1", "", ""},
		"foo=&bar=&baz=":                  {"", "", "", ""},
	}

	for query, values := range queries {
		t.Run(query, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/foo/bar?"+query, nil)
			router.ServeHTTP(w, r)

			if w.Code != http.StatusOK {
				t.Errorf("expected status code %d, got %d", http.StatusOK, w.Code)
			}

			message := &bytes.Buffer{}
			for _, value := range values {
				message.WriteString(value)
				message.WriteRune('\n')
			}

			if !bytes.Equal(w.Body.Bytes(), message.Bytes()) {
				t.Errorf("expected %s, got %s", message.String(), w.Body.String())
			}
		})
	}
}

func Benchmark_ValuesParser(b *testing.B) {
	parser := Parser([]string{"foo", "bar", "baz"})
	req := httptest.NewRequest("GET", "/foo/bar?foo=123&bar=456&baz=789&foo=000", nil)
	ctx := &beehive.Context{
		ResponseWriter: nil,
		Request:        req,
		Context:        context.Background(),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for iter := 0; iter < b.N; iter++ {
		parser(ctx)
	}

	values := ContextValues(ctx)
	if values == nil {
		b.Fatal("expected request.Values to be not nil")
	}

	expected := []string{"", "000", "456", "789"}
	if !reflect.DeepEqual(expected, values.values) {
		b.Errorf("expected %v, got %v", expected, values.values)
	}
}
