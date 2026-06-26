package packet

import (
	"sync"
)

const (
	bufferSize = 1500
)

type BufferPool struct {
	pool sync.Pool
}

func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() any {
				return make([]byte, bufferSize)
			},
		},
	}
}

func (bp *BufferPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

func (bp *BufferPool) Put(buffer []byte) {
	bp.pool.Put(buffer)
}

func (bp *BufferPool) PutP(pkt Packet) {
	pkt.free()
	bp.pool.Put(pkt.buffer)
}

func (bp *BufferPool) PutPSA(pkt PacketWithSrcAddrs) {
	pkt.free()
	bp.pool.Put(pkt.buffer)
}

func (bp *BufferPool) PutPCID(pkt PacketWithClientID) {
	pkt.free()
	bp.pool.Put(pkt.buffer)
}

func (bp *BufferPool) PutPCIDSA(pkt PacketWithClientIDAndSrcAddr) {
	pkt.free()
	bp.pool.Put(pkt.buffer)
}
