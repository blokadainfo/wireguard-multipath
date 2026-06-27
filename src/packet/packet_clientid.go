package packet

import (
	"fmt"
	"net"
	"slices"
	"sync/atomic"

	"github.com/google/uuid"
)

const (
	uuidv4Size = 16
)

type PacketWithClientID struct {
	Packet

	cid    uuid.UUID
	offset uint8
}

func NewPacketWithClientID(buffer []byte, n int, clientId ...uuid.UUID) PacketWithClientID {
	if len(clientId) > 1 {
		panic("only one client id can be passed as parameter")
	}

	if n == 0 {
		panic("packet contains no data")
	}

	if len(clientId) == 1 {
		return PacketWithClientID{Packet: Packet{buffer: buffer, n: n, freed: &atomic.Bool{}}, cid: clientId[0], offset: 0}
	}

	if n <= uuidv4Size {
		panic("packet contains no data (<= uuidv4Size)")
	}

	cid := uuid.Must(uuid.FromBytes(buffer[:uuidv4Size]))
	if err := uuid.Validate(cid.String()); err != nil {
		panic(fmt.Errorf("failed to validate client id (%v): %v", cid.String(), err))
	}

	return PacketWithClientID{Packet: Packet{buffer: buffer, n: n, freed: &atomic.Bool{}}, cid: cid, offset: uuidv4Size}
}

// Redefined Bytes() method which returns bytes with the client id prefixed
func (p PacketWithClientID) Bytes() []byte {
	if p.freed.Load() {
		panic("PacketWithClientID.Bytes(): trying to access bytes of a freed packet")
	}

	return slices.Concat(p.cid[:], p.buffer[p.offset:p.n])
}

// Similar to Bytes() method but returns the bytes without client id
func (p PacketWithClientID) StripClientID() []byte {
	if p.freed.Load() {
		panic("PacketWithClientID.StripClientID(): trying to access bytes of a freed packet")
	}

	return p.buffer[p.offset:p.n]
}

func (p PacketWithClientID) ClientID() uuid.UUID {
	return p.cid
}

type PacketWithClientIDAndSrcAddr struct {
	PacketWithClientID

	sa *net.UDPAddr
}

func NewPacketWithClientIDAndSrcAddr(sa *net.UDPAddr, buffer []byte, n int, clientId ...uuid.UUID) PacketWithClientIDAndSrcAddr {
	if sa == nil {
		panic("source address can't be nil")
	}

	pkt := NewPacketWithClientID(buffer, n, clientId...)

	return PacketWithClientIDAndSrcAddr{PacketWithClientID: pkt, sa: sa}
}

func (p PacketWithClientIDAndSrcAddr) SrcAddr() *net.UDPAddr {
	return p.sa
}
