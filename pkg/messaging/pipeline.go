package messaging

import (
	"context"
)

type RequestHandler func(ctx context.Context, msg *MessageEnvelope)
type Middleware func(next RequestHandler) RequestHandler

type Pipeline struct {
	Middlewares []Middleware
}

func (p Pipeline) Build(handler RequestHandler) RequestHandler {
	for i := len(p.Middlewares) - 1; i >= 0; i-- {
		handler = p.Middlewares[i](handler)
	}
	return handler
}

func (p *Pipeline) UseMiddleware(middleware Middleware) {
	p.Middlewares = append(p.Middlewares, middleware)
}
