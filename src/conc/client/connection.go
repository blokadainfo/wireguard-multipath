package client

import (
	"net"
	"sync"
	"time"
)

type Connection struct {
	srcAddr  *net.UDPAddr
	lastSeen time.Time
	l        sync.RWMutex
}

func NewConnection(srcAddr *net.UDPAddr) *Connection {
	return &Connection{
		srcAddr:  srcAddr,
		lastSeen: time.Time{},
		l:        sync.RWMutex{},
	}
}

func (c *Connection) GetSrcAddr() *net.UDPAddr {
	c.l.RLock()
	defer c.l.RUnlock()

	return c.srcAddr
}

func (c *Connection) SameSrcAddr(srcAddr *net.UDPAddr) bool {
	c.l.RLock()
	defer c.l.RUnlock()

	return c.srcAddr.String() == srcAddr.String()
}

func (c *Connection) IsStale() bool {
	c.l.RLock()
	defer c.l.RUnlock()

	return time.Since(c.lastSeen) > time.Minute
}

func (c *Connection) UpdateLastSeen() {
	c.l.Lock()
	defer c.l.Unlock()

	c.lastSeen = time.Now()
}
