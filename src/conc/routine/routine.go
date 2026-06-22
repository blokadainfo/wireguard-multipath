package routine

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/ifaceutils"
	"github.com/blokadainfo/wireguard-multipath/src/packet"
)

var ErrRoutineClosed = errors.New("routine is closed")

type Routine struct {
	ifname   string
	sock     *net.UDPConn
	srcAddr  *net.UDPAddr
	dstAddr  *net.UDPAddr
	closed   bool
	lastSeen time.Time
	l        sync.RWMutex
}

func NewRoutine(ifname string, ifaddr string, serverAddr string) (*Routine, error) {
	dstAddr, err := net.ResolveUDPAddr("udp4", serverAddr) // TODO: IPv6
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination (server) address")
	}

	srcAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%v:0", ifaddr)) // TODO: IPv6
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source address, not using this interface")
	}

	sock, err := ifaceutils.BoundUdpConn(srcAddr, ifname)
	if err != nil {
		return nil, fmt.Errorf("failed to create a socket bount to interface, not using this inteface")
	}

	return &Routine{
		ifname:   ifname,
		sock:     sock,
		srcAddr:  srcAddr,
		dstAddr:  dstAddr,
		closed:   false,
		lastSeen: time.Time{},
		l:        sync.RWMutex{},
	}, nil
}

func (r *Routine) Close() error {
	r.l.Lock()
	defer r.l.Unlock()

	if r.closed {
		return nil
	}

	r.closed = true

	return r.sock.Close()
}

func (r *Routine) Read() (packet.Packet, error) {
	buffer := make([]byte, packet.BufferSize)
	n, _, err := r.sock.ReadFromUDP(buffer) // WARN: This is blocking
	if err != nil {
		// WARN: Has to be after the blocking action
		r.l.RLock()
		defer r.l.RUnlock()

		if r.closed {
			return packet.Packet{}, ErrRoutineClosed
		}

		return packet.Packet{}, fmt.Errorf("failed to read from socket: %v", err)
	}

	// WARN: Has to be after the blocking action
	r.l.Lock()
	defer r.l.Unlock()

	pkt := packet.NewPacket(buffer, n)

	r.lastSeen = time.Now()

	return pkt, nil
}

func (r *Routine) Write(pkt packet.PacketWithClientID, deadline time.Duration) error {
	r.l.Lock()
	defer r.l.Unlock()

	if r.closed {
		return ErrRoutineClosed
	}

	if err := r.sock.SetWriteDeadline(time.Now().Add(deadline)); err != nil {
		return fmt.Errorf("failed to set write deadline for socket: %v", err)
	}

	if _, err := r.sock.WriteToUDP(pkt.Bytes(), r.dstAddr); err != nil {
		return fmt.Errorf("failed to write to socket: %v", err)
	}

	return nil
}

// Returns true if the srcAddr stayed the same, otherwise false
func (r *Routine) SameSrcAddr(ifaddr string) bool {
	r.l.RLock()
	defer r.l.RUnlock()

	return r.srcAddr.String() == fmt.Sprintf("%v:0", ifaddr)
}
