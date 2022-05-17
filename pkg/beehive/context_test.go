package beehive

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	t.Parallel()

	var ctxKey struct{}

	ctx := &Context{
		Context: context.WithValue(context.Background(), ctxKey, "bar"),
	}

	deadline, _ := ctx.Deadline()
	if !deadline.IsZero() {
		t.Errorf("deadline should be zero")
	}

	if ctx.Done() != nil {
		t.Errorf("done should be nil")
	}

	if ctx.Err() != nil {
		t.Errorf("err should be nil")
	}

	foo := ctx.Value(ctxKey)
	if foo.(string) != "bar" {
		t.Errorf("value should be bar")
	}
}

func TestContext_String(t *testing.T) {
	t.Parallel()

	ctxKey := &struct{}{}

	ctx := &Context{
		Context: context.WithValue(context.Background(), ctxKey, "bar"),
	}

	str := ctx.String()
	if str != "beehive.Context(idx=0, handlers=[], ctx=(context.Background.WithValue(type *struct {}, val bar)))" {
		t.Errorf("bad context str, got '%s'", str)
	}
}

func TestContext_WithValue(t *testing.T) {
	t.Parallel()

	var ctxKey struct{}
	ctx1 := context.Background()

	ctxA := &Context{
		ResponseWriter: nil,
		Request:        nil,
		Context:        ctx1,
	}
	ctxB := &Context{
		ResponseWriter: nil,
		Request:        nil,
		Context:        ctx1,
	}

	ctxA.WithValue(ctxKey, "foo")
	ctxB.Context = context.WithValue(ctxB.Context, ctxKey, "bar")

	if ctxA.Value(ctxKey).(string) != "foo" {
		t.Errorf("value should be foo")
	}

	if ctxB.Value(ctxKey).(string) != "bar" {
		t.Errorf("value should be bar")
	}
}

//type testContextContractResponder struct{}
//
//func (t testContextContractResponder) StatusCode(ctx context.Context, _ *http.Request) int {
//	switch ctx.(type) {
//	case *Context:
//	default:
//		panic("context should be *Context")
//	}
//
//	value := ctx.Value("middleware")
//	if value != "yes" {
//		return 500
//	}
//
//	value = ctx.Value("foo")
//	if value != "bar" {
//		return 501
//	}
//
//	return http.StatusAccepted
//}
//
//func (t testContextContractResponder) Body(_ context.Context, _ *http.Request) []byte {
//	return nil
//}
//
//func (t testContextContractResponder) Headers(_ context.Context, _ *http.Request, _ http.Header) {
//	return
//}
//
//func (t testContextContractResponder) Cookies(_ context.Context, _ *http.Request) []*http.Cookie {
//	return nil
//}
//
//func testContextContractMiddleware(ctx *Context) Responder {
//	// return ctx.Next(context.WithValue(ctx, "middleware", "yes"), Request)
//	return ctx.Next()
//}
//
//func Test_Context_Contract(t *testing.T) {
//	t.Parallel()
//
//	myCtx := context.WithValue(context.Background(), "foo", "bar")
//
//	router := NewRouter()
//	router.Context = func(Request *http.Request) context.Context {
//		return myCtx
//	}
//
//	foo := router.Group("/foo", testContextContractMiddleware)
//	foo.Handle("GET", "/do", func(ctx *Context) Responder {
//		if ctx.Value("middleware") != "yes" {
//			t.Errorf("middleware should be yes")
//		}
//
//		if ctx.Value("foo") != "bar" {
//			t.Errorf("foo should be bar")
//		}
//
//		return testContextContractResponder{}
//	})
//
//	ResponseWriter := httptest.NewRecorder()
//	Request := httptest.NewRequest("GET", "/foo/do", nil)
//	router.ServeHTTP(ResponseWriter, Request)
//
//	if ResponseWriter.Code != http.StatusAccepted {
//		t.Errorf("bad status code, got %d, meaning values did not propagate with the context properly", ResponseWriter.Code)
//	}
//}
