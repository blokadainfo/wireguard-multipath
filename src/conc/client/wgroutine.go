package client

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/packet"
	"github.com/google/uuid"
)

var ErrWgRoutineClosed = errors.New("wireguard routine is closed")

type wgRoutine struct {
	clientId uuid.UUID
	wgSock   *net.UDPConn
	wgAddr   *net.UDPAddr
	closed   atomic.Bool
}

func NewWgRoutine(clientId uuid.UUID, wgServerAddr string) (*wgRoutine, error) {
	wgAddr, err := net.ResolveUDPAddr("udp4", wgServerAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination (wireguard server) address")
	}

	wgSrc, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination (wireguard server) address")
	}

	wgSock, err := net.ListenUDP("udp", wgSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to create a socket to the Wireguard server")
	}

	return &wgRoutine{
		clientId: clientId,
		wgSock:   wgSock,
		wgAddr:   wgAddr,
		closed:   atomic.Bool{},
	}, nil
}

func (r *wgRoutine) Close() error {
	if r.closed.Load() {
		return nil
	}

	r.closed.Store(true)

	return r.wgSock.Close()
}

func (r *wgRoutine) Read(bp *packet.BufferPool) (packet.Packet, error) {
	buffer := bp.Get()
	n, _, err := r.wgSock.ReadFromUDP(buffer) // WARN: This is blocking
	if err != nil {
		bp.Put(buffer)

		if r.closed.Load() {
			return packet.Packet{}, ErrWgRoutineClosed
		}

		return packet.Packet{}, fmt.Errorf("failed to read from wg socket: %v", err)
	}

	pkt := packet.NewPacket(buffer, n)

	return pkt, nil
}

func (r *wgRoutine) Write(pkt packet.PacketWithClientIDAndSrcAddr, deadline time.Duration) error {
	if r.closed.Load() {
		return ErrWgRoutineClosed
	}

	if err := r.wgSock.SetWriteDeadline(time.Now().Add(deadline)); err != nil {
		return fmt.Errorf("failed to set write deadline for wg socket: %v", err)
	}

	if _, err := r.wgSock.WriteToUDP(pkt.StripClientID(), r.wgAddr); err != nil {
		return fmt.Errorf("failed to write to wg socket: %v", err)
	}

	return nil
}
