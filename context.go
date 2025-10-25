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

// C represents the request context, providing methods to access request data, set responses, and manage flow.
//
// Example:
//
//	func handler(c *mofu.C) error {
//	    name := c.Param("name")
//	    return c.String(200, "Hello " + name)
//	}
type C struct {
	Writer  http.ResponseWriter
	Request *http.Request

	params  map[string]string
	values  map[string]any
	next    Handler
	aborted bool

	router *Router
}

// Set stores a value in context.
func (c *C) Set(key string, val any) {
	c.values[key] = val
}

// Get retrieves a value stored in context.
func (c *C) Get(key string) any {
	return c.values[key]
}

// SetHeader sets response header.
func (c *C) SetHeader(k, v string) {
	c.Writer.Header().Set(k, v)
}

// GetHeader returns request header value.
func (c *C) GetHeader(k string) string {
	return c.Request.Header.Get(k)
}

// Param returns URL parameter value.
func (c *C) Param(name string) string {
	if c.params == nil {
		return ""
	}
	return c.params[name]
}

// Query returns URL query parameter value.
func (c *C) Query(key string) string {
	return c.Request.URL.Query().Get(key)
}

// Form returns form value by name.
func (c *C) FormValue(name string) string {
	return c.Request.FormValue(name)
}

// Status sets response status code.
func (c *C) Status(code int) {
	c.Writer.WriteHeader(code)
}

// String sends plain text response.
func (c *C) String(code int, s string) error {
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteHeader(code)

	_, err := io.WriteString(c.Writer, s)
	return err
}

// func (c *C) HTML(code int, s string) error {
// 	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
// 	c.Writer.WriteHeader(code)

// 	_, err := c.Writer.Write([]byte(s))
// 	return err
// }

// BindJSON parses the request body as JSON into the given struct.
//
// Example:
//
//	type User struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	func createUser(c *mofu.C) error {
//	    var user User
//	    if err := c.BindJSON(&user); err != nil {
//	        return c.Error(400, "Invalid JSON")
//	    }
//	    return c.OK(user)
//	}
func (c *C) BindJSON(v any) error {
	defer c.Request.Body.Close()
	return json.NewDecoder(c.Request.Body).Decode(v)
}

// JSON sends JSON response.
func (c *C) JSON(code int, v any) error {
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(code)

	return json.NewEncoder(c.Writer).Encode(v)
}

// Render renders a template with data.
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

// OK sends a standardized success JSON response.
//
// Example Response:
//
//	{
//	    "success": true,
//	    "data": data
//	}
func (c *C) OK(data any) error {
	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    data,
	})
}

// Error sends a standardized JSON error response
//
// Example Response:
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

// Abort stops the execution of subsequent middleware and handlers.
func (c *C) Abort() {
	c.aborted = true
}

// Next executes the next handler in the chain.
func (c *C) Next() error {
	if c.aborted || c.next == nil {
		return nil
	}
	return c.next(c)
}

// Redirect redirects the request.
func (c *C) Redirect(url string, code ...int) {
	status := http.StatusFound
	if len(code) > 0 {
		status = code[0]
	}
	http.Redirect(c.Writer, c.Request, url, status)
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
