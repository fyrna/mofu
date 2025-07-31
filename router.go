package mofu

import (
	"net/http"
	"strings"
)

type node struct {
	seg   string
	wild  bool
	catch bool
	h     HandlerFunc
	child []*node
}

func (n *node) insert(path string, h HandlerFunc) {
	cur := n
	for {
		idx := strings.IndexByte(path, '/')
		seg := path

		if idx > 0 {
			seg = path[:idx]
		}

		child := cur.childBySeg(seg)

		if child == nil {
			child = &node{seg: seg}

			if strings.HasPrefix(seg, ":") {
				child.wild = true
			} else if seg == "*" {
				child.catch = true
			}

			cur.child = append(cur.child, child)
		}

		if idx < 0 {
			child.h = h
			return
		}

		cur = child
		path = path[idx:]
	}
}

func (n *node) search(path string) (*node, map[string]string) {
	var (
		cur    = n
		params = map[string]string{}
	)

	for cur != nil && path != "" {
		idx := strings.IndexByte(path, '/')
		seg := path

		if idx > 0 {
			seg = path[:idx]
		}

		var next *node

		for _, c := range cur.child {
			switch {
			case c.wild:
				params[c.seg[1:]] = seg
				next = c
			case c.catch:
				params["*"] = strings.TrimPrefix(path, "/")
				return c, params
			case c.seg == seg:
				next = c
			}
		}

		if next == nil {
			return nil, nil
		}

		cur = next

		if idx >= 0 {
			path = path[idx:]
		} else {
			break
		}
	}

	if cur != nil && cur.h != nil {
		return cur, params
	}

	return nil, nil
}

func (n *node) childBySeg(seg string) *node {
	for _, c := range n.child {
		if c.seg == seg || c.wild || c.catch {
			return c
		}
	}
	return nil
}

func (r *Router) add(method, path string, h HandlerFunc) {
	r.tree.insert(method+path, h)
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

// OnNotFound sets global 404 handler.
func (r *Router) OnNotFound(h HandlerFunc) {
	r.notFound = h
}

// Use adds middleware simple and compatible with net/http :3
func (r *Router) Use(mw func(http.Handler) http.Handler) {
	r.middleware = append(r.middleware, mw)
}
