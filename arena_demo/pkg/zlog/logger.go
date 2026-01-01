package zlog

import (
	"arena_demo/pkg/arena"
	"strconv"
)

// Logger 是一个极速、零分配的日志记录器
// 它直接将日志数据写入 Arena 内存，不进行任何 syscall
type Logger struct {
	buf []byte // 实际上指向 Arena 的内存
}

// New 在 Arena 上创建一个 Logger
func New(a *arena.Arena) *Logger {
	// 预分配 4KB 的日志缓冲区
	return &Logger{
		buf: arena.MakeSlice[byte](a, 0, 4096),
	}
}

// Wrap 使用外部提供的 buffer 创建 Logger (实现 Caller-Allocated Logging)
func Wrap(buf []byte) *Logger {
	return &Logger{
		buf: buf,
	}
}

// Int 写入一个整数 (无 GC, 无 strconv 开销)
func (l *Logger) Int(key string, val int) *Logger {
	l.appendString(key)
	l.appendString("=")
	l.appendInt(val)
	l.appendString(" ")
	return l
}

// Str 写入一个字符串
func (l *Logger) Str(key string, val string) *Logger {
	l.appendString(key)
	l.appendString("=")
	l.appendString(val)
	l.appendString(" ")
	return l
}

// Msg 结束一条日志并写入消息
func (l *Logger) Msg(msg string) {
	l.appendString("msg=")
	l.appendString(msg)
	l.appendString("\n")
}

// Bytes 返回当前缓冲区的所有内容 (用于最后一次性输出)
func (l *Logger) Bytes() []byte {
	return l.buf
}

// --- 内部极速实现 ---

func (l *Logger) appendString(s string) {
	// 直接 append，如果 Arena 足够大，这里只是简单的内存 copy
	// 注意：这里为了简化直接用了 append，实际上如果要极致优化，
	// 应该手动 copy 内存，避免 Go 编译器的边界检查
	l.buf = append(l.buf, s...)
}

func (l *Logger) appendInt(i int) {
	// 使用 strconv.AppendInt 是最高效的标准库方法，
	// 它不会产生内存分配，直接写入 buffer
	l.buf = strconv.AppendInt(l.buf, int64(i), 10)
}

// 为了绕过 Go 的一些安全检查，我们可以用 unsafe 来实现更快的 copy
// 但为了代码可读性，这里暂时保留 append
