//go:build !linux

package scanner

import "sync"

// EvdevListener stub para sistemas no Linux.
type EvdevListener struct {
	mu       sync.Mutex
	callback func(code string, durationMs float64, source string)
}

func NewEvdevListener(cb func(code string, durationMs float64, source string)) *EvdevListener {
	return &EvdevListener{
		callback: cb,
	}
}

func (l *EvdevListener) Start() bool {
	return false
}

func (l *EvdevListener) Stop() {}

func (l *EvdevListener) IsRunning() bool {
	return false
}
