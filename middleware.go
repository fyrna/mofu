package mofu

import "net/http"

// Middleware is a function that wraps handlers
type Middleware func(Handler) Handler

// chain composes middlewares around a handler (last added runs first).
func chain(h Handler, mws []Middleware) Handler {
	if len(mws) == 0 {
		return h
	}
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// MwHug adapts native Mofu middleware.
// style:
//
//	func(*C) error
func MwHug(fn func(*C) error) Middleware {
	return func(next Handler) Handler {
		return func(c *C) error {
			prevNext := c.next
			prevAbort := c.aborted

			c.next = next
			c.aborted = false

			defer func() {
				c.next = prevNext
				c.aborted = prevAbort
			}() // restore

			return fn(c)
		}
	}
}

// MwHandler adapts standard http.Handler middleware.
// style:
//
// func(http.Handler) http.Handler
func MwHandler(adapt func(http.Handler) http.Handler) Middleware {
	return func(next Handler) Handler {
		return func(c *C) error {
			// inner handler will call next using existing context
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// call next with the same context (ignore returned error here)
				_ = next(c)
			})
			wrapped := adapt(inner)
			wrapped.ServeHTTP(c.Writer, c.Request)
			// Note: adapt may have already called next; return nil to continue
			return nil
		}
	}
}

// MwHandlerFunc adapts http.HandlerFunc-style middleware.
// style:
//
// func(http.ResponseWriter, *http.Request, http.HandlerFunc)
func MwHandlerFunc(fn func(http.ResponseWriter, *http.Request, http.HandlerFunc)) Middleware {
	return func(next Handler) Handler {
		return func(c *C) error {
			inner := func(w http.ResponseWriter, r *http.Request) {
				_ = next(c)
			}
			fn(c.Writer, c.Request, inner)
			return nil
		}
	}
}
