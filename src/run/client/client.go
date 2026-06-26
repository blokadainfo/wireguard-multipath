package client

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/conc/routine"
	"github.com/blokadainfo/wireguard-multipath/src/config"
	"github.com/blokadainfo/wireguard-multipath/src/ifaceutils"
	"github.com/blokadainfo/wireguard-multipath/src/interrupt"
	"github.com/blokadainfo/wireguard-multipath/src/packet"

	"github.com/google/uuid"
)

func Run(cfg config.ClientConfig) {
	// Make a shared ctx used for stopping all goroutines
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// Make a listener socket for client-side wg proxy
	lAddr, err := net.ResolveUDPAddr("udp4", cfg.ListenAddr) // TODO: IPv6
	if err != nil {
		slog.Error("Error resolving UDP address for local proxy", "address", cfg.ListenAddr, "error", err)
		return
	}
	lSock, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		slog.Error("Error opening UDP socket for local proxy", "address", cfg.ListenAddr, "error", err)
		return
	}
	defer lSock.Close()
	slog.Info("Listening", "address", lAddr.String())

	// Make a thread-safe shared source address variable
	srcAddr := routine.NewSrcAddr()

	// Generate a client id for the packet header (UUIDv4)
	clientId := uuid.New()
	slog.Info("Generated new client id", "client_id", clientId.String())

	// Read from listener socket and send to the READ ch
	lReadCh := make(chan packet.PacketWithClientID, 1000)
	go readFromListener(ctx, lSock, lReadCh, srcAddr, clientId)

	// Write to the listener socket from the WRITE ch
	lWriteCh := make(chan packet.Packet, 1000)
	go writeToListener(ctx, lSock, lWriteCh, srcAddr)

	// Make a thread-safe shared routine map variable
	rm := routine.NewRoutineMap()

	// Monitor all interfaces and automatically create/destroy sockets on each one
	go monitorInterfaces(ctx, cfg, rm, lWriteCh)
	go writeToInterfaces(ctx, cfg, rm, lReadCh)

	// Block until an interrupt is received
	interrupt.Wait(ctx)
	ctxCancel() // Cancel immediately so all goroutines clean up nicely
}

func readFromListener(ctx context.Context, lSock *net.UDPConn, lReadCh chan packet.PacketWithClientID, srcAddr *routine.SrcAddr, clientId uuid.UUID) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromListener()", "seq", "1")
			return
		default:
		}

		buffer := make([]byte, packet.BufferSize)
		n, sa, err := lSock.ReadFromUDP(buffer)
		if err != nil {
			slog.Error("Failed to read data from the listener socket", "error", err)
			continue
		}
		srcAddr.SetAddress(sa)

		pkt := packet.NewPacketWithClientID(buffer, n, clientId)
		slog.Debug("Read data from the listener socket", "clientId", clientId.String(), "address", srcAddr.String(), "packet", pkt.String())

		select {
		case <-ctx.Done():
			return
		case lReadCh <- pkt:
		}
	}
}

func writeToListener(ctx context.Context, lSock *net.UDPConn, lWriteCh chan packet.Packet, srcAddr *routine.SrcAddr) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "writeToListener()", "seq", "1")
			return
		case pkt := <-lWriteCh:
			if _, err := lSock.WriteToUDP(pkt.Bytes(), srcAddr.GetAddress()); err != nil {
				slog.Error("Failed to write to the listener socket", "address", srcAddr.String(), "packet", pkt.String(), "error", err)
			} else {
				slog.Debug("Written data to the listener socket", "address", srcAddr.String(), "packet", pkt.String())
			}
		}
	}
}

