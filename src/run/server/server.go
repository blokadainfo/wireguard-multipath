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

func Run(ctx context.Context, cfg config.ServerConfig) {
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

	// Make a buffer pool for reading and writing packets
	bp := packet.NewBufferPool()

	// Read from listener socket and send to the lReadCh
	lReadCh := make(chan packet.PacketWithClientIDAndSrcAddr, 1000)
	go readFromListener(ctx, bp, lSock, lReadCh)

	// Receive from lWriteCh and write to the listener socket
	lWriteCh := make(chan packet.PacketWithSrcAddrs, 1000)
	go writeToListener(ctx, bp, lSock, lWriteCh)

	// Make a thread-safe shared client map variable
	cm := client.NewClientMap()

	// Monitor clients and delete inactive ones
	go monitorClients(ctx, cm)

	// Receive from lReadCh and write to wireguard routine socket (based on client ID present in the packet)
	//
	// If the client (based on client ID present in the packet) doesn't exist,
	// creates it and starts a goroutine for reading from it's socket to the wireguard server and sending to lWriteCh
	go writeToClientWgRoutines(ctx, cfg, cm, bp, lReadCh, lWriteCh)

	// Block until an interrupt is received
	interrupt.Wait(ctx)
}

func readFromListener(ctx context.Context, bp *packet.BufferPool, lSock *net.UDPConn, lReadCh chan packet.PacketWithClientIDAndSrcAddr) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromListener()", "seq", "1")
			return
		default:
		}

		buffer := bp.Get()
		n, sa, err := lSock.ReadFromUDP(buffer)
		if err != nil {
			slog.Error("Failed to read data from the listener socket", "error", err)
			bp.Put(buffer)
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

func writeToListener(ctx context.Context, bp *packet.BufferPool, lSock *net.UDPConn, lWriteCh chan packet.PacketWithSrcAddrs) {
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

				bp.PutPSA(pkt)
				slog.Debug("Returned buffer to the pool", "packet", pkt.String())
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

func writeToClientWgRoutines(ctx context.Context, cfg config.ServerConfig, cm *client.ClientMap, bp *packet.BufferPool, lReadCh chan packet.PacketWithClientIDAndSrcAddr, lWriteCh chan packet.PacketWithSrcAddrs) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "writeToClientWgRoutines()", "seq", "1")
			return
		case pkt := <-lReadCh:
			if c, ok := cm.GetClient(pkt.ClientID()); ok {
				if err := c.WriteToQueue(pkt); err != nil {
					slog.Error("Failed to write the packet to the queue", "client_id", pkt.ClientID().String(), "error", err, "seq", "1")
					bp.PutPCIDSA(pkt)
					continue
				}
			} else {
				slog.Info("Creating new client", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String())
				c, err := client.NewClient(pkt.ClientID(), cfg.WireguardAddr)
				if err != nil {
					slog.Error("Failed to create client", "client_id", pkt.ClientID().String(), "error", err)
					bp.PutPCIDSA(pkt)
					continue
				}

				slog.Debug("Adding client to client map", "client_id", pkt.ClientID().String(), "address", pkt.SrcAddr().String(), "packet", pkt.String())
				if err := cm.AddClient(pkt.ClientID(), c); err != nil {
					slog.Error("Failed to add client to the client map", "client_id", pkt.ClientID().String(), "error", err)
					bp.PutPCIDSA(pkt)
					continue
				}

				go readFromClientWgRoutine(ctx, cm, bp, pkt.ClientID(), c, lWriteCh)
				go writeToClientWgRoutine(ctx, cfg, cm, bp, pkt.ClientID(), c)

				if err := c.WriteToQueue(pkt); err != nil {
					slog.Error("Failed to write the packet to the queue", "client_id", pkt.ClientID().String(), "error", err, "seq", "2")
					bp.PutPCIDSA(pkt)
					continue
				}
			}
		}
	}
}

func readFromClientWgRoutine(ctx context.Context, cm *client.ClientMap, bp *packet.BufferPool, clientId uuid.UUID, c *client.Client, lWriteCh chan packet.PacketWithSrcAddrs) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "readFromClientWgRoutine()", "seq", "1")
			return
		default:
		}

		pkt, err := c.ReadFromWgRoutine(bp)
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

func writeToClientWgRoutine(ctx context.Context, cfg config.ServerConfig, cm *client.ClientMap, bp *packet.BufferPool, clientId uuid.UUID, c *client.Client) {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("Context is done", "func", "writeToClientWgRoutine()", "seq", "1")
			return
		case pkt := <-c.ReadFromQueue():
			if err := c.WriteToWgRoutine(pkt, cfg.SocketWriteTimeout); err != nil {
				if errors.Is(err, client.ErrWgRoutineClosed) {
					slog.Debug("Failed to write to the wg routine for already existing client", "client_id", clientId.String(), "address", pkt.SrcAddr().String(), "packet", pkt.String(), "error", err)
				} else {
					slog.Error("Failed to write to the wg routine for already existing client", "client_id", clientId.String(), "address", pkt.SrcAddr().String(), "packet", pkt.String(), "error", err)
				}

				if _, err := cm.DelClient(pkt.ClientID()); err != nil {
					slog.Error("Failed to delete client from the client map", "client_id", clientId.String(), "error", err)
				}

				bp.PutPCIDSA(pkt)

				// Closes the goroutine for writing to the wg routine
				return
			}

			slog.Debug("Written to the wg routine for already existing client", "client_id", clientId.String(), "address", pkt.SrcAddr().String(), "packet", pkt.String())

			bp.PutPCIDSA(pkt)
			slog.Debug("Returned buffer to the pool", "packet", pkt.String())
		}
	}
}
