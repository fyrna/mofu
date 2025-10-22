package mofu

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

var ctxPool = sync.Pool{
	New: func() any {
		return &C{
			params: make(map[string]string),
			values: make(map[string]any),
		}
	},
}

type C struct {
	Writer  http.ResponseWriter
	Request *http.Request

	params map[string]string
	values map[string]any
}

func (c *C) Set(key string, val any) {
	c.values[key] = val
}

func (c *C) Get(key string) any {
	return c.values[key]
}

func (c *C) Status(code int) {
	c.Writer.WriteHeader(code)
}

func (c *C) SetHeader(k, v string) {
	c.Writer.Header().Set(k, v)
}

func (c *C) GetHeader(k string) string {
	return c.Request.Header.Get(k)
}

func (c *C) Param(name string) string {
	if c.params == nil {
		return ""
	}
	return c.params[name]
}

func (c *C) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

func (c *C) Form(name string) string {
	return c.Request.FormValue(name)
}

func (c *C) BindJSON(v any) error {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func (c *C) String(code int, s string) error {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(code)

	_, err := io.WriteString(c.Writer, s)
	return err
}

func (c *C) JSON(code int, v any) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)

	return json.NewEncoder(c.Writer).Encode(v)
}

func (c *C) HTML(code int, s string) error {
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(code)

	_, err := c.Writer.Write([]byte(s))
	return err
}

func (c *C) Redirect(url string, code ...int) {
	status := http.StatusFound
	if len(code) > 0 {
		status = code[0]
	}
	http.Redirect(c.Writer, c.Request, url, status)
}

// i love these names
func alloc(w http.ResponseWriter, r *http.Request) *C {
	c := ctxPool.Get().(*C)
	c.Writer, c.Request = w, r

	for k := range c.params {
		delete(c.params, k)
	}
	for k := range c.values {
		delete(c.values, k)
	}

	return c
}

func free(c *C) {
	for k := range c.params {
		delete(c.params, k)
	}
	for k := range c.values {
		delete(c.values, k)
	}

	c.Writer, c.Request = nil, nil
	ctxPool.Put(c)
}
