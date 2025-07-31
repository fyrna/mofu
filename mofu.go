// mofu, a http micro-framework
package mofu

import "net/http"

// Miaw returns a new Router instance.
func Miaw() *Router {
	return &Router{tree: new(node)}
}

// Router implements http.Handler.
type Router struct {
	tree       *node
	notFound   HandlerFunc
	middleware []func(http.Handler) http.Handler
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := r.handler(req)

	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}

	h.ServeHTTP(w, req)
}

type HandlerFunc func(*C) error

// handler wraps HandlerFunc into http.Handler.
func (r *Router) handler(req *http.Request) http.Handler {
	n, ps := r.tree.search(req.Method + req.URL.Path)

	if n == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			c := alloc(w, req)

			defer free(c)

			if r.notFound != nil {
				_ = r.notFound(c)
			} else {
				c.SendText(http.StatusNotFound, "404 page not found")
			}
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		c := alloc(w, req)
		c.params = ps

		defer free(c)

		_ = n.h(c)
	})
}