func monitorInterfaces(ctx context.Context, cfg config.ClientConfig, rm *routine.RoutineMap, lWriteCh chan packet.Packet) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "monitorInterfaces()", "seq", "1")
			return
		default:
		}

		interfaces, err := net.Interfaces()
		if err != nil {
			slog.Error("Failed to get all interfaces", "error", err)
			time.Sleep(time.Second)
			continue
		}

		// Delete old interface routines
		for ifname, rtn := range rm.GetRoutines() {
			iface, err := net.InterfaceByName(ifname)
			if err != nil {
				slog.Info("Interface no longer exists, deleting routine", "interface", ifname)
				if _, err := rm.DelRoutine(ifname); err != nil {
					slog.Error("Failed to delete routine", "interface", ifname, "error", err)
				}
				continue
			}

			ifaddr, err := ifaceutils.GetAddressByInterface(*iface)
			if errors.Is(err, ifaceutils.ErrAddressNotAllowed) {
				slog.Debug("Failed to get interface address, deleting routine", "interface", ifname, "error", err)
				if _, err := rm.DelRoutine(ifname); err != nil {
					slog.Error("Failed to delete routine", "interface", ifname, "error", err)
				}
				continue
			}
			if err != nil {
				slog.Error("Failed to get interface address, deleting routine", "interface", ifname, "error", err)
				if _, err := rm.DelRoutine(ifname); err != nil {
					slog.Error("Failed to delete routine", "interface", ifname, "error", err)
				}
				continue
			}

			if !rtn.SameSrcAddr(ifaddr) {
				slog.Info("Interface changed address, deleting routine", "interface", ifname)
				if _, err := rm.DelRoutine(ifname); err != nil {
					slog.Error("Failed to delete routine", "interface", ifname, "error", err)
				}
				continue
			}
		}

		// Create new interface routines
		for _, iface := range interfaces {
			ifname := iface.Name

			if ifaceutils.IsInterfaceExcluded(cfg.ExcludedInterfaces, ifname) {
				slog.Debug("Skipping excluded", "interface", ifname)
				continue
			}

			if _, ok := rm.GetRoutine(ifname); ok {
				slog.Debug("Skipping already existing", "interface", ifname)
				continue
			}

			ifaddr, err := ifaceutils.GetAddressByInterface(iface)
			if err != nil {
				slog.Debug("Failed to get address", "interface", ifname, "error", err)
				continue
			}

			if ifaceutils.IsCIDRExcluded(cfg.ExcludedCIDRs, ifaddr) {
				slog.Debug("Skipping excluded CIDR", "interface", ifname, "address", ifaddr)
				continue
			}

			slog.Info("Adding new interface", "interface", ifname, "address", ifaddr)
			go createInterfaceRoutine(ctx, cfg, rm, ifname, ifaddr, lWriteCh)
		}

		time.Sleep(time.Second)
	}
}

func createInterfaceRoutine(ctx context.Context, cfg config.ClientConfig, rm *routine.RoutineMap, ifname string, ifaddr string, lWriteCh chan packet.Packet) {
	rtn, err := routine.NewRoutine(ifname, ifaddr, cfg.ServerAddr)
	if err != nil {
		slog.Error("Failed to create routine", "interface", ifname, "address", ifaddr, "error", err)
		return
	}

	if err := rm.SetRoutine(ifname, rtn); err != nil {
		slog.Error("Failed to set routine", "interface", ifname, "error", err)
		return
	}

	go readFromInterface(ctx, rm, ifname, rtn, lWriteCh)
}

func readFromInterface(ctx context.Context, rm *routine.RoutineMap, ifname string, rtn *routine.Routine, lWriteCh chan packet.Packet) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromInterface()", "interface", ifname, "seq", 1)
			return
		default:
		}

		pkt, err := rtn.Read()
		if err != nil {
			if errors.Is(err, routine.ErrRoutineClosed) {
				slog.Debug("Failed to read, removing routine", "interface", ifname, "error", err)
			} else {
				slog.Error("Failed to read, removing routine", "interface", ifname, "error", err)
			}

			if _, err := rm.DelRoutine(ifname); err != nil {
				slog.Error("Failed to delete routine", "interface", ifname, "error", err)
			}

			return
		}

		slog.Debug("Read data from the routine socket", "interface", ifname, "packet", pkt.String())

		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromInterface()", "interface", ifname, "seq", 2)
			return
		case lWriteCh <- pkt:
		}
	}
}

func writeToInterfaces(ctx context.Context, cfg config.ClientConfig, rm *routine.RoutineMap, lReadCh chan packet.PacketWithClientID) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "writeToInterfaces()", "seq", "1")
			return
		case pkt := <-lReadCh:
			wg := sync.WaitGroup{}
			for ifname, rtn := range rm.GetRoutines() {
				wg.Go(func() {
					if err := rtn.Write(pkt, cfg.SocketWriteTimeout); err != nil {
						if errors.Is(err, routine.ErrRoutineClosed) {
							slog.Debug("Failed to write, removing routine", "interface", ifname, "error", err)
						} else {
							slog.Error("Failed to write, removing routine", "interface", ifname, "error", err)
						}

						if _, err := rm.DelRoutine(ifname); err != nil {
							slog.Error("Failed to delete routine", "interface", ifname, "error", err)
						}
					} else {
						slog.Debug("Written data to interface", "interface", ifname, "packet", pkt.String())
					}
				})
			}
			wg.Wait()
			slog.Debug("Written data to all interfaces", "packet", pkt.String())
		}
	}
}
