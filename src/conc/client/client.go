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
		conns: make([]*Connection, 0, 5),
		l:     sync.RWMutex{},
	}, nil
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

	return pktWSA, err
}

func (c *Client) WriteToWgRoutine(pkt packet.PacketWithClientIDAndSrcAddr, deadline time.Duration) error {
	c.l.Lock()
	defer c.l.Unlock()

	if err := c.r.Write(pkt, deadline); err != nil {
		return err
	}

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
