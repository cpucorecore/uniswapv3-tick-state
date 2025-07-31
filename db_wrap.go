package main

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/linxGnu/grocksdb"
)

var (
	HeightKey            = []byte("1:")
	KeyPrefixTickState   = []byte("2:")
	KeyPrefixCurrentTick = []byte("3:")
	KeyPrefixTickSpacing = []byte("4:")
	KeyPrefixPoolHeight  = []byte("5:")
)

func makeCurrentTickKey(addr common.Address) [22]byte {
	var key [22]byte
	copy(key[:2], KeyPrefixCurrentTick)
	copy(key[2:22], addr[:])
	return key
}

func makeTickSpacingKey(addr common.Address) [22]byte {
	var key [22]byte
	copy(key[:2], KeyPrefixTickSpacing)
	copy(key[2:22], addr[:])
	return key
}

func makePoolHeightKey(addr common.Address) [22]byte {
	var key [22]byte
	copy(key[:2], KeyPrefixPoolHeight)
	copy(key[2:22], addr[:])
	return key
}

type TickStateDB interface {
	SetTickState(addr common.Address, tickState *TickState) error
	GetTickState(addr common.Address, tick int32) (*TickState, error)
	GetTickStates(addr common.Address, tickLower, tickUpper int32) ([]*TickState, error)
	GetPoolTickStates(addr common.Address) ([]*TickState, error)
	SetCurrentTick(addr common.Address, currentTick int32) error
	GetCurrentTick(addr common.Address) (int32, error)
	SetTickSpacing(addr common.Address, tickSpacing int32) error
	GetTickSpacing(addr common.Address) (int32, error)
	PoolExists(addr common.Address) (bool, error)
	SetPoolHeight(addr common.Address, height uint64) error
	GetPoolHeight(addr common.Address) (uint64, error)
	GetPoolState(addr common.Address) (*PoolState, error)
	SetPoolState(addr common.Address, poolTicks *PoolState) error
	DeletePoolState(addr common.Address) error
}

type HeightDB interface {
	SetHeight(height uint64) error
	GetHeight() (uint64, error)
}

type DB interface {
	TickStateDB
	HeightDB
	Close()
}

type rocksDBWrap struct {
	db *RocksDB
}

func (r *rocksDBWrap) Close() {
	r.db.Close()
}

func NewDB(db *RocksDB) DB {
	return &rocksDBWrap{
		db: db,
	}
}

