package routine

import (
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/ifaceutils"
	"github.com/blokadainfo/wireguard-multipath/src/packet"
)

var ErrRoutineClosed = errors.New("routine is closed")

type Routine struct {
	ifname  string
	sock    *net.UDPConn
	srcAddr *net.UDPAddr
	dstAddr *net.UDPAddr
	closed  atomic.Bool
}

func NewRoutine(ifname string, ifaddr string, serverAddr string) (*Routine, error) {
	dstAddr, err := net.ResolveUDPAddr("udp4", serverAddr) // TODO: IPv6
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination (server) address: %w", err)
	}

	srcAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%v:0", ifaddr)) // TODO: IPv6
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source address, not using this interface: %w", err)
	}

	sock, err := ifaceutils.BoundUDPConn(srcAddr, ifname)
	if err != nil {
		return nil, fmt.Errorf("failed to create a socket bound to interface, not using this inteface: %w", err)
	}

	if err := sock.SetReadBuffer(packet.SocketReadBufferSize); err != nil {
		return nil, fmt.Errorf("failed to set read buffer, not using this interface: %w", err)
	}
	if err := sock.SetWriteBuffer(packet.SocketWriteBufferSize); err != nil {
		return nil, fmt.Errorf("failed to set write buffer, not using this interface: %w", err)
	}

	return &Routine{
		ifname:  ifname,
		sock:    sock,
		srcAddr: srcAddr,
		dstAddr: dstAddr,
		closed:  atomic.Bool{},
	}, nil
}

func (r *Routine) Close() error {
	if r.closed.Load() {
		return nil
	}

	r.closed.Store(true)

	return r.sock.Close()
}

// If error is returned this function handles putting the buffer back into the pool
func (r *Routine) Read(bp *packet.BufferPool) (packet.Packet, error) {
	buffer := bp.Get()
	n, _, err := r.sock.ReadFromUDP(buffer) // WARN: This is blocking
	if err != nil {
		bp.Put(buffer)

		if r.closed.Load() {
			return packet.Packet{}, ErrRoutineClosed
		}

		return packet.Packet{}, fmt.Errorf("failed to read from socket: %w", err)
	}

	pkt := packet.NewPacket(buffer, n)

	return pkt, nil
}

func (r *Routine) Write(pkt packet.PacketWithClientID, deadline time.Duration) error {
	if r.closed.Load() {
		return ErrRoutineClosed
	}

	if err := r.sock.SetWriteDeadline(time.Now().Add(deadline)); err != nil {
		return fmt.Errorf("failed to set write deadline for socket: %w", err)
	}

	if _, err := r.sock.WriteToUDP(pkt.Bytes(), r.dstAddr); err != nil {
		return fmt.Errorf("failed to write to socket: %w", err)
	}

	return nil
}

// Returns true if the srcAddr stayed the same, otherwise false
func (r *Routine) SameSrcAddr(ifaddr string) bool {
	return r.srcAddr.String() == fmt.Sprintf("%v:0", ifaddr)
}
