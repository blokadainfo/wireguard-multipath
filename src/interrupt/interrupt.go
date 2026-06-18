package interrupt

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// Blocks until CTRL+C (os.Interrupt or syscall.SIGTERM) is received
func Wait(ctx context.Context) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case <-c:
		slog.Info("Received an interrupt, stopping")
	case <-ctx.Done():
		slog.Info("Context is done, stopping")
	}
}
