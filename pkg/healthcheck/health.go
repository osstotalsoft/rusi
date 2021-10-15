package healthcheck

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type response struct {
	Status string            `json:"status,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
}

type health struct {
	checkers map[string]HealthChecker
	timeout  time.Duration
}

// Handler returns an http.Handler
func Handler(opts ...Option) http.Handler {
	h := &health{
		checkers: make(map[string]HealthChecker),
		timeout:  30 * time.Second,
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// HandlerFunc returns an http.HandlerFunc to mount the API implementation at a specific route
func HandlerFunc(opts ...Option) http.HandlerFunc {
	return Handler(opts...).ServeHTTP
}

// Option adds optional parameter for the HealthcheckHandlerFunc
type Option func(*health)

// WithChecker adds a status checker that needs to be added as part of healthcheck. i.e database, cache or any external dependency
func WithChecker(name string, s HealthChecker) Option {
	return func(h *health) {
		h.checkers[name] = &timeoutChecker{s}
	}
}

// WithTimeout configures the global timeout for all individual checkers.
func WithTimeout(timeout time.Duration) Option {
	return func(h *health) {
		h.timeout = timeout
	}
}

func (h *health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nCheckers := len(h.checkers)

	code := http.StatusOK
	errorMsgs := make(map[string]string, nCheckers)

	ctx, cancel := context.Background(), func() {}
	if h.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
	}
	defer cancel()

	var mutex sync.Mutex
	var wg sync.WaitGroup
	wg.Add(nCheckers)

	for key, checker := range h.checkers {
		go func(key string, checker HealthChecker) {
			if r := checker.IsHealthy(ctx); r.Status != Healthy {
				mutex.Lock()
				errorMsgs[key] = r.Description
				code = http.StatusServiceUnavailable
				mutex.Unlock()
			}
			wg.Done()
		}(key, checker)
	}

	wg.Wait()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response{
		Status: http.StatusText(code),
		Errors: errorMsgs,
	})
}

type timeoutChecker struct {
	checker HealthChecker
}

func (t *timeoutChecker) IsHealthy(ctx context.Context) HealthResult {
	checkerChan := make(chan HealthResult)
	go func() {
		checkerChan <- t.checker.IsHealthy(ctx)
	}()
	select {
	case r := <-checkerChan:
		return r
	case <-ctx.Done():
		return HealthResult{
			Status:      Unhealthy,
			Description: "max check time exceeded",
		}
	}
}
