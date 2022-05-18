package beehive_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/sudoless/beehive/pkg/beehive"
)

func Example_newRouterWithHttpServer() {
	// create a new, empty router
	router := beehive.NewRouter()

	// the beehive.Router has no builtin Server (the element that listens on a port for TCP/UDP connections)
	// so you must attach it to a Server (or multiple Servers)

	// let's create a new Server
	server := http.Server{
		Addr:    "localhost:8080", // assign the listening address and port
		Handler: router,           // pass the beehive.Router as the handler (as it implements ServeHTTP, http.Handler interface)

		// in production, you should configure TLS, timeouts, and other Server settings
		// ...
	}

	// start the server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		// handle error
		log.Fatalf("server error: %v\n", err)
	}

	// this router has no handlers, no middlewares, no 404 and other _error_ handlers, so calling any path with any
	// method should result in a 200 OK empty response
}

func Example_routerHandlerFunc() {
	// create a new, empty router
	router := beehive.NewRouter()

	// tell the router, to handle the path "/my/path" for any "GET" method requests, and execute the anonymously
	// defined beehive.HandlerFunc which will return the string "Hello, World!" as the response body and a 200 status
	router.Handle("GET", "/my/path", func(ctx *beehive.Context) beehive.Responder {
		return &beehive.DefaultResponder{
			Message: []byte("Hello World!"),
			Status:  200,
		}
	})

	// setup http server
	// ...

	// simulate a http request
	r := httptest.NewRequest("GET", "/my/path", nil)
	w := httptest.NewRecorder()

	// serve the request
	router.ServeHTTP(w, r)

	// check the response
	fmt.Println(w.Body.String())
	fmt.Println(w.Code)

	// Output:
	// Hello World!
	// 200
}

func Example_grouping() {
	// create a new, empty router
	router := beehive.NewRouter()

	// define a simple handler func
	hello := func(ctx *beehive.Context) beehive.Responder {
		return &beehive.DefaultResponder{
			Message: []byte("Hello World!"),
			Status:  200,
		}
	}

	// define a simple handler func for our other group
	helloInternal := func(ctx *beehive.Context) beehive.Responder {
		return &beehive.DefaultResponder{
			Message: []byte("Hello Internal!"),
			Status:  200,
		}
	}

	// group /api* endpoints
	api := router.Group("/api")
	{ // brackets can be used for styling and better IDE folding, but are not required
		api.Handle("GET", "/foo", hello) // GET /api/foo
		api.Handle("PUT", "/bar", hello) // PUT /api/bar
	}

	// group /internal* endpoints
	internal := router.Group("/internal")
	{
		internal.Handle("GET", "/foo", helloInternal) // GET /internal/foo
		internal.Handle("PUT", "/bar", helloInternal) // PUT /internal/bar
	}

	// setup http server
	// ...

	// simulate a http request
	r := httptest.NewRequest("GET", "/api/foo", nil)
	w := httptest.NewRecorder()

	// serve the request
	router.ServeHTTP(w, r)

	// check the response
	fmt.Println(w.Body.String())

	// simulate a http request
	r = httptest.NewRequest("GET", "/internal/foo", nil)
	w = httptest.NewRecorder()

	// serve the request
	router.ServeHTTP(w, r)

	// check the response
	fmt.Println(w.Body.String())

	// Output:
	// Hello World!
	// Hello Internal!
}

func Example_middleware() {
	// create a new, empty router
	router := beehive.NewRouter()

	// define middleware handler func
	checkAuth := func(ctx *beehive.Context) beehive.Responder {
		// check request basic auth
		user, pass, ok := ctx.Request.BasicAuth()
		if !ok {
			// if none, return 401
			return &beehive.DefaultResponder{
				Message: []byte("Unauthorized"),
				Status:  401,
			}
		}

		// check user/pass
		if user != "admin" || pass != "secret" {
			// if not matching, return 401
			return &beehive.DefaultResponder{
				Message: []byte("Unauthorized"),
				Status:  401,
			}
		}

		// if all ok, return nil to continue the handler chain
		return nil
	}

	// define the handler func we want to protect with our checkAuth middleware
	secretResource := func(ctx *beehive.Context) beehive.Responder {
		return &beehive.DefaultResponder{
			Message: []byte("Secret Resource"),
			Status:  200,
		}
	}

	// tell the router to handle the middleware and handler
	router.Handle("GET", "/secret", checkAuth, secretResource)

	// setup http server
	// ...

	// simulate a http request (with no auth)
	r := httptest.NewRequest("GET", "/secret", nil)
	w := httptest.NewRecorder()

	// serve the request
	router.ServeHTTP(w, r)

	// check the response
	fmt.Println(w.Body.String())
	fmt.Println(w.Code)

	// simulate a http request (with auth)
	r = httptest.NewRequest("GET", "/secret", nil)
	w = httptest.NewRecorder()
	r.SetBasicAuth("admin", "secret")

	// serve the request
	router.ServeHTTP(w, r)

	// check the response
	fmt.Println(w.Body.String())
	fmt.Println(w.Code)

	// Output:
	// Unauthorized
	// 401
	// Secret Resource
	// 200
}

