package arena

import (
	"sync"
	"unsafe"
)

// Arena 是一个基于切片的内存分配器
type Arena struct {
	buf    []byte
	offset int
}

// 全局对象池，复用 Arena 对象本身及其底层的 buf
// 避免反复向 OS 申请大块内存
var arenaPool = sync.Pool{
	New: func() any {
		// 默认分配 64MB 的块，根据需要调整
		return &Arena{
			buf:    make([]byte, 64*1024*1024),
			offset: 0,
		}
	},
}

// Acquire 从全局池中借出一个 Arena
// 必须配合 Release 使用
func Acquire() *Arena {
	return arenaPool.Get().(*Arena)
}

// Release 重置 Arena 并归还给全局池
// 调用后，之前通过该 Arena 分配的所有指针都将失效（逻辑上）
// 严禁在 Release 后继续使用这些指针！
func (a *Arena) Release() {
	a.Reset()
	arenaPool.Put(a)
}

// Reset 仅重置偏移量，不归还给 Pool
// 适用于同一个 Arena 被同一个线程反复复用的场景
func (a *Arena) Reset() {
	a.offset = 0
}

// New 在 Arena 上分配一个 T 类型对象
// 返回 *T
func New[T any](a *Arena) *T {
	var zero T
	size := int(unsafe.Sizeof(zero))
	align := int(unsafe.Alignof(zero))

	// 处理对齐
	padding := (align - (a.offset % align)) % align
	if a.offset+padding+size > len(a.buf) {
		// 内存不足时的策略：
		// 1. 简单 panic (当前实现)
		// 2. 自动扩容 (分配更大的 buf 并链接起来，较复杂)
		panic("arena: out of memory")
	}

	a.offset += padding
	ptr := unsafe.Pointer(&a.buf[a.offset])
	a.offset += size

	// 必须清零内存，因为这是复用的 buf，可能包含脏数据
	// 对于小对象，编译器通常会优化这个 clear 操作
	*(*T)(ptr) = zero

	return (*T)(ptr)
}

// MakeSlice 在 Arena 上分配一个 T 类型的切片
// length: 切片长度, capacity: 切片容量
func MakeSlice[T any](a *Arena, length, capacity int) []T {
	var zero T
	elemSize := int(unsafe.Sizeof(zero))
	elemAlign := int(unsafe.Alignof(zero))

	size := elemSize * capacity

	// 处理对齐
	padding := (elemAlign - (a.offset % elemAlign)) % elemAlign
	if a.offset+padding+size > len(a.buf) {
		panic("arena: out of memory")
	}

	a.offset += padding
	basePtr := unsafe.Pointer(&a.buf[a.offset])
	a.offset += size

	// 构造切片头
	// sliceHeader := struct {
	// 	Data uintptr
	// 	Len  int
	// 	Cap  int
	// }{uintptr(basePtr), length, capacity}
	// return *(*[]T)(unsafe.Pointer(&sliceHeader))

	// 使用 unsafe.Slice 更安全 (Go 1.17+)
	s := unsafe.Slice((*T)(basePtr), capacity)

	// 清零切片内存 (如果需要)
	// 注意：对于大块内存，清零可能有开销，如果确认会立即覆盖可跳过
	// 这里为了安全默认清零
	var empty T
	for i := 0; i < capacity; i++ {
		s[i] = empty
	}

	return s[:length]
}
