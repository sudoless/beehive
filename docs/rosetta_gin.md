<table>
<tr>
<th>Gin</th>
<th>Beehive</th>
</tr>
<tr>
<td>

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
```

</td>
<td>

```go
package main

import (
	"net/http"

	"github.com/sudoless/beehive/pkg/beehive"
)


func main() {
	bee := beehive.NewRouter()

	bee.Handle("GET", "/ping", func(ctx *beehive.Context) beehive.Responder {
		h := ctx.ResponseWriter.Header()
		h.Set("Content-Type", "application/json")

		return &beehive.DefaultResponder{
			Message: []byte(`{"message": "pong"}`),
			Status:  200,
		}
	})

	server := &http.Server{
		Addr:    ":8080",
		Handler: bee,
	}

	_ = server.ListenAndServe()
}
```

</td>
</tr>
</table>
