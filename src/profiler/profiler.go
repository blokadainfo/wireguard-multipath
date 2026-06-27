package profiler

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/config"
)

func ConfigureProfiler(ctx context.Context, cfg config.ProfilerConfig) {
	if !cfg.Enabled {
		return
	}

	srv := &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: nil,
	}

	go func() {
		<-ctx.Done()

		slog.Info("Stopping profiler")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("Failed to shut down profiler", "error", err)
		}
	}()

	go func() {
		slog.Info("Starting profiler", "address", cfg.ListenAddr)

		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start profiler: %v", err)
		}

		slog.Info("Profiler stopped")
	}()
}
