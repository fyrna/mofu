package mofu

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"
)

var (
	ctxPool = sync.Pool{
		New: func() any {
			return &C{
				params: make(map[string]string),
				values: make(map[string]any),
			}
		},
	}
)

type C struct {
	Writer  http.ResponseWriter
	Request *http.Request

	params  map[string]string
	values  map[string]any
	next    Handler
	aborted bool

	router *Router
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

func (c *C) String(code int, s string) error {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(code)

	_, err := io.WriteString(c.Writer, s)
	return err
}

// TODO: use this for templating stuff
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

func (c *C) Abort() { c.aborted = true }

func (c *C) Next() error {
	if c.aborted || c.next == nil {
		return nil
	}
	return c.next(c)
}

func (c *C) BindJSON(v any) error {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func (c *C) JSON(code int, v any) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)

	return json.NewEncoder(c.Writer).Encode(v)
}

func (c *C) Render(code int, name string, data any) error {
	if c.router == nil {
		return c.Error(http.StatusInternalServerError, "router not configured in context")
	}

	if c.router.config == nil {
		return c.Error(http.StatusInternalServerError, "router config not configured")
	}

	if c.router.config.Templating == nil {
		return c.Error(http.StatusInternalServerError, "template engine not configured")
	}

	// Lazy initialization template engine
	if c.router.config.templateEngine == nil {
		engine, err := c.router.config.Templating.CreateEngine()
		if err != nil {
			return c.Error(http.StatusInternalServerError, "failed to initialize template engine: "+err.Error())
		}
		c.router.config.templateEngine = engine
	}

	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(code)

	return c.router.config.templateEngine.Render(c.Writer, name, data)
}

// standard shortcut
// OK
//
//	{
//	    "success" : true,
//	    "data" : yourData,
//	}
func (c *C) OK(data any) error {
	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    data,
	})
}

// standard shortcut
// Error
//
//	{
//	    "success" : false,
//	    "error" : message,
//	}
func (c *C) Error(code int, message string) error {
	return c.JSON(code, map[string]any{
		"success": false,
		"error":   message,
	})
}

// i love these names
func alloc(router *Router, w http.ResponseWriter, r *http.Request) *C {
	c := ctxPool.Get().(*C)
	c.router = router
	c.Writer, c.Request = w, r
	c.aborted = false
	c.next = nil

	clear(c.params)
	clear(c.values)

	return c
}

func free(c *C) {
	clear(c.params)
	clear(c.values)

	c.router = nil
	c.Writer, c.Request = nil, nil
	c.next = nil
	c.aborted = false
	ctxPool.Put(c)
}
