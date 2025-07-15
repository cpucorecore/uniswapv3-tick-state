package main

import "sync"

type MutexValue[T any] struct {
	mu sync.Mutex
	v  T
}

func (m *MutexValue[T]) Get() T {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.v
}

func (m *MutexValue[T]) Set(val T) {
	m.mu.Lock()
	m.v = val
	m.mu.Unlock()
}
