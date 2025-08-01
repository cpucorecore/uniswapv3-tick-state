package main

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type SafeDB struct {
	db    DB
	locks map[common.Address]*sync.RWMutex
	mu    sync.RWMutex
}

func NewSafeDB(db DB) DB {
	return &SafeDB{
		db:    db,
		locks: make(map[common.Address]*sync.RWMutex),
	}
}

func (s *SafeDB) getOrCreateLock(addr common.Address) *sync.RWMutex {
	s.mu.RLock()
	lock, exists := s.locks[addr]
	s.mu.RUnlock()

	if exists {
		return lock
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if lock, exists = s.locks[addr]; exists {
		return lock
	}

	lock = &sync.RWMutex{}
	s.locks[addr] = lock
	return lock
}

func (s *SafeDB) Close() {
	s.db.Close()
}

func (s *SafeDB) SetFinishHeight(height uint64) error {
	return s.db.SetFinishHeight(height)
}

func (s *SafeDB) GetFinishHeight() (uint64, error) {
	return s.db.GetFinishHeight()
}

func (s *SafeDB) SetTickState(addr common.Address, tickState *TickState) error {
	lock := s.getOrCreateLock(addr)
	lock.Lock()
	defer lock.Unlock()
	return s.db.SetTickState(addr, tickState)
}

func (s *SafeDB) GetTickState(addr common.Address, tick int32) (*TickState, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.GetTickState(addr, tick)
}

func (s *SafeDB) GetTickStates(addr common.Address, fromTick, toTick int32) ([]*TickState, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.GetTickStates(addr, fromTick, toTick)
}

func (s *SafeDB) GetAllTickStates(addr common.Address) ([]*TickState, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.GetAllTickStates(addr)
}

func (s *SafeDB) SetCurrentTick(addr common.Address, tick int32) error {
	lock := s.getOrCreateLock(addr)
	lock.Lock()
	defer lock.Unlock()
	return s.db.SetCurrentTick(addr, tick)
}

func (s *SafeDB) GetCurrentTick(addr common.Address) (int32, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.GetCurrentTick(addr)
}

func (s *SafeDB) SetTickSpacing(addr common.Address, tickSpacing int32) error {
	lock := s.getOrCreateLock(addr)
	lock.Lock()
	defer lock.Unlock()
	return s.db.SetTickSpacing(addr, tickSpacing)
}

func (s *SafeDB) GetTickSpacing(addr common.Address) (int32, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.GetTickSpacing(addr)
}

func (s *SafeDB) PoolExists(addr common.Address) (bool, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.PoolExists(addr)
}

func (s *SafeDB) SetHeight(addr common.Address, height uint64) error {
	lock := s.getOrCreateLock(addr)
	lock.Lock()
	defer lock.Unlock()
	return s.db.SetHeight(addr, height)
}

func (s *SafeDB) GetHeight(addr common.Address) (uint64, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.GetHeight(addr)
}

func (s *SafeDB) GetPoolState(addr common.Address) (*PoolState, error) {
	lock := s.getOrCreateLock(addr)
	lock.RLock()
	defer lock.RUnlock()
	return s.db.GetPoolState(addr)
}

func (s *SafeDB) SetPoolState(addr common.Address, poolState *PoolState) error {
	lock := s.getOrCreateLock(addr)
	lock.Lock()
	defer lock.Unlock()
	return s.db.SetPoolState(addr, poolState)
}

func (s *SafeDB) DeletePoolState(addr common.Address) error {
	lock := s.getOrCreateLock(addr)
	lock.Lock()
	defer lock.Unlock()
	return s.db.DeletePoolState(addr)
}

func (s *SafeDB) CleanupLocks() {
	s.mu.Lock()
	defer s.mu.Unlock()
	// TODO
}
