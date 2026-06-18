package main

import (
	"log/slog"
	"os"
	_ "time/tzdata" // TODO: This package will be automatically imported if you build with -tags timetzdata.

	"github.com/blokadainfo/wireguard-multipath/src/config"
	"github.com/blokadainfo/wireguard-multipath/src/ifaceutils"
	"github.com/blokadainfo/wireguard-multipath/src/logger"
	"github.com/blokadainfo/wireguard-multipath/src/run/client"
	"github.com/blokadainfo/wireguard-multipath/src/run/server"
)

func main() {
	logger.ConfigureDefaultLogger(config.LoadLogger())

	if len(os.Args) == 1 {
		os.Args = append(os.Args, "default")
	}

	switch os.Args[1] {
	case "client":
		client.Run(config.LoadClient())
	case "server":
		server.Run(config.LoadServer())
	case "list", "list-interfaces":
		ifaceutils.ListInterfaces()
	default:
		slog.Info("Use wireguard-multipath client/server to start or list/list-interfaces")
	}
}
