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
	GetPoolTicks(address common.Address) ([]*TickState, error)
	SetCurrentTick(address common.Address, currentTick int32) error
	GetCurrentTick(address common.Address) (int32, error)
	SetTickSpacing(address common.Address, tickSpacing int32) error
	GetTickSpacing(address common.Address) (int32, error)
	PoolExists(address common.Address) (bool, error)
	SetPoolHeight(address common.Address, height uint64) error
	GetPoolHeight(address common.Address) (uint64, error)
	GetPoolState(poolAddr common.Address) (*PoolTicks, error)
	SetPoolState(poolAddr common.Address, poolTicks *PoolTicks) error
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

func (r *repo) GetPoolTicks(address common.Address) ([]*TickState, error) {
	fk := GetTickStateKey(address, minTick)
	tk := GetTickStateKey(address, maxTick)
	tickStates, err := r.GetFromTo(fk.GetKey(), tk.GetKey())
	if err != nil {
		return nil, err
	}

	tickState, ok := tickStates[address]
	if !ok {
		return EmptyTickStates, nil
	}

	return tickState, nil
}

var HeightKey = []byte("1:")

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

var (
	KeyPrefixCurrentTick = []byte("3:")
	KeyPrefixTickSpacing = []byte("4:")
	KeyPrefixPoolHeight  = []byte("5:")
)

func int32ToBytes(n int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(n))
	return buf
}

func bytesToInt32(data []byte) int32 {
	return int32(binary.BigEndian.Uint32(data))
}

func uint64ToBytes(n uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, n)
	return buf
}

func bytesToUint64(data []byte) uint64 {
	return binary.BigEndian.Uint64(data)
}

func (r *repo) SetCurrentTick(address common.Address, currentTick int32) error {
	var key [22]byte
	copy(key[:2], KeyPrefixCurrentTick)
	copy(key[2:22], address[:])

	return r.db.Set(key[:], int32ToBytes(currentTick))
}

func (r *repo) GetCurrentTick(address common.Address) (int32, error) {
	var key [22]byte
	copy(key[:2], KeyPrefixCurrentTick)
	copy(key[2:22], address[:])

	bytes, err := r.db.Get(key[:])
	if err != nil {
		return 0, err
	}

	return bytesToInt32(bytes), nil
}

func (r *repo) SetTickSpacing(address common.Address, tickSpacing int32) error {
	var key [22]byte
	copy(key[:2], KeyPrefixTickSpacing)
	copy(key[2:22], address[:])

	return r.db.Set(key[:], int32ToBytes(tickSpacing))
}

func (r *repo) GetTickSpacing(address common.Address) (int32, error) {
	var key [22]byte
	copy(key[:2], KeyPrefixTickSpacing)
	copy(key[2:22], address[:])

	bytes, err := r.db.Get(key[:])
	if err != nil {
		return 0, err
	}

	return bytesToInt32(bytes), nil
}

func (r *repo) PoolExists(address common.Address) (bool, error) {
	tickSpacing, err := r.GetTickSpacing(address)
	if err != nil {
		return false, err
	}

	return tickSpacing != 0, nil
}

func (r *repo) SetPoolHeight(address common.Address, height uint64) error {
	var key [22]byte
	copy(key[:2], KeyPrefixPoolHeight)
	copy(key[2:22], address[:])

	return r.db.Set(key[:], uint64ToBytes(height))
}

func (r *repo) GetPoolHeight(address common.Address) (uint64, error) {
	var key [22]byte
	copy(key[:2], KeyPrefixPoolHeight)
	copy(key[2:22], address[:])

	bytes, err := r.db.Get(key[:])
	if err != nil {
		return 0, err
	}

	return bytesToUint64(bytes), nil
}

func (r *repo) GetPoolState(poolAddr common.Address) (*PoolTicks, error) {
	ok, err := r.PoolExists(poolAddr)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil // TODO check
	}

	height, err := r.GetPoolHeight(poolAddr)
	if err != nil {
		return nil, err
	}

	tickSpacing, err := r.GetTickSpacing(poolAddr)
	if err != nil {
		return nil, err
	}

	tick, err := r.GetCurrentTick(poolAddr)
	if err != nil {
		return nil, err
	}

	ticks, err := r.GetPoolTicks(poolAddr)
	if err != nil {
		return nil, err
	}

	return &PoolTicks{
		State: &PoolState{
			Height:      big.NewInt(int64(height)),
			TickSpacing: big.NewInt(int64(tickSpacing)),
			Tick:        big.NewInt(int64(tick)),
		},
		Ticks: ticks,
	}, nil
}

func (r *repo) SetPoolState(poolAddr common.Address, poolTicks *PoolTicks) error {
	// TODO mutex lock
	err := r.SetPoolHeight(poolAddr, poolTicks.State.Height.Uint64())
	if err != nil {
		return err
	}

	err = r.SetCurrentTick(poolAddr, int32(poolTicks.State.Tick.Int64()))
	if err != nil {
		return err
	}

	err = r.SetTickSpacing(poolAddr, int32(poolTicks.State.TickSpacing.Int64()))
	if err != nil {
		return err
	}
	for _, ts := range poolTicks.Ticks {
		err = r.SetTickState(poolAddr, ts.Tick, ts)
		if err != nil {
			return err
		}
	}

	return nil
}
