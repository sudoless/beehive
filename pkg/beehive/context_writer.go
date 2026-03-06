package beehive

import (
	"go.sdls.io/beehive/internal/unsafe"
)

// Write forwards the write call to ResponseWriter.
func (c *Context) Write(b []byte) (int, error) {
	return c.ResponseWriter.Write(b)
}

// WriteString forwards the write call to ResponseWriter by converting the string to bytes.
func (c *Context) WriteString(s string) (int, error) {
	return c.ResponseWriter.Write(unsafe.StringToBytes(s))
}

// WriteHeader forwards the write call to ResponseWriter.
func (c *Context) WriteHeader(status int) {
	c.ResponseWriter.WriteHeader(status)
}

// WriteHeaders is a utility to write a series of headers before writing the status.
func (c *Context) WriteHeaders(status int, headers map[string]string) {
	wh := c.ResponseWriter.Header()
	for k, v := range headers {
		wh.Set(k, v)
	}

	c.ResponseWriter.WriteHeader(status)
}
