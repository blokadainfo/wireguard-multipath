package main

import (
	"context"
	"log/slog"
	_ "net/http/pprof"
	"os"
	"time"
	_ "time/tzdata" // TODO: This package will be automatically imported if you build with -tags timetzdata.

	"github.com/blokadainfo/wireguard-multipath/src/config"
	"github.com/blokadainfo/wireguard-multipath/src/ifaceutils"
	"github.com/blokadainfo/wireguard-multipath/src/logger"
	"github.com/blokadainfo/wireguard-multipath/src/profiler"
	"github.com/blokadainfo/wireguard-multipath/src/run/client"
	"github.com/blokadainfo/wireguard-multipath/src/run/server"
)

func main() {
	// Make a shared ctx used for stopping all goroutines
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// Setup logging
	logger.ConfigureDefaultLogger(config.LoadLogger())

	// Setup profiler
	profiler.ConfigureProfiler(ctx, config.LoadProfiler())

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "default")
	}

	switch os.Args[1] {
	case "client":
		client.Run(ctx, config.LoadClient())
	case "server":
		server.Run(ctx, config.LoadServer())
	case "list", "list-interfaces":
		ifaceutils.ListInterfaces()
	default:
		slog.Info("Use wireguard-multipath client/server to start or list/list-interfaces")
	}

	ctxCancel() // Cancel immediately so all goroutines clean up nicely
	time.Sleep(time.Second)
}
