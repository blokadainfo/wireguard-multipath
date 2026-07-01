//go:build linux

package ifaceutils

import (
	"context"
	"fmt"
	"net"
	"syscall"
)

func BoundUDPConn(laddr *net.UDPAddr, ifname string) (*net.UDPConn, error) {
	if laddr == nil {
		laddr = &net.UDPAddr{
			IP: net.IPv4zero,
		}
	}

	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var ctrlErr error

			if err := c.Control(func(fd uintptr) {
				if err := syscall.SetsockoptInt(
					int(fd),
					syscall.SOL_SOCKET,
					syscall.SO_REUSEADDR,
					1,
				); err != nil {
					ctrlErr = fmt.Errorf("set SO_REUSEADDR: %w", err)
					return
				}

				if ifname != "" {
					if err := syscall.SetsockoptString(
						int(fd),
						syscall.SOL_SOCKET,
						syscall.SO_BINDTODEVICE,
						ifname,
					); err != nil {
						ctrlErr = fmt.Errorf("bind to device %q: %w", ifname, err)
					}
				}
			}); err != nil {
				return err
			}

			return ctrlErr
		},
	}

	pc, err := lc.ListenPacket(context.Background(), "udp", laddr.String())
	if err != nil {
		return nil, fmt.Errorf("listen on %v: %w", laddr, err)
	}

	uc, ok := pc.(*net.UDPConn)
	if !ok {
		pc.Close()
		return nil, fmt.Errorf("expected *net.UDPConn, got %T", pc)
	}

	return uc, nil
}
