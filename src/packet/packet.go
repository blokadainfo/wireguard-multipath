package packet

import (
	"fmt"
	"hash/crc32"
	"net"
	"slices"

	"github.com/google/uuid"
)

const (
	wireguardMTU      = 1340 // TODO: Make this configurable
	wireguardOverhead = 80
	uuidv4Size        = 16

	RawDataBufferSize                   = wireguardMTU + wireguardOverhead // Buffer size for the packet with raw data (without client id header)
	RawDataWithClientIdHeaderBufferSize = RawDataBufferSize + uuidv4Size   // Buffer size for the packet with both the client id header and raw data
)

type Packet struct {
	b []byte
}

func NewPacket(buffer []byte, n int) Packet {
	b := make([]byte, n)
	copy(b, buffer[:n])

	return Packet{b: b}
}

func (p Packet) String() string {
	h := crc32.NewIEEE()
	if _, err := h.Write(p.b); err != nil {
		panic("failed to write packet bytes to hash")
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func (p Packet) Bytes() []byte {
	b := make([]byte, len(p.b))
	copy(b, p.b)

	return b
}

type PacketWithSrcAddrs struct {
	Packet

	srcAddrs []*net.UDPAddr
}

func NewPacketWithSrcAddrs(pkt Packet, srcAddrs []*net.UDPAddr) PacketWithSrcAddrs {
	return PacketWithSrcAddrs{Packet: pkt, srcAddrs: srcAddrs}
}

func (p PacketWithSrcAddrs) GetSourceAddresses() []*net.UDPAddr {
	return p.srcAddrs
}

type PacketWithClientID struct {
	Packet

	cid uuid.UUID
}

func NewPacketWithClientID(buffer []byte, n int, clientId ...uuid.UUID) PacketWithClientID {
	b := make([]byte, n)
	copy(b, buffer[:n])

	switch len(clientId) {
	case 0:
	case 1:
		b = slices.Insert(b, 0, clientId[0][:]...)
	default:
		panic("only one client id can be passed as parameter")
	}

	if len(b) < uuidv4Size {
		panic(fmt.Errorf("packet has less than the size of UUIDv4 in bytes: %v", string(b)))
	}
	if len(b) == uuidv4Size {
		panic(fmt.Errorf("packet contains only the UUIDv4 in bytes: %v", string(b)))
	}

	cid, err := uuid.FromBytes(b[:uuidv4Size])
	if err != nil {
		panic(fmt.Errorf("failed to extract client id from buffer: %v", err))
	}

	if err := uuid.Validate(cid.String()); err != nil {
		panic(fmt.Errorf("failed to validate client id (%v): %v", cid.String(), err))
	}

	return PacketWithClientID{Packet: Packet{b: b}, cid: cid}
}

func (p PacketWithClientID) ClientID() uuid.UUID {
	return p.cid
}

func (p PacketWithClientID) StripClientID() Packet {
	if len(p.b) < uuidv4Size {
		panic(fmt.Errorf("packet has less than the size of UUIDv4 in bytes: %v", p.String()))
	}

	if len(p.b) == uuidv4Size {
		panic(fmt.Errorf("packet contains only the UUIDv4 in bytes: %v", p.String()))
	}

	b := make([]byte, len(p.b)-uuidv4Size)
	copy(b, p.b[uuidv4Size:])

	return Packet{b: b}
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
