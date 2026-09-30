package internal

import "sync/atomic"

// Atomic boolean defines atomic.
const (
	AtomicFalse = int32(iota)
	AtomicTrue
)

// AtomicBool is an atomic boolean.
type AtomicBool struct {
	int32
}

// Get gets atomic value.
func (a *AtomicBool) Get() bool {
	return atomic.LoadInt32(&a.int32) == AtomicTrue
}

// Swap swaps atomic value.
func (a *AtomicBool) Swap(val bool) bool {
	flag := AtomicFalse
	if val {
		flag = AtomicTrue
	}
	return atomic.SwapInt32(&a.int32, flag) == AtomicTrue
}

// Set sets atomic value.
func (a *AtomicBool) Set(val bool) {
	flag := AtomicFalse
	if val {
		flag = AtomicTrue
	}
	atomic.StoreInt32(&a.int32, flag)
}
