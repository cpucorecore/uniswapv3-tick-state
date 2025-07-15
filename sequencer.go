package main

import (
	"sync"
)

type Sequenceable[T any] interface {
	Sequence() uint64
}

type Sequencer[T Sequenceable[T]] interface {
	Init(uint64)
	Commit(T, chan T)
}

type sequence[T Sequenceable[T]] struct {
	mu       *sync.Mutex
	cond     *sync.Cond
	sequence uint64
}

func NewSequencer[T Sequenceable[T]](fromSequence uint64) Sequencer[T] {
	mu := &sync.Mutex{}
	cond := sync.NewCond(mu)
	return &sequence[T]{
		mu:       mu,
		cond:     cond,
		sequence: fromSequence,
	}
}

func (s *sequence[T]) Init(sequence uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sequence == 0 {
		s.sequence = sequence
	} else {
		panic("sequencer already initialized")
	}
}

func (s *sequence[T]) Commit(item T, resultChan chan T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for s.sequence+1 != item.Sequence() {
		s.cond.Wait()
	}

	resultChan <- item
	s.sequence = item.Sequence()
	s.cond.Broadcast()
}
