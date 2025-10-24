// mofu, a http micro-framework
package mofu

import (
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
	segment    string
	paramName  string
	paramName2 string
	prefix     string
	multi      bool
	delimiter  string
	wildcard   bool
	catchAll   bool

	handler  HandlerFunc
	children []*node

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
		seg, rest := next_segment(path)

		prefix, p1, p2, multi, delimiter, kind := analyze_segment(seg)

		if p1 != "" {
			if paramNames[p1] {
				panic("duplicate parameter name: " + p1)
			}
			paramNames[p1] = true
		}
		if p2 != "" {
			if paramNames[p2] {
				panic("duplicate parameter name: " + p2)
			}
			paramNames[p2] = true
		}

		// Check existing children first
		var child *node
		if !current.hasWildcard && !current.hasCatchAll {
			child = current.findExactChild(seg)
		} else {
			child = current.findChild(seg)
		}

		if child == nil {
			// Validate constraints
			if (kind == '*' || kind == '+') && rest != "" {
				panic("catch-all must be last segment")
			}

			if kind != 0 { // wildcard segment
				for _, existing := range current.children {
					if existing.wildcard {
						panic("multiple wildcards at same level: " + existing.segment + " and " + seg)
					}
				}
			}

			child = &node{
				segment:    seg,
				paramName:  p1,
				paramName2: p2,
				prefix:     prefix,
				multi:      multi,
				delimiter:  delimiter,
				wildcard:   kind != 0, // any kind of parameter
				catchAll:   kind == '*',
			}

			if child.wildcard {
				current.hasWildcard = true
			}
			if child.catchAll {
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

func (n *node) search(actualpath string) (*node, map[string]string) {
	path := normalize_path(actualpath)
	params := make(map[string]string, 4)
	current := n

	for {
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}

		if path == "" {
			if current.handler != nil {
				return current, params
			}
			break
		}

		seg, rest := next_segment(path)

		// fast path: exact match first
		if child := current.findExactChild(seg); child != nil {
			current = child
			path = rest
			continue
		}

		// check catch-all first (highest priority)
		if current.hasCatchAll {
			if child := current.findCatchAllChild(); child != nil {
				params[child.paramName] = path // entire remaining path
				return child, params
			}
		}

		// Check wildcards
		if current.hasWildcard {
			found := false
			for _, child := range current.children {
				if child.wildcard && !child.catchAll {
					switch {
					case child.prefix != "": // Prefix parameter ::filename
						if pre, ok := strings.CutPrefix(seg, child.prefix); ok {
							params[child.paramName] = pre
							current = child
							path = rest
							found = true
						}

					case child.paramName2 != "": // Double parameter :from-:to
						if dash := strings.IndexByte(seg, '-'); dash > 0 {
							params[child.paramName] = seg[:dash]
							params[child.paramName2] = seg[dash+1:]
							current = child
							path = rest
							found = true
						}

					case child.multi && child.delimiter != "": // Multiple values :tags(,)
						params[child.paramName] = seg
						current = child
						path = rest
						found = true

					default: // Single parameter :name
						params[child.paramName] = seg
						current = child
						path = rest
						found = true
					}
				}
				if found {
					break
				}
			}
			if found {
				continue
			}
		}

		// No match found
		break
	}

	return nil, params
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

func (n *node) findExactChild(seg string) *node {
	for _, child := range n.children {
		if !child.wildcard && !child.catchAll && child.segment == seg {
			return child
		}
	}
	return nil
}

func analyze_segment(seg string) (
	prefix, p1, p2 string,
	multi bool,
	delimiter string,
	kind byte,
) {
	// Catch-all *
	if seg == "*" {
		return "", "*", "", false, "", '*'
	}

	// Multi-segment +
	if seg == "+" {
		return "", "+", "", true, "", '+'
	}

	// Prefix parameter ::filename
	if strings.Contains(seg, "::") {
		parts := strings.SplitN(seg, "::", 2)
		prefix := parts[0]
		param := parts[1]
		return prefix + ":", param, "", false, "", 'p'
	}

	// Multiple values dengan delimiter :tags(,)
	if strings.Contains(seg, "(") && strings.HasSuffix(seg, ")") {
		idx := strings.Index(seg, "(")
		param := seg[1:idx]              // :type -> type
		delim := seg[idx+1 : len(seg)-1] // (,) -> ,
		return "", param, "", true, delim, ','
	}

	// Double parameter :from-:to
	if strings.Count(seg, ":") == 2 && strings.Contains(seg, "-") {
		parts := strings.SplitN(seg, "-", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], ":") && strings.HasPrefix(parts[1], ":") {
			p1 = parts[0][1:]
			p2 = parts[1][1:]
			kind = '-'
			return
		}
	}

	// Single parameter :param
	if strings.HasPrefix(seg, ":") {
		p1 = seg[1:]
		kind = ':'
		return
	}

	// static segment
	return "", "", "", false, "", 0
}

func next_segment(path string) (string, string) {
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
	if path != "/" && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}
