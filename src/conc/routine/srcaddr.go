package routine

import (
	"net"
	"sync"
)

type SrcAddr struct {
	addr *net.UDPAddr
	l    sync.RWMutex
	c    chan struct{}
}

func NewSrcAddr() *SrcAddr {
	return &SrcAddr{
		addr: nil,
		l:    sync.RWMutex{},
		c:    make(chan struct{}),
	}
}

func (sa *SrcAddr) SetAddress(addr *net.UDPAddr) {
	if addr == nil {
		panic("Parameter addr can't be nil")
	}

	sa.l.Lock()
	defer sa.l.Unlock()
	sa.addr = addr

	// Close the channel if it's not already closed
	select {
	case <-sa.c:
	default:
		close(sa.c)
	}
}

// Returns the address as soon as it gets set by SetAddress(), it will block otherwise
func (sa *SrcAddr) GetAddress() *net.UDPAddr {
	<-sa.c // Wait for the channel to close i.e. address to be written
	sa.l.RLock()
	defer sa.l.RUnlock()

	if sa.addr == nil {
		panic("Should be impossible to be nil")
	}

	return sa.addr
}

// Returns the address as string as soon as it gets set by SetAddress(), it will block otherwise
func (sa *SrcAddr) String() string {
	return sa.GetAddress().String()
}
