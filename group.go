package mofu

type Group struct {
	router     *Router
	prefix     string
	middleware []Middleware
}

func (r *Router) Group(prefix string) *Group {
	return &Group{
		router: r,
		prefix: normalize_path(prefix),
	}
}

func (g *Group) Group(prefix string) *Group {
	return &Group{
		router:     g.router,
		prefix:     g.prefix + normalize_path(prefix),
		middleware: append([]Middleware(nil), g.middleware...),
	}
}

func (g *Group) Use(mw ...Middleware) {
	g.middleware = append(g.middleware, mw...)
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

	return chain(h, g.middleware)
}
