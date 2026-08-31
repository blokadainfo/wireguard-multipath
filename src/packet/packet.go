package packet

import (
	"net"
	"sync/atomic"
)

type Packet struct {
	buffer []byte // buffer of original size
	n      int    // number of bytes that are actually written to the buffer

	freed *atomic.Bool
}

func NewPacket(buffer []byte, n int) Packet {
	if n == 0 {
		panic("packet contains no data")
	}

	return Packet{buffer: buffer, n: n, freed: &atomic.Bool{}}
}

func (p Packet) free() {
	if p.freed.Swap(true) {
		panic("Packet.free(): trying to free an already freed packet")
	}
}

func (p Packet) Bytes() []byte {
	if p.freed.Load() {
		panic("Packet.Bytes(): trying to access bytes of a freed packet")
	}

	return p.buffer[:p.n]
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
