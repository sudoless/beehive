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
    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "net/http"
)

func main() {
    // Echo instance
    e := echo.New()
    
    // Middleware
    e.Use(middleware.Logger())
    e.Use(middleware.Recover())
    
    // Routes
    e.GET("/", hello)
    
    // Start server
    e.Logger.Fatal(e.Start(":1323"))
}

// Handler
func hello(c echo.Context) error {
    return c.String(http.StatusOK, "Hello, World!")
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

	"github.com/sudoless/beehive/pkg/beehive"
)


func main() {
	bee := beehive.NewRouter()

	bee.Recover = func(ctx *beehive.Context, panicErr any) beehive.Responder {
		// implement desired recovery strategy here
		return nil
	}

	bee.Handle("GET", "/", loggerMiddleware, hello)

	server := &http.Server{
		Addr:    ":1323",
		Handler: bee,
	}

	_ = server.ListenAndServe()
}

func hello(_ *beehive.Context) beehive.Responder {
	return &beehive.DefaultResponder{
		Message: []byte("Hello, World!"),
		Status:  200,
	}
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
