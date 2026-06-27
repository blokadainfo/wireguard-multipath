package client

import (
	"fmt"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/blokadainfo/wireguard-multipath/src/packet"
	"github.com/google/uuid"
)

type Client struct {
	r     *wgRoutine
	q     chan packet.PacketWithClientIDAndSrcAddr
	conns []*Connection
	l     sync.RWMutex
}

func NewClient(clientId uuid.UUID, wgServerAddr string) (*Client, error) {
	r, err := NewWgRoutine(clientId, wgServerAddr)
	if err != nil {
		return nil, fmt.Errorf("Failed to create routine: %v", err)
	}

	return &Client{
		r:     r,
		q:     make(chan packet.PacketWithClientIDAndSrcAddr, 1000),
		conns: make([]*Connection, 0, 5),
		l:     sync.RWMutex{},
	}, nil
}

func (c *Client) ReadFromQueue() <-chan packet.PacketWithClientIDAndSrcAddr {
	return c.q
}

// WARN: Drops packet if the queue is full
func (c *Client) WriteToQueue(pkt packet.PacketWithClientIDAndSrcAddr) error {
	select {
	case c.q <- pkt:
		return nil
	default:
		return fmt.Errorf("queue is full, dropping packet") // TODO: Make sure this is desired behaviour
	}
}

func (c *Client) CloseWgRoutine() error {
	c.l.Lock()
	defer c.l.Unlock()

	c.conns = slices.Delete(c.conns, 0, len(c.conns))
	return c.r.Close()
}

func (c *Client) ReadFromWgRoutine(bp *packet.BufferPool) (packet.PacketWithSrcAddrs, error) {
	pkt, err := c.r.Read(bp) // WARN: This is blocking
	if err != nil {
		return packet.PacketWithSrcAddrs{}, err
	}

	c.l.RLock()
	defer c.l.RUnlock()

	srcAddrs := make([]*net.UDPAddr, 0, len(c.conns))
	for _, conn := range c.conns {
		if conn.IsStale() {
			continue
		}

		srcAddrs = append(srcAddrs, conn.GetSrcAddr())
	}

	if len(srcAddrs) == 0 {
		return packet.PacketWithSrcAddrs{}, fmt.Errorf("no active connections to send to")
	}

	pktWSA := packet.NewPacketWithSrcAddrs(pkt, srcAddrs)

	return pktWSA, nil
}

func (c *Client) WriteToWgRoutine(pkt packet.PacketWithClientIDAndSrcAddr, deadline time.Duration) error {
	if err := c.r.Write(pkt, deadline); err != nil {
		return err
	}

	c.l.Lock()
	defer c.l.Unlock()

	for _, conn := range c.conns {
		if conn.SameSrcAddr(pkt.SrcAddr()) {
			conn.UpdateLastSeen()
			return nil
		}
	}

	conn := NewConnection(pkt.SrcAddr())
	c.conns = append(c.conns, conn)

	return nil
}

func (c *Client) IsActive() bool {
	c.l.Lock()
	defer c.l.Unlock()

	c.conns = slices.DeleteFunc(c.conns, func(conn *Connection) bool {
		return conn.IsStale()
	})

	if len(c.conns) == 0 {
		return false
	}

	return true
}
