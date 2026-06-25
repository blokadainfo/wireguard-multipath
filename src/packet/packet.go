package packet

import (
	"fmt"
	"hash/crc32"
	"net"
)

type Packet struct {
	buffer []byte // buffer of original size
	n      int    // number of bytes that are actually written to the buffer
	hash   string // hash of the actually written bytes
}

func NewPacket(buffer []byte, n int) Packet {
	if n == 0 {
		panic("packet contains no data")
	}

	h := crc32.NewIEEE()
	if _, err := h.Write(buffer[:n]); err != nil {
		panic("failed to write packet bytes to hash")
	}
	hS := fmt.Sprintf("%x", h.Sum(nil))

	return Packet{buffer: buffer, n: n, hash: hS}
}

func (p Packet) Bytes() []byte {
	return p.buffer[:p.n]
}

func (p Packet) String() string {
	return p.hash
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
