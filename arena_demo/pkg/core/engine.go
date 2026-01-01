package core

import (
	"arena_demo/pkg/arena"
	"arena_demo/pkg/fastqueue"
	"arena_demo/pkg/sysclock"
	"arena_demo/pkg/zlog"
	"fmt"
	"runtime"
)

// TaskType 定义任务类型 (Tagged Union 的 Tag)
const (
	TaskTypeCalc  = 0
	TaskTypeOrder = 1
)

// Task 是传递的数据结构 (Tagged Union 模式)
// 我们不使用 interface{}，而是将所有字段平铺在结构体中
// 这样可以避免类型断言和动态内存分配
type Task struct {
	Type int // 任务类型

	// Calc 任务字段
	Value int

	// Order 任务字段
	Price    float64
	Quantity int

	// 结果回传 (这里为了通用暂时用 any，极致优化可以使用 typed channel 或 callback)
	Resp chan any

	// LogBuf 是调用者提供的日志缓冲区 (实现 Zero Allocation Logging)
	LogBuf []byte
}

type OrderResult struct {
	Total       float64
	ProcessedAt int64
	Log         []byte
}

type Engine struct {
	Queue *fastqueue.RingBuffer[Task]
	Mem   *arena.Arena

	// 演示 Solution 1: 替代 Map
	// 使用定长数组存储用户状态 (Key: UserID 0-1023)
	// 访问速度: O(1)
	// GC 开销: 0 (这是大对象的一部分)
	UserVolume [1024]float64
}

func NewEngine() *Engine {
	return &Engine{
		Queue: fastqueue.New[Task](1024),
		Mem:   arena.Acquire(), // C World 独占的大内存块
	}
}

// Start 启动 "C 模式" 线程
func (e *Engine) Start() {
	go func() {
		// 1. 锁死线程，拒绝调度
		runtime.LockOSThread()

		fmt.Println("[Core] Started in C-Mode (Pinned Thread, Arena Memory)")

		for {
			// 2. 自旋轮询 (Busy Loop)，完全不让出 CPU
			// 就像 C 的 while(1)
			task, ok := e.Queue.Pop()
			if !ok {
				// 空转，为了避免 CPU 100% 稍微 yield 一下，
				// 在极低延迟场景下，这里可以使用 runtime.Gosched() 或者更底层的 cpu pause 指令
				// 但为了演示效果，我们不做任何 sleep
				runtime.Gosched()
				continue
			}

			// 3. 处理任务 (Zero GC)
			e.process(task)

			// 4. 重置 Arena (每处理一个任务重置一次，或者批量重置)
			// 这样保证内存永远在一个固定的小范围内复用，极大提高 Cache 命中率
			e.Mem.Reset()
		}
	}()
}

//go:nosplit
func (e *Engine) process(t Task) {
	// 演示：根据 Type 处理不同逻辑 (Tagged Union)
	switch t.Type {
	case TaskTypeCalc:
		// 演示：在 Arena 上分配内存 (完全绕过 Go GC)
		tempPtr := arena.New[int](e.Mem)
		*tempPtr = t.Value * 2
		e.UserVolume[0] += float64(*tempPtr) // 简单更新状态
		t.Resp <- *tempPtr
	case TaskTypeOrder:
		// 演示：处理订单逻辑
		// 1. 获取时间 (Zero Syscall)
		ts := sysclock.Now()

		// 2. 业务逻辑
		total := t.Price * float64(t.Quantity)

		// 演示：更新状态 (替代 Map)
		// 优化：使用位运算替代求模 (& 1023)
		// 1. 速度快 (CPU 指令周期少)
		// 2. 必定为正数，帮助编译器消除边界检查 (BCE)
		userID := t.Value & 1023
		e.UserVolume[userID] += total

		// 3. 记录日志 (Zero Allocation)
		var logBytes []byte
		if t.LogBuf != nil {
			// 使用调用者提供的 buffer
			logger := zlog.Wrap(t.LogBuf)
			logger.Int("ts", int(ts)).Str("type", "order").Int("uid", userID).Msg("processed")
			logBytes = logger.Bytes()
		}

		// 4. 返回结果
		t.Resp <- OrderResult{
			Total:       total,
			ProcessedAt: ts,
			Log:         logBytes,
		}
	}
}
