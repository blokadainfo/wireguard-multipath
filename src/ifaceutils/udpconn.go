//go:build windows || darwin

package ifaceutils

import (
	"net"
)

// In Windows, it's enough to listen on a particular address to bind to its interface
func BoundUdpConn(laddr *net.UDPAddr, ifname string) (*net.UDPConn, error) {
	return net.ListenUDP("udp", laddr)
}
