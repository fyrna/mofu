// mofu, a http micro-framework
package mofu

import (
	"bytes"
	"log"
	"net/http"
	"strings"
)

// Router implements http.Handler.
type Router struct {
	tree       *node
	notFound   HandlerFunc
	middleware []Middleware
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

// Miaw returns a new Router instance.
func Miaw() *Router {
	return &Router{tree: new(node)}
}

func (r *Router) GET(path string, h HandlerFunc) {
	r.add(http.MethodGet, path, h)
}

func (r *Router) POST(path string, h HandlerFunc) {
	r.add(http.MethodPost, path, h)
}

func (r *Router) PUT(path string, h HandlerFunc) {
	r.add(http.MethodPut, path, h)
}

func (r *Router) DELETE(path string, h HandlerFunc) {
	r.add(http.MethodDelete, path, h)
}

func (r *Router) Handle(method, path string, h HandlerFunc) {
	r.add(method, path, h)
}

// OnNotFound sets global 404 handler.
func (r *Router) OnNotFound(h HandlerFunc) {
	r.notFound = h
}

// Use adds middleware simple and compatible with net/http :3
func (r *Router) Use(mws ...Middleware) {
	r.middleware = append([]Middleware(nil), mws...)
}

func (r *Router) Start(addr string) error {
	log.Printf("mofu listening on %s\n", addr)
	return http.ListenAndServe(addr, r)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	h := r.handler(req)

	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}

	c := alloc(w, req)
	defer free(c)
	_ = h(c)
}

func (r *Router) add(method, path string, h HandlerFunc) {
	fullPath := method + normalize_path(path)
	r.tree.insert(fullPath, h)
}

// handler wraps HandlerFunc into http.Handler.
func (r *Router) handler(req *http.Request) HandlerFunc {
	n, ps := r.tree.search(req.Method + req.URL.Path)

	if n == nil {
		return func(c *C) error {
			if r.notFound != nil {
				return r.notFound(c)
			}
			return c.String(http.StatusNotFound, "404 page not found")
		}
	}

	handlerFunc := n.handler
	return func(c *C) error {
		c.params = ps
		return handlerFunc(c)
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

func normalize_path(path string) string {
	if path == "" {
		path = "/"
	}
	if path[0] != '/' {
		path = "/" + path
	}
	return path
}

// eum, i dont know actually, but i works though
func normalize_prefix(prefix string) string {
	p := normalize_path(prefix)
	if p != "/" && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	return p
}
