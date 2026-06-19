package config

import (
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/envutils"
	"github.com/blokadainfo/wireguard-multipath/src/glob"
)

type LoggerConfig struct {
	Debug bool // WGMP_LOGGER_DEBUG
	JSON  bool // WGMP_LOGGER_JSON
}

type ClientConfig struct {
	ListenAddr         string        // WGMP_CLIENT_LISTEN_ADDR
	ServerAddr         string        // WGMP_CLIENT_SERVER_ADDR
	SocketWriteTimeout time.Duration // WGMP_CLIENT_SOCKET_WRITE_TIMEOUT_MS
	ExcludedInterfaces []glob.Glob   // WGMP_CLIENT_EXCLUDED_INTERFACES
}

type ServerConfig struct {
	ListenAddr         string        // WGMP_SERVER_LISTEN_ADDR
	WireguardAddr      string        // WGMP_SERVER_WIREGUARD_ADDR
	SocketWriteTimeout time.Duration // WGMP_SERVER_SOCKET_WRITE_TIMEOUT_MS
}

func LoadLogger() LoggerConfig {
	return LoggerConfig{
		Debug: envutils.GetBool("WGMP_LOGGER_DEBUG", false),
		JSON:  envutils.GetBool("WGMP_LOGGER_JSON", false),
	}
}

func LoadClient() ClientConfig {
	return ClientConfig{
		ListenAddr:         envutils.GetString("WGMP_CLIENT_LISTEN_ADDR", "127.0.0.1:59401"),
		ServerAddr:         envutils.GetString("WGMP_CLIENT_SERVER_ADDR"), // REQUIRED
		SocketWriteTimeout: envutils.GetTimeDuration(time.Millisecond, "WGMP_CLIENT_SOCKET_WRITE_TIMEOUT_MS", 10*time.Millisecond),
		ExcludedInterfaces: envutils.GetGlobList("WGMP_CLIENT_EXCLUDED_INTERFACES", []glob.Glob{glob.MustCompile("wg*"), glob.MustCompile("vlan*")}),
	}
}

func LoadServer() ServerConfig {
	return ServerConfig{
		ListenAddr:         envutils.GetString("WGMP_SERVER_LISTEN_ADDR", "0.0.0.0:59501"),
		WireguardAddr:      envutils.GetString("WGMP_SERVER_WIREGUARD_ADDR"), // REQUIRED
		SocketWriteTimeout: envutils.GetTimeDuration(time.Millisecond, "WGMP_SERVER_SOCKET_WRITE_TIMEOUT_MS", 10*time.Millisecond),
	}
}
