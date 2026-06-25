package client

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/packet"
	"github.com/google/uuid"
)

var ErrWgRoutineClosed = errors.New("wireguard routine is closed")

type wgRoutine struct {
	clientId uuid.UUID
	wgSock   *net.UDPConn
	wgAddr   *net.UDPAddr
	closed   bool
	lastSeen time.Time
	l        sync.RWMutex
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
		closed:   false,
		lastSeen: time.Time{},
		l:        sync.RWMutex{},
	}, nil
}

func (r *wgRoutine) Close() error {
	r.l.Lock()
	defer r.l.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	return r.wgSock.Close()
}

func (r *wgRoutine) Read() (packet.Packet, error) {
	buffer := make([]byte, packet.BufferSize)
	n, _, err := r.wgSock.ReadFromUDP(buffer) // WARN: This is blocking
	if err != nil {
		// WARN: Has to be after the blocking action
		r.l.RLock()
		defer r.l.RUnlock()

		if r.closed {
			return packet.Packet{}, ErrWgRoutineClosed
		}

		return packet.Packet{}, fmt.Errorf("failed to read from wg socket: %v", err)
	}

	// WARN: Has to be after the blocking action
	r.l.Lock()
	defer r.l.Unlock()

	pkt := packet.NewPacket(buffer, n)

	r.lastSeen = time.Now()

	return pkt, nil
}

func (r *wgRoutine) Write(pkt packet.PacketWithClientIDAndSrcAddr, deadline time.Duration) error {
	r.l.Lock()
	defer r.l.Unlock()

	if r.closed {
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
