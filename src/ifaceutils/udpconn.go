//go:build freebsd || darwin || windows

package ifaceutils

import (
	"net"
)

// In FreeBSD, MacOS and Windows, it's enough to listen on a particular address to bind to its interface
func BoundUDPConn(laddr *net.UDPAddr, ifname string) (*net.UDPConn, error) {
	return net.ListenUDP("udp", laddr)
}
