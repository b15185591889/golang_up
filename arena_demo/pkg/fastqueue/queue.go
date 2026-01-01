package fastqueue

import (
	"sync/atomic"
)

// CacheLinePad 用于防止 False Sharing
// 现代 CPU Cache Line 通常是 64 字节
type CacheLinePad struct {
	_ [64]byte
}

// RingBuffer 是一个单生产者单消费者(SPSC)的无锁队列。
// 优化：增加了 Cache Padding 防止伪共享
type RingBuffer[T any] struct {
	buffer []T
	size   uint64
	mask   uint64

	_ CacheLinePad // 隔离只读区和读写区

	head uint64 // write index (Producer Only)

	_ CacheLinePad // 隔离 Head 和 Tail，防止两个核心争抢同一个 Cache Line

	tail uint64 // read index (Consumer Only)

	_ CacheLinePad
}

func New[T any](size uint64) *RingBuffer[T] {
	// size must be power of 2
	if size&(size-1) != 0 {
		panic("size must be power of 2")
	}
	return &RingBuffer[T]{
		buffer: make([]T, size),
		size:   size,
		mask:   size - 1,
	}
}

// Push 写入数据 (Go World -> C World)
func (rb *RingBuffer[T]) Push(item T) bool {
	head := atomic.LoadUint64(&rb.head)
	tail := atomic.LoadUint64(&rb.tail)

	if head-tail >= rb.size {
		return false // Full
	}

	// 简单的自旋或者直接写入
	// 注意：这里为了简化没做复杂的 Memory Barrier 处理，
	// 在生产级代码中需要 Padding 防止 False Sharing。
	rb.buffer[head&rb.mask] = item
	atomic.AddUint64(&rb.head, 1)
	return true
}

// Pop 读取数据 (C World 内部使用)
func (rb *RingBuffer[T]) Pop() (T, bool) {
	head := atomic.LoadUint64(&rb.head)
	tail := atomic.LoadUint64(&rb.tail)

	var empty T
	if tail >= head {
		return empty, false // Empty
	}

	item := rb.buffer[tail&rb.mask]
	atomic.AddUint64(&rb.tail, 1)
	return item, true
}
