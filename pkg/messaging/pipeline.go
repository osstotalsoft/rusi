package messaging

type Middleware func(next Handler) Handler

type Pipeline struct {
	Middlewares []Middleware
}

func (p Pipeline) Build(handler Handler) Handler {
	for i := len(p.Middlewares) - 1; i >= 0; i-- {
		handler = p.Middlewares[i](handler)
	}
	return handler
}

func (p *Pipeline) UseMiddleware(middleware Middleware) {
	p.Middlewares = append(p.Middlewares, middleware)
}
