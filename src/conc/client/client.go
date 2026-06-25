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
	return c.r.Close() // TODO: should this be a goroutine?
}

func (c *Client) ReadFromWgRoutine() (packet.PacketWithSrcAddrs, error) {
	c.l.Lock()

	c.conns = slices.DeleteFunc(c.conns, func(conn *Connection) bool {
		return conn.IsStale()
	})

	srcAddrs := make([]*net.UDPAddr, 0, len(c.conns))
	for _, conn := range c.conns {
		srcAddrs = append(srcAddrs, conn.GetSrcAddr())
	}

	c.l.Unlock()

	// WARN: This is blocking
	pkt, err := c.r.Read()
	pktWSA := packet.NewPacketWithSrcAddrs(pkt, srcAddrs)

	return pktWSA, err
}

func (c *Client) WriteToWgRoutine(pkt packet.PacketWithClientIDAndSrcAddr, deadline time.Duration) error {
	c.l.Lock()
	defer c.l.Unlock()

	found := false
	for _, conn := range c.conns {
		if conn.SameSrcAddr(pkt.SrcAddr()) {
			found = true
			conn.UpdateLastSeen()
			break
		}
	}
	if !found {
		conn := NewConnection(pkt.SrcAddr())
		conn.UpdateLastSeen()
		c.conns = append(c.conns, conn)
	}

	return c.r.Write(pkt, deadline)
}

func (c *Client) IsActive() bool {
	c.l.RLock()
	defer c.l.RUnlock()

	for _, conn := range c.conns {
		if !conn.IsStale() {
			return true
		}
	}

	return false
}
