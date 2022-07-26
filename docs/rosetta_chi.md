<table>
<tr>
<th>Echo</th>
<th>Beehive</th>
</tr>
<tr>
<td>

```go
package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	http.ListenAndServe(":3000", r)
}
```

</td>
<td>

```go
package main

import (
	"log"
	"net/http"
	"time"

	"go.sdls.io/beehive/pkg/beehive"
)


func main() {
	bee := beehive.NewRouter()
	bee.Handle("GET", "/", loggerMiddleware, func(ctx *beehive.Context) beehive.Responder {
		return &beehive.DefaultResponder{
            Message: "welcome",
            Status:  200,
        }
	})

	server := &http.Server{
		Addr:    ":3000",
		Handler: bee,
	}

	_ = server.ListenAndServe()
}

func loggerMiddleware(ctx *beehive.Context) beehive.Responder {
	start := time.Now()
	res := ctx.Next()
	elapsed := time.Since(start)

	log.Printf("%s %s %d %dms\n",
		ctx.Request.Method,
		ctx.Request.URL.Path,
		res.StatusCode(ctx),
		elapsed.Milliseconds(),
	)

	return res
}

```

</td>
</tr>
</table>
