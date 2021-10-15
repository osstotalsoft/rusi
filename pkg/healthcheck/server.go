package healthcheck

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

func Run(ctx context.Context, port int, options ...Option) error {
	router := http.NewServeMux()
	router.Handle("/healthz", HandlerFunc(options...))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	doneCh := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			klog.Info("Healthz server is shutting down")
			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*5,
			)
			defer cancel()
			srv.Shutdown(shutdownCtx) // nolint: errcheck
		case <-doneCh:
		}
	}()

	klog.Infof("Healthz server is listening on %s", srv.Addr)
	err := srv.ListenAndServe()
	if err != http.ErrServerClosed {
		klog.ErrorS(err, "Healthz server error")
	}
	close(doneCh)
	return err
}
