package mofu

import (
	"net/http"
	"strings"
)

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

func (n *node) insert(path string, h HandlerFunc) {
	current := n
	for {
		seg, rest := nextSegment(path)

		// check existing children first
		var child *node
		if !current.hasWildcard && !current.hasCatchAll {
			child = current.findExactChild(seg)
		} else {
			child = current.findChild(seg)
		}

		if child == nil {
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

func nextSegment(path string) (string, string) {
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i], path[i+1:]
	}
	return path, ""
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

func (r *Router) PATCH(path string, h HandlerFunc) {
	r.add("PATCH", path, h)
}

func (r *Router) HEAD(path string, h HandlerFunc) {
	r.add("HEAD", path, h)
}

func (r *Router) OPTIONS(path string, h HandlerFunc) {
	r.add("OPTIONS", path, h)
}

// OnNotFound sets global 404 handler.
func (r *Router) OnNotFound(h HandlerFunc) {
	r.notFound = h
}

// Use adds middleware simple and compatible with net/http :3
func (r *Router) Use(mw func(http.Handler) http.Handler) {
	r.middleware = append(r.middleware, mw)
}