func Example_context() {
	// create a new, empty router
	router := beehive.NewRouter()

	// assign a context function
	router.Context = func(r *http.Request) context.Context {
		return context.WithValue(context.Background(), "router_name", "🍯")
	}

	// define a handler func that takes the value from the context and returns it
	router.Handle("GET", "/router_name", func(ctx *beehive.Context) beehive.Responder {
		return &beehive.DefaultResponder{
			Message: []byte(ctx.Value("router_name").(string)),
			Status:  200,
		}
	})

	// setup http server
	// ...

	// simulate a http request
	r := httptest.NewRequest("GET", "/router_name", nil)
	w := httptest.NewRecorder()

	// serve the request
	router.ServeHTTP(w, r)

	// check the response
	fmt.Println(w.Body.String())

	// Output:
	// 🍯
}

func Example_contextMiddleware() {
	// create a new, empty router
	router := beehive.NewRouter()

	// define a middleware that assigns a value to the context
	withUser := func(ctx *beehive.Context) beehive.Responder {
		user, _, _ := ctx.Request.BasicAuth()
		if user == "" {
			user = "valued user"
		}

		ctx.WithValue("user", user)

		return nil
	}

	// define a handler that makes use of the value from the context
	hello := func(ctx *beehive.Context) beehive.Responder {
		user := ctx.Value("user")
		if user == nil {
			user = "nil"
		}

		return &beehive.DefaultResponder{
			Message: []byte(fmt.Sprintf("Hello, '%s'!", user.(string))),
			Status:  200,
		}
	}

	router.Handle("GET", "/with_user/hello", withUser, hello)
	router.Handle("GET", "/hello", hello)

	// setup http server
	// ...

	// simulate a http request (with user to the /with_user/hello endpoint)
	r := httptest.NewRequest("GET", "/with_user/hello", nil)
	w := httptest.NewRecorder()
	r.SetBasicAuth("alex", "")
	router.ServeHTTP(w, r)
	fmt.Println(w.Body.String())

	// simulate a http request (without user to the /with_user/hello endpoint)
	r = httptest.NewRequest("GET", "/with_user/hello", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	fmt.Println(w.Body.String())

	// simulate a http request (with user to the /hello endpoint)
	r = httptest.NewRequest("GET", "/hello", nil)
	w = httptest.NewRecorder()
	r.SetBasicAuth("alex", "")
	router.ServeHTTP(w, r)
	fmt.Println(w.Body.String())

	// Output:
	// Hello, 'alex'!
	// Hello, 'valued user'!
	// Hello, 'nil'!
}

func Example_middlewareNext() {
	// create a new, empty router
	router := beehive.NewRouter()

	logger := func(ctx *beehive.Context) beehive.Responder {
		fmt.Printf("request %s %s\n", ctx.Request.Method, ctx.Request.URL.Path) // <-- #1
		responder := ctx.Next()
		fmt.Printf("response %d\n", responder.StatusCode(ctx)) // <-- #3

		return responder
	}

	elapsed := func(ctx *beehive.Context) beehive.Responder {
		start := time.Now()
		responder := ctx.Next()
		elapsed := time.Since(start)

		if elapsed > time.Millisecond*100 {
			fmt.Println("🐌") // <-- #2a
		} else {
			fmt.Println("⚡️") // <-- #2b
		}

		return responder
	}

	// assign both middlewares to the same group, in order 'logger -> elapsed'
	api := router.Group("/api", logger, elapsed)
	{
		api.Handle("GET", "/fast", func(ctx *beehive.Context) beehive.Responder {
			return &beehive.DefaultResponder{
				Message: []byte("fast"),
				Status:  200,
			}
		})

		api.Handle("GET", "/slow", func(ctx *beehive.Context) beehive.Responder {
			time.Sleep(time.Millisecond * 100)
			return &beehive.DefaultResponder{
				Message: []byte("slow"),
				Status:  418,
			}
		})
	}

	// setup http server
	// ...

	// simulate a http request (fast)
	r := httptest.NewRequest("GET", "/api/fast", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)
	fmt.Println(w.Body.String())

	// simulate a http request (slow)
	r = httptest.NewRequest("GET", "/api/slow", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, r)
	fmt.Println(w.Body.String())

	// explaining the output:
	// the first 4 lines are from the first request
	// the last 4 lines are from the second request
	// the line 'request GET /api/fast' is generated by the logger (#1), then the Next function is called leading to
	// the next middleware, which is elapsed, which before printing anything calls the Next function again, which is
	// the handler for the request, which depending on the endpoint can be /fast or /slow, in the first call it is
	// fast, causing the return to be sub 100ms elapsed, making (#2a) the output then we get back to the logger
	// middleware, right where we left off by calling Next, and reach the next print (#3) which is the status code
	// from the handler response
	// for the second request we have similar behaviour, but the difference is that the elapsed time is greater than
	// 100ms causing (#2b) to be the output

	// Output:
	// request GET /api/fast
	// ⚡️
	// response 200
	// fast
	// request GET /api/slow
	// 🐌
	// response 418
	// slow
}
