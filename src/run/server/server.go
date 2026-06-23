package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/conc/client"
	"github.com/blokadainfo/wireguard-multipath/src/config"
	"github.com/blokadainfo/wireguard-multipath/src/interrupt"
	"github.com/blokadainfo/wireguard-multipath/src/packet"
	"github.com/google/uuid"
)

func Run(cfg config.ServerConfig) {
	// Make a shared ctx used for stopping all goroutines
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// Make a listener socket for wg-proxy clients
	lAddr, err := net.ResolveUDPAddr("udp", cfg.ListenAddr)
	if err != nil {
		slog.Error("Error resolving UDP address for the proxy server (me)", "address", cfg.ListenAddr, "error", err)
		return
	}
	lSock, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		slog.Error("Error opening UDP socket for the proxy server (me)", "address", cfg.ListenAddr, "error", err)
		return
	}
	defer lSock.Close()
	slog.Info("Listening", "address", lAddr.String())

	// Read from listener socket and send to the lReadCh
	lReadCh := make(chan packet.PacketWithClientIDAndSrcAddr) // TODO: Buffer of 1000?
	go readFromListener(ctx, lSock, lReadCh)

	// Receive from lWriteCh and write to the listener socket
	lWriteCh := make(chan packet.PacketWithSrcAddrs) // TODO: Buffer of 1000?
	go writeToListener(ctx, lSock, lWriteCh)

	// Make a thread-safe shared client map variable
	cm := client.NewClientMap()

	// Monitor clients and delete inactive ones
	go monitorClients(ctx, cm)

	// Receive from lReadCh and write to wireguard routine socket (based on client ID present in the packet)
	//
	// If the client (based on client ID present in the packet) doesn't exist,
	// creates it and starts a goroutine for reading from it's socket to the wireguard server and sending to lWriteCh
	go writeToClientWgRoutines(ctx, cfg, cm, lReadCh, lWriteCh)

	// Block until an interrupt is received
	interrupt.Wait(ctx)
	ctxCancel() // Cancel immediately so all goroutines clean up nicely
}

func readFromListener(ctx context.Context, lSock *net.UDPConn, lReadCh chan packet.PacketWithClientIDAndSrcAddr) {
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
		pkt := packet.NewPacketWithClientIDAndSrcAddr(sa, buffer, n)
		slog.Debug("Read data from the listener socket", "client_id", pkt.ClientID().String(), "address", sa.String(), "packet", pkt.String())

		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromListener()", "seq", "2")
			return
		case lReadCh <- pkt:
		}
	}
}

func writeToListener(ctx context.Context, lSock *net.UDPConn, lWriteCh chan packet.PacketWithSrcAddrs) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "writeToListener()", "seq", "1")
			return
		case pkt := <-lWriteCh:
			for _, srcAddr := range pkt.GetSourceAddresses() {
				if _, err := lSock.WriteToUDP(pkt.Bytes(), srcAddr); err != nil {
					slog.Error("Failed to write to the listener socket", "adress", srcAddr.String(), "packet", pkt.String(), "error", err)
				} else {
					slog.Debug("Written data to the listener socket", "address", srcAddr.String(), "packet", pkt.String())
				}
			}
		}
	}
}

func monitorClients(ctx context.Context, cm *client.ClientMap) {
	for {
		delCtx, delCtxCancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(30 * time.Second)
			cm.DelInactiveClients()
			delCtxCancel()
		}()

		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "monitorClients()", "seq", "1")
			return
		case <-delCtx.Done():
			slog.Debug("Deleted inactive clients")
		}
	}
}

func writeToClientWgRoutines(ctx context.Context, cfg config.ServerConfig, cm *client.ClientMap, lReadCh chan packet.PacketWithClientIDAndSrcAddr, lWriteCh chan packet.PacketWithSrcAddrs) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "writeToClientWgRoutines()", "seq", "1")
			return
		case pkt := <-lReadCh:
			if c, ok := cm.GetClient(pkt.ClientID()); ok {
				go func() {
					if err := c.WriteToWgRoutine(pkt.SrcAddr(), pkt.StripClientID(), cfg.SocketWriteTimeout); err != nil {
						if errors.Is(err, client.ErrWgRoutineClosed) {
							slog.Debug("Failed to write to the wg routine for already existing client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String(), "error", err)
						} else {
							slog.Error("Failed to write to the wg routine for already existing client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String(), "error", err)
						}
					} else {
						slog.Debug("Written to the wg routine for already existing client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String())
					}
				}()
			} else {
				slog.Info("Creating new client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String())
				c, err := client.NewClient(pkt.ClientID(), cfg.WireguardAddr)
				if err != nil {
					slog.Error("Failed to create client", "client_id", pkt.ClientID().String(), "error", err)
					continue
				}

				slog.Debug("Adding client to client map", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String())
				if err := cm.AddClient(pkt.ClientID(), c); err != nil {
					slog.Error("Failed to add client to the client map", "client_id", pkt.ClientID().String(), "error", err)
					continue
				}

				go func() {
					if err := c.WriteToWgRoutine(pkt.SrcAddr(), pkt.StripClientID(), cfg.SocketWriteTimeout); err != nil {
						if errors.Is(err, client.ErrWgRoutineClosed) {
							slog.Debug("Failed to write to the wg routine for the new client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String(), "error", err)
						} else {
							slog.Error("Failed to write to the wg routine for the new client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String(), "error", err)
						}
					} else {
						slog.Debug("Written to the wg routine for the new client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String())
					}
				}()

				go readFromClientWgRoutine(ctx, cm, pkt.ClientID(), c, lWriteCh)
			}
		}
	}
}

func readFromClientWgRoutine(ctx context.Context, cm *client.ClientMap, clientId uuid.UUID, c *client.Client, lWriteCh chan packet.PacketWithSrcAddrs) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromClientWgRoutine()", "seq", "1")
			return
		default:
		}

		pkt, err := c.ReadFromWgRoutine()
		if err != nil {
			if errors.Is(err, client.ErrWgRoutineClosed) {
				slog.Debug("Failed to read, removing client", "client_id", clientId.String(), "error", err)
			} else {
				slog.Error("Failed to read, removing client", "client_id", clientId.String(), "error", err)
			}

			if _, err := cm.DelClient(clientId); err != nil {
				slog.Error("Failed to delete client from the client map", "client_id", clientId.String(), "error", err)
			}

			// Closes the goroutine for reading from the wg routine
			return
		}
		slog.Debug("Read data from the client wg routine", "client_id", clientId.String(), "packet", pkt.String())

		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromClientWgRoutine()", "seq", "2")
			return
		case lWriteCh <- pkt:
		}
	}
}
