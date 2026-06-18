package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/packet"
	"github.com/google/uuid"
)

type wgRoutine struct {
	clientId uuid.UUID
	wgSock   *net.UDPConn
	wgAddr   *net.UDPAddr
	lastSeen time.Time
	l        sync.RWMutex
}

func NewWgRoutine(clientId uuid.UUID, wgServerAddr string) (*wgRoutine, error) {
	wgAddr, err := net.ResolveUDPAddr("udp4", wgServerAddr)
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve destination (wireguard server) address")
	}

	wgSrc, err := net.ResolveUDPAddr("udp4", "0.0.0.0:0")
	if err != nil {
		return nil, fmt.Errorf("Failed to resolve destination (wireguard server) address")
	}

	wgSock, err := net.ListenUDP("udp", wgSrc)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a socket to the Wireguard server")
	}

	return &wgRoutine{
		clientId: clientId,
		wgSock:   wgSock,
		wgAddr:   wgAddr,
		lastSeen: time.Time{},
		l:        sync.RWMutex{},
	}, nil
}

func (r *wgRoutine) Close() error {
	r.l.Lock()
	defer r.l.Unlock()

	return r.wgSock.Close()
}

func (r *wgRoutine) Read() (packet.Packet, error) {
	buffer := make([]byte, packet.RawDataBufferSize)
	n, _, err := r.wgSock.ReadFromUDP(buffer) // WARN: This is blocking
	if err != nil {
		return packet.Packet{}, fmt.Errorf("failed to read from wg socket with client ID %v: %v", r.clientId.String(), err)
	}

	// WARN: Has to be after the blocking action
	r.l.Lock()
	defer r.l.Unlock()

	pkt := packet.NewPacket(buffer, n)

	r.lastSeen = time.Now()

	return pkt, nil
}

func (r *wgRoutine) Write(pkt packet.Packet, deadline time.Duration) error {
	r.l.Lock()
	defer r.l.Unlock()

	if err := r.wgSock.SetWriteDeadline(time.Now().Add(deadline)); err != nil {
		return fmt.Errorf("failed to set write deadline for socket with client ID %v: %v", r.clientId.String(), err)
	}

	if _, err := r.wgSock.WriteToUDP(pkt.Bytes(), r.wgAddr); err != nil {
		return fmt.Errorf("failed to write to socket with client ID %v: %v", r.clientId.String(), err)
	}

	return nil
}
