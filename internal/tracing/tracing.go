package tracing

import (
	"go.opentelemetry.io/otel"
)

func SetDefaultTracerProvider(url string) error {
	tp, err := TracerProvider(url)
	if err != nil {
		return err
	}
	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)
	return nil
}
