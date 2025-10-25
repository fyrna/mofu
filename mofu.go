// mofu, a http micro-framework
package mofu

import (
	"log"
	"maps"
	"net/http"
	"strings"
)

type param byte

const (
	paramStatic       param = iota
	paramSingle             // :name
	paramDouble             // :x-:y
	paramPrefix             // ::rawr
	paramMulti              // :tags(,)
	paramCatchAll           // *
	paramMultiSegment       // +
)

type Handler func(*C) error

type Config struct {
	Templating TemplateConfig

	templateEngine TemplateEngine
}

// Router implements http.Handler.
type Router struct {
	tree       *node
	notFound   Handler
	middleware []Middleware

	config *Config
}

type node struct {
	segment string
	kind    param

	paramName  string
	paramName2 string
	prefix     string
	delimiter  string

	handler  Handler
	children []*node
}

// Miaw returns a new Router instance.
func Miaw(c ...*Config) *Router {
	r := &Router{tree: new(node)}

	if len(c) > 0 && c[0] != nil {
		r.config = c[0]
	} else {
		r.config = &Config{}
	}

	return r
}

func (r *Router) GET(path string, h Handler) {
	r.add(http.MethodGet, path, h)
}

func (r *Router) POST(path string, h Handler) {
	r.add(http.MethodPost, path, h)
}

func (r *Router) PUT(path string, h Handler) {
	r.add(http.MethodPut, path, h)
}

func (r *Router) DELETE(path string, h Handler) {
	r.add(http.MethodDelete, path, h)
}

func (r *Router) Handle(method, path string, h Handler) {
	r.add(method, path, h)
}

// OnNotFound sets global 404 handler.
func (r *Router) OnNotFound(h Handler) {
	r.notFound = h
}

// Use adds middleware compatible with net/http
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

	// Apply middleware in reverse order
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}

	c := alloc(r, w, req)
	defer free(c)
	_ = h(c)
}

func (r *Router) add(method, path string, h Handler) {
	fullPath := method + normalize_path(path)
	r.tree.insert(fullPath, h)
}

