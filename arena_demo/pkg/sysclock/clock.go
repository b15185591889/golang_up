package sysclock

import (
	"sync/atomic"
	"time"
)

var (
	// nowNano stores the current time in nanoseconds (UnixNano)
	// Accessed via atomic, updated by a background ticker.
	nowNano atomic.Int64
)

func init() {
	// Initialize with current time
	nowNano.Store(time.Now().UnixNano())

	// Start a background goroutine to update time every 1ms
	// This allows Core layer to get "approximate" time with 0 syscall overhead.
	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for t := range ticker.C {
			nowNano.Store(t.UnixNano())
		}
	}()
}

// Now returns the cached current time in nanoseconds.
// Cost: ~0.5ns (Atomic Load), compared to ~50ns (syscall time.Now)
func Now() int64 {
	return nowNano.Load()
}
