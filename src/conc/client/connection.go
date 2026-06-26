package client

import (
	"net"
	"time"
)

type Connection struct {
	srcAddr  *net.UDPAddr
	lastSeen time.Time
}

func NewConnection(srcAddr *net.UDPAddr) *Connection {
	return &Connection{
		srcAddr:  srcAddr,
		lastSeen: time.Now(),
	}
}

func (c Connection) GetSrcAddr() *net.UDPAddr {
	return c.srcAddr
}

func (c Connection) SameSrcAddr(srcAddr *net.UDPAddr) bool {
	return c.srcAddr.String() == srcAddr.String()
}

func (c Connection) IsStale() bool {
	return time.Since(c.lastSeen) > time.Minute
}

func (c *Connection) UpdateLastSeen() {
	c.lastSeen = time.Now()
}
