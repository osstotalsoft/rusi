package diagnostics

import (
	"context"
	"fmt"
	"k8s.io/klog/v2"
	"net/http"
	"time"
)

func Run(ctx context.Context, port int, router http.Handler) error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	doneCh := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			klog.Info("Diagnostics server is shutting down")
			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*5,
			)
			defer cancel()
			srv.Shutdown(shutdownCtx) // nolint: errcheck
		case <-doneCh:
		}
	}()

	klog.Infof("Diagnostics server is listening on %s", srv.Addr)
	err := srv.ListenAndServe()
	close(doneCh)
	return err
}
