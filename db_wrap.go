package main

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type TickStateRepo interface {
	SetTickState(address common.Address, tick int32, tickState *TickState) error
	GetTickState(address common.Address, tick int32) (*TickState, error)
	GetTickStates(address common.Address, tickLower, tickUpper int32) ([]*TickState, error)
	GetAllTicks() (map[common.Address][]*TickState, error) // for dev
	SetCurrentTick(address common.Address, currentTick int32) error
	GetCurrentTick(address common.Address) (int32, error)
	SetTickSpacing(address common.Address, tickSpacing int32) error
	GetTickSpacing(address common.Address) (int32, error)
	TickExists(address common.Address) (bool, error)
}

type HeightRepo interface {
	SetHeight(height uint64) error
	GetHeight() (uint64, error)
}

type Repo interface {
	TickStateRepo
	HeightRepo
	Close()
}

type repo struct {
	db *RocksDB
}

func (r *repo) Close() {
	r.db.Close()
}

func NewRepo(db *RocksDB) Repo {
	return &repo{
		db: db,
	}
}

func (r *repo) SetTickState(addr common.Address, tick int32, tickState *TickState) error {
	key := GetTickStateKey(addr, tick).GetKey()
	value, err := tickState.MarshalBinary()
	if err != nil {
		return err
	}
	return r.db.Set(key, value)
}

var (
	EmptyTickState = &TickState{
		LiquidityNet: big.NewInt(0),
	}
)

func (r *repo) GetTickState(addr common.Address, tick int32) (*TickState, error) {
	key := GetTickStateKey(addr, tick).GetKey()
	bytes, err := r.db.Get(key)
	if err != nil {
		if IsNotExist(err) {
			return EmptyTickState, nil
		}
		return nil, err
	}

	tickState := NewTickState(tick)
	if err := tickState.UnmarshalBinary(bytes); err != nil {
		return nil, err
	}

	return tickState, nil
}

type TickStateCollector struct {
	collection map[common.Address][]*TickState
}

func NewTickStateCollector() *TickStateCollector {
	return &TickStateCollector{
		collection: make(map[common.Address][]*TickState),
	}
}

func (c *TickStateCollector) Add(addr common.Address, tickState *TickState) {
	if _, exists := c.collection[addr]; !exists {
		c.collection[addr] = make([]*TickState, 0)
	}
	c.collection[addr] = append(c.collection[addr], tickState)
}

func (c *TickStateCollector) Get() map[common.Address][]*TickState {
	return c.collection
}

func (r *repo) GetFromTo(from, to []byte) (map[common.Address][]*TickState, error) {
	entries, err := r.db.GetRange(from, to)
	if err != nil {
		return nil, err
	}

	collector := NewTickStateCollector()
	for _, entry := range entries {
		key := BytesToTickStateKey(entry.K())
		tickState := NewTickState(key.GetTick())
		if err := tickState.UnmarshalBinary(entry.V()); err != nil {
			return nil, err
		}
		collector.Add(key.GetAddress(), tickState)
	}

	return collector.Get(), nil
}

var (
	EmptyTickStates = make([]*TickState, 0)
)

func (r *repo) GetTickStates(addr common.Address, tickLower, tickUpper int32) ([]*TickState, error) {
	tickStatesByAddr, err := r.GetFromTo(GetTickStateKey(addr, tickLower).GetKey(), GetTickStateKey(addr, tickUpper).GetKey())
	if err != nil {
		return nil, err
	}
	if states, exists := tickStatesByAddr[addr]; exists {
		return states, nil
	}
	return EmptyTickStates, nil
}

func (r *repo) GetAllTicks() (map[common.Address][]*TickState, error) {
	return r.GetFromTo(MinKey.GetKey(), MaxKey.GetKey())
}

var HeightKey = []byte("1")

func (r *repo) SetHeight(height uint64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], height)
	return r.db.Set(HeightKey, buf[:])
}

func (r *repo) GetHeight() (uint64, error) {
	data, err := r.db.Get(HeightKey)
	if err != nil {
		if IsNotExist(err) {
			return 0, nil
		}

		return 0, err
	}

	if len(data) != 8 {
		return 0, nil
	}

	height := binary.BigEndian.Uint64(data)
	return height, nil
}
