package healthcheck

import "context"

type HealthChecker interface {
	IsHealthy(ctx context.Context) HealthResult
}

// CheckerFunc is a convenience type to create functions that implement the HealthChecker interface.
type CheckerFunc func(ctx context.Context) HealthResult

// IsHealthy Implements the HealthChecker interface to allow for any func() HealthResult method
// to be passed as a HealthChecker
func (c CheckerFunc) IsHealthy(ctx context.Context) HealthResult {
	return c(ctx)
}

type HealthStatus int

const (
	Unhealthy HealthStatus = 0
	Degraded               = 1
	Healthy                = 2
)

type HealthResult struct {
	Status      HealthStatus
	Description string
}

var HealthyResult = HealthResult{Status: Healthy}
