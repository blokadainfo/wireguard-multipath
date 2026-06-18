package logger

import (
	"log/slog"
	"os"

	"github.com/blokadainfo/wireguard-multipath/src/config"
)

func ConfigureDefaultLogger(cfg config.LoggerConfig) {
	var opts *slog.HandlerOptions
	if cfg.Debug {
		opts = &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, opts))
	if cfg.JSON {
		l = slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}

	slog.SetDefault(l)
}