func (r *rocksDBWrap) SetTickState(addr common.Address, tickState *TickState) error {
	key := GetTickStateKey(addr, tickState.Tick).GetKey()
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

func (r *rocksDBWrap) GetTickState(addr common.Address, tick int32) (*TickState, error) {
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

func (r *rocksDBWrap) GetFromTo(from, to []byte) (map[common.Address][]*TickState, error) {
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

func (r *rocksDBWrap) GetTickStates(addr common.Address, tickLower, tickUpper int32) ([]*TickState, error) {
	tickStatesByAddr, err := r.GetFromTo(GetTickStateKey(addr, tickLower).GetKey(), GetTickStateKey(addr, tickUpper).GetKey())
	if err != nil {
		return nil, err
	}
	if states, exists := tickStatesByAddr[addr]; exists {
		return states, nil
	}
	return EmptyTickStates, nil
}

func (r *rocksDBWrap) GetPoolTickStates(addr common.Address) ([]*TickState, error) {
	fk := GetTickStateKey(addr, MinTick)
	tk := GetTickStateKey(addr, MaxTick)
	tickStates, err := r.GetFromTo(fk.GetKey(), tk.GetKey())
	if err != nil {
		return nil, err
	}

	tickState, ok := tickStates[addr]
	if !ok {
		return EmptyTickStates, nil
	}

	return tickState, nil
}

func (r *rocksDBWrap) SetHeight(height uint64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], height)
	return r.db.Set(HeightKey, buf[:])
}

func (r *rocksDBWrap) GetHeight() (uint64, error) {
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

func (r *rocksDBWrap) SetCurrentTick(addr common.Address, currentTick int32) error {
	key := makeCurrentTickKey(addr)
	return r.db.Set(key[:], int32ToBytes(currentTick))
}

func (r *rocksDBWrap) GetCurrentTick(addr common.Address) (int32, error) {
	key := makeCurrentTickKey(addr)
	bytes, err := r.db.Get(key[:])
	if err != nil {
		return 0, err
	}

	return bytesToInt32(bytes), nil
}

func (r *rocksDBWrap) SetTickSpacing(addr common.Address, tickSpacing int32) error {
	key := makeTickSpacingKey(addr)
	return r.db.Set(key[:], int32ToBytes(tickSpacing))
}

func (r *rocksDBWrap) GetTickSpacing(addr common.Address) (int32, error) {
	key := makeTickSpacingKey(addr)
	bytes, err := r.db.Get(key[:])
	if err != nil {
		return 0, err
	}

	return bytesToInt32(bytes), nil
}

func IsNotExistErr(err error) bool {
	return errors.Is(err, ErrKeyNotFound)
}

func (r *rocksDBWrap) PoolExists(addr common.Address) (bool, error) {
	tickSpacing, err := r.GetTickSpacing(addr)
	if err != nil {
		if IsNotExistErr(err) {
			return false, nil
		}
		return false, err
	}

	return tickSpacing != 0, nil
}

func (r *rocksDBWrap) SetPoolHeight(addr common.Address, height uint64) error {
	key := makePoolHeightKey(addr)
	return r.db.Set(key[:], uint64ToBytes(height))
}

func (r *rocksDBWrap) GetPoolHeight(addr common.Address) (uint64, error) {
	key := makePoolHeightKey(addr)
	bytes, err := r.db.Get(key[:])
	if err != nil {
		return 0, err
	}

	return bytesToUint64(bytes), nil
}

func (r *rocksDBWrap) GetPoolState(addr common.Address) (*PoolState, error) {
	height, err := r.GetPoolHeight(addr)
	if err != nil {
		return nil, err
	}

	tickSpacing, err := r.GetTickSpacing(addr)
	if err != nil {
		return nil, err
	}

	tick, err := r.GetCurrentTick(addr)
	if err != nil {
		return nil, err
	}

	tickStates, err := r.GetPoolTickStates(addr)
	if err != nil {
		return nil, err
	}

	return &PoolState{
		Global: &PoolGlobalState{
			Height:      big.NewInt(int64(height)),
			TickSpacing: big.NewInt(int64(tickSpacing)),
			Tick:        big.NewInt(int64(tick)),
		},
		TickStates: tickStates,
	}, nil
}

func (r *rocksDBWrap) SetPoolState(addr common.Address, poolState *PoolState) error {
	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	heightKey := makePoolHeightKey(addr)
	batch.Put(heightKey[:], uint64ToBytes(poolState.Global.Height.Uint64()))

	tickKey := makeCurrentTickKey(addr)
	batch.Put(tickKey[:], int32ToBytes(int32(poolState.Global.Tick.Int64())))

	spacingKey := makeTickSpacingKey(addr)
	batch.Put(spacingKey[:], int32ToBytes(int32(poolState.Global.TickSpacing.Int64())))

	for _, ts := range poolState.TickStates {
		tickStateKey := GetTickStateKey(addr, ts.Tick).GetKey()
		value, err := ts.MarshalBinary()
		if err != nil {
			return err
		}
		batch.Put(tickStateKey, value)
	}

	return r.db.WriteBatch(batch)
}

func (r *rocksDBWrap) DeletePoolState(addr common.Address) error {
	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()

	heightKey := makePoolHeightKey(addr)
	batch.Delete(heightKey[:])

	spacingKey := makeTickSpacingKey(addr)
	batch.Delete(spacingKey[:])

	tickKey := makeCurrentTickKey(addr)
	batch.Delete(tickKey[:])

	startKey := GetTickStateKey(addr, MinTick).GetKey()
	endKey := GetTickStateKey(addr, MaxTick).GetKey()
	batch.DeleteRange(startKey, endKey)

	return r.db.WriteBatch(batch)
}