// handler finds the appropriate handler for the request
func (r *Router) handler(req *http.Request) Handler {
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

func (n *node) insert(path string, h Handler) {
	current := n
	paramNames := make(map[string]bool)

	for {
		seg, rest := next_segment(path)

		kind, p1, p2, prefix, delim := analyze_segment(seg)

		// Validate parameter names for duplicates
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

		// Validate catch-all/multi-segment must be last
		if (kind == paramCatchAll || kind == paramMultiSegment) && rest != "" {
			panic("catch-all/multi-segment must be last segment")
		}

		// Find or create child node
		child := current.findChildBySegment(seg)
		if child == nil {
			child = &node{
				segment:    seg,
				kind:       kind,
				paramName:  p1,
				paramName2: p2,
				prefix:     prefix,
				delimiter:  delim,
			}

			// Prevent multiple wildcards at same level
			if kind != paramStatic {
				for _, existing := range current.children {
					if existing.kind != paramStatic && existing.kind != paramCatchAll {
						panic("multiple wildcards at same level: " + existing.segment + " and " + seg)
					}
				}
			}

			current.children = append(current.children, child)
		}

		if rest == "" {
			child.handler = h
			return
		}

		if kind == paramCatchAll || kind == paramMultiSegment {
			panic("catch-all/multi-segment must be last segment")
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

		// Search priority:
		// 1. Exact static match first
		if child := current.findExactChild(seg); child != nil {
			current = child
			path = rest
			continue
		}

		// 2. Parameter wildcards (excluding catch-all)
		if child, childParams := current.findParamChild(seg); child != nil {
			// Merge parameters
			maps.Copy(params, childParams)
			current = child
			path = rest
			continue
		}

		// 3. Multi-segment
		if child := current.findMultiSegmentChild(); child != nil {
			params[child.paramName] = path
			return child, params
		}

		// 4. Catch-all (lowest priority)
		if child := current.findCatchAllChild(); child != nil {
			params[child.paramName] = path // entire remaining path
			return child, params
		}

		// No match found
		break
	}

	return nil, params
}

func (n *node) matchSegment(seg string) map[string]string {
	switch n.kind {
	case paramSingle:
		return map[string]string{n.paramName: seg}

	case paramDouble:
		parts := strings.Split(seg, n.delimiter)
		if len(parts) == 2 {
			return map[string]string{
				n.paramName:  parts[0],
				n.paramName2: parts[1],
			}
		}
		return nil

	case paramPrefix:
		if strings.HasPrefix(seg, n.prefix) {
			return map[string]string{
				n.paramName: seg[len(n.prefix):],
			}
		}
		return nil

	case paramMulti:
		// Validate delimiter exists in segment
		if n.delimiter != "" && strings.Contains(seg, n.delimiter) {
			return map[string]string{n.paramName: seg}
		}
		return nil
	}

	return nil
}

func (n *node) findParamChild(seg string) (*node, map[string]string) {
	for _, child := range n.children {
		// Skip multi-segment and catch-all, theyre handled separately
		if child.kind == paramMultiSegment || child.kind == paramCatchAll {
			continue
		}

		if !child.isWildcard() {
			continue
		}

		params := child.matchSegment(seg)
		if params != nil {
			return child, params
		}
	}
	return nil, nil
}

func (n *node) findChildBySegment(seg string) *node {
	for _, child := range n.children {
		if child.segment == seg {
			return child
		}
	}
	return nil
}

func (n *node) findExactChild(seg string) *node {
	for _, child := range n.children {
		if child.kind == paramStatic && child.segment == seg {
			return child
		}
	}
	return nil
}

func (n *node) findMultiSegmentChild() *node {
	for _, child := range n.children {
		if child.kind == paramMultiSegment {
			return child
		}
	}
	return nil
}

func (n *node) findCatchAllChild() *node {
	for _, child := range n.children {
		if child.kind == paramCatchAll {
			return child
		}
	}
	return nil
}

func (n *node) isWildcard() bool {
	return n.kind != paramStatic
}

// analyze_segment determines the parameter type and extracts components
func analyze_segment(seg string) (kind param, p1, p2, prefix, delim string) {
	// Catch-all *
	if seg == "*" {
		return paramCatchAll, "*", "", "", ""
	}

	// Multi-segment +
	if seg == "+" {
		return paramMultiSegment, "+", "", "", ""
	}

	// Prefix parameter ::filename
	if strings.HasPrefix(seg, "::") {
		// ::filename -> prefix = "", param = "filename"
		return paramPrefix, seg[2:], "", "", ""
	}

	// Prefix with separator (e.g., img::id)
	if idx := strings.Index(seg, "::"); idx > 0 {
		return paramPrefix, seg[idx+2:], "", seg[:idx+1], "" // "img:" as prefix
	}

	// Multiple values with delimiter :tags(,)
	if strings.HasPrefix(seg, ":") && strings.Contains(seg, "(") && strings.HasSuffix(seg, ")") {
		idx := strings.Index(seg, "(")
		param := seg[1:idx]
		delim := seg[idx+1 : len(seg)-1]
		return paramMulti, param, "", "", delim
	}

	// Double parameter with various delimiters
	if strings.HasPrefix(seg, ":") {
		// Check for common delimiters (-, _, ., ~)
		for _, d := range []string{"-", "_", ".", "~"} {
			if parts := strings.Split(seg, d); len(parts) == 2 {
				if strings.HasPrefix(parts[0], ":") && strings.HasPrefix(parts[1], ":") {
					return paramDouble, parts[0][1:], parts[1][1:], "", d
				}
			}
		}
	}

	// Single parameter :param
	if strings.HasPrefix(seg, ":") {
		return paramSingle, seg[1:], "", "", ""
	}

	// Static segment
	return paramStatic, "", "", "", ""
}

// next_segment splits path into current segment and remainder
func next_segment(path string) (string, string) {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i], path[i+1:]
	}
	return path, ""
}

// normalize_path ensures consistent path formatting
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
