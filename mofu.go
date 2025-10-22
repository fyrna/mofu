// mofu, a http micro-framework
package mofu

import (
	"bytes"
	"net/http"
	"strings"
)

// Router implements http.Handler.
type Router struct {
	tree       *node
	notFound   HandlerFunc
	middleware []func(http.Handler) http.Handler
}

type node struct {
	segment  string // immutable after creation
	wildcard bool   // :
	catchAll bool   // *
	handler  HandlerFunc
	children []*node

	// pre-scan hints, gotta-go-fast
	hasWildcard bool
	hasCatchAll bool
}

type HandlerFunc func(*C) error

// Miaw returns a new Router instance.
func Miaw() *Router {
	return &Router{tree: new(node)}
}

func (r *Router) GET(path string, h HandlerFunc) {
	r.add("GET", path, h)
}

func (r *Router) POST(path string, h HandlerFunc) {
	r.add("POST", path, h)
}

func (r *Router) PUT(path string, h HandlerFunc) {
	r.add("PUT", path, h)
}

func (r *Router) DELETE(path string, h HandlerFunc) {
	r.add("DELETE", path, h)
}

func (r *Router) Handle(method, path string, h HandlerFunc) {
	r.add(method, path, h)
}

// OnNotFound sets global 404 handler.
func (r *Router) OnNotFound(h HandlerFunc) {
	r.notFound = h
}

// Use adds middleware simple and compatible with net/http :3
func (r *Router) Use(mw func(http.Handler) http.Handler) {
	r.middleware = append(r.middleware, mw)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := r.handler(req)
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}
	h.ServeHTTP(w, req)
}

func (r *Router) add(method, path string, h HandlerFunc) {
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}

	fullPath := method + path
	r.tree.insert(fullPath, h)
}

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
				c.String(http.StatusNotFound, "404 page not found")
			}
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		c := alloc(w, req)
		c.params = ps

		defer free(c)

		_ = n.handler(c)
	})
}

type Group struct {
	router     *Router
	prefix     string
	middleware []func(http.Handler) http.Handler
}

func (r *Router) Group(prefix string) *Group {
	if prefix == "" {
		prefix = "/"
	}

	if prefix[0] != '/' {
		prefix = "/" + prefix
	}

	if prefix != "/" && prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}

	return &Group{
		router: r,
		prefix: prefix,
	}
}

func (g *Group) Group(prefix string) *Group {
	if prefix == "" {
		prefix = "/"
	}

	if prefix[0] != '/' {
		prefix = "/" + prefix
	}

	if prefix != "/" && prefix[len(prefix)-1] == '/' {
		prefix = prefix[:len(prefix)-1]
	}

	fullPrefix := g.prefix + prefix
	if fullPrefix == "" {
		fullPrefix = "/"
	}

	return &Group{
		router:     g.router,
		prefix:     fullPrefix,
		middleware: append([]func(http.Handler) http.Handler{}, g.middleware...),
	}
}

func (g *Group) Use(mw func(http.Handler) http.Handler) {
	g.middleware = append(g.middleware, mw)
}

func (g *Group) GET(path string, h HandlerFunc) {
	g.router.GET(g.prefix+path, g.wrap(h))
}

func (g *Group) POST(path string, h HandlerFunc) {
	g.router.POST(g.prefix+path, g.wrap(h))
}

func (g *Group) PUT(path string, h HandlerFunc) {
	g.router.PUT(g.prefix+path, g.wrap(h))
}

func (g *Group) DELETE(path string, h HandlerFunc) {
	g.router.DELETE(g.prefix+path, g.wrap(h))
}

func (g *Group) Handle(method, path string, h HandlerFunc) {
	g.router.add(method, g.prefix+path, g.wrap(h))
}

func (g *Group) wrap(h HandlerFunc) HandlerFunc {
	if len(g.middleware) == 0 {
		return h
	}

	// Convert HandlerFunc to http.Handler
	var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		c := alloc(w, req)
		defer free(c)
		_ = h(c)
	})

	// Apply middleware in reverse order
	for i := len(g.middleware) - 1; i >= 0; i-- {
		handler = g.middleware[i](handler)
	}

	// Convert back to HandlerFunc
	return func(c *C) error {
		handler.ServeHTTP(c.Writer, c.Req)
		return nil
	}
}

func (n *node) insert(path string, h HandlerFunc) {
	current := n
	paramNames := make(map[string]bool)

	for {
		seg, rest := nextSegment(path)

		if bytes.HasPrefix([]byte(seg), []byte(":")) {
			paramName := seg[1:]
			if paramName == "" {
				panic("empy parameter name")
			}
			if paramNames[paramName] {
				panic("duplicate parameter name: " + paramName)
			}
			paramNames[paramName] = true
		}

		// check existing children first
		var child *node
		if !current.hasWildcard && !current.hasCatchAll {
			child = current.findExactChild(seg)
		} else {
			child = current.findChild(seg)
		}

		if child == nil {
			// validate
			if seg == "*" && rest != "" {
				panic("catch-all must be last segment")
			}

			if bytes.HasPrefix([]byte(seg), []byte(":")) {
				for _, existing := range current.children {
					if existing.wildcard {
						panic("multiple wildcards at same level: " + existing.segment + " and " + seg)
					}
				}
			}

			child = &node{
				segment:  seg,
				wildcard: seg[0] == ':',
				catchAll: seg == "*",
			}

			// update parent hints
			if child.wildcard {
				current.hasWildcard = true
			} else if child.catchAll {
				current.hasCatchAll = true
			}
			current.children = append(current.children, child)
		}

		if rest == "" {
			child.handler = h
			return
		}

		if child.catchAll {
			panic("catch-all must be last segment")
		}

		current = child
		path = rest
	}
}

func (n *node) search(path string) (*node, map[string]string) {
	params := make(map[string]string, 2) // pre-allocate for common cases
	current := n

	for {
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
			continue
		}

		seg, rest := nextSegment(path)

		// fast path: exact match first
		if child := current.findExactChild(seg); child != nil {
			current = child
			path = rest
			if rest == "" {
				if current.handler != nil {
					return current, params
				}
				return nil, nil
			}
			continue
		}

		// only check wildcards if they exist
		if current.hasWildcard {
			if child := current.findWildcardChild(); child != nil {
				params[child.segment[1:]] = seg
				current = child
				path = rest
				if rest == "" {
					if current.handler != nil {
						return current, params
					}
					return nil, nil
				}
				continue
			}
		}

		// catch-all shortcut
		if current.hasCatchAll {
			if child := current.findCatchAllChild(); child != nil {
				params["*"] = path
				return child, params
			}
		}

		return nil, nil
	}
}

func (n *node) findWildcardChild() *node {
	for _, child := range n.children {
		if child.wildcard {
			return child
		}
	}
	return nil
}

func (n *node) findCatchAllChild() *node {
	for _, child := range n.children {
		if child.catchAll {
			return child
		}
	}
	return nil
}

func (n *node) findChild(seg string) *node {
	for _, child := range n.children {
		if child.segment == seg {
			return child
		}
	}
	return nil
}

// findExactChild uses direct comparison
func (n *node) findExactChild(seg string) *node {
	for _, child := range n.children {
		if !child.wildcard && !child.catchAll && child.segment == seg {
			return child
		}
	}
	return nil
}

func nextSegment(path string) (string, string) {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i], path[i+1:]
	}
	return path, ""
}
