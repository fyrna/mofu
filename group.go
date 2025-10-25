package mofu

// Group represents a route group with common prefix and middleware.
type Group struct {
	router     *Router
	prefix     string
	middleware []Middleware
}

// Group creates route group.
func (r *Router) Group(prefix string) *Group {
	return &Group{
		router: r,
		prefix: normalize_path(prefix),
	}
}

// Group creates nested route group.
//
// Example:
//
// api := router.Group("/api")
// v1 := api.Group("/v1")
// v1.GET("/users", getUsers)
func (g *Group) Group(prefix string) *Group {
	return &Group{
		router:     g.router,
		prefix:     g.prefix + normalize_path(prefix),
		middleware: append([]Middleware(nil), g.middleware...),
	}
}

// Use adds middleware to group.
func (g *Group) Use(mw ...Middleware) {
	g.middleware = append(g.middleware, mw...)
}

// GET registers GET route.
func (g *Group) GET(path string, h Handler) {
	g.router.GET(g.prefix+path, g.wrap(h))
}

// POST registers POST route.
func (g *Group) POST(path string, h Handler) {
	g.router.POST(g.prefix+path, g.wrap(h))
}

// PUT registers PUT route.
func (g *Group) PUT(path string, h Handler) {
	g.router.PUT(g.prefix+path, g.wrap(h))
}

// DELETE registers DELETE route.
func (g *Group) DELETE(path string, h Handler) {
	g.router.DELETE(g.prefix+path, g.wrap(h))
}

// Handle registers route for specific HTTP method.
func (g *Group) Handle(method, path string, h Handler) {
	g.router.add(method, g.prefix+path, g.wrap(h))
}

func (g *Group) wrap(h Handler) Handler {
	if len(g.middleware) == 0 {
		return h
	}

	return chain(h, g.middleware)
}
