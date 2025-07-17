package main

import (
	"encoding/binary"
)

type TickStateDB interface {
	SaveTickState(k []byte, tick *TickState) error
	GetTickState(k []byte) (*TickState, error)
	GetTickStates(from, to []byte) ([]*TickState, error)
}

type HeightDB interface {
	GetHeight() (uint64, error)
	SetHeight(height uint64) error
}

type DBWrap interface {
	TickStateDB
	HeightDB
	close()
}

type rocksDBWrap struct {
	db *RocksDB
}

func (r *rocksDBWrap) close() {
	r.db.Close()
}

func NewDBWrap(db *RocksDB) DBWrap {
	return &rocksDBWrap{
		db: db,
	}
}

func (r *rocksDBWrap) SaveTickState(k []byte, tick *TickState) error {
	data, err := tick.MarshalBinary()
	if err != nil {
		return err
	}

	return r.db.Set(k, data)
}

func (r *rocksDBWrap) GetTickState(k []byte) (*TickState, error) {
	data, err := r.db.Get(k)
	if err != nil {
		return nil, err
	}

	tick := NewTickState(0)
	if err := tick.UnmarshalBinary(data); err != nil {
		return nil, err
	}

	return tick, nil
}

func (r *rocksDBWrap) GetTickStates(from, to []byte) ([]*TickState, error) {
	entries, err := r.db.GetRange(from, to)
	if err != nil {
		return nil, err
	}

	var ticks []*TickState
	for _, entry := range entries {
		tick := NewTickState(0)
		if err := tick.UnmarshalBinary(entry.V()); err != nil {
			return nil, err
		}
		ticks = append(ticks, tick)
	}

	return ticks, nil
}

var HeightKey = []byte("1")

func (r *rocksDBWrap) GetHeight() (uint64, error) {
	data, err := r.db.Get(HeightKey)
	if err != nil {
		if IsNotExist(err) {
			return 0, nil
		}

		return 0, err
	}

	if len(data) != 8 {
		//return 0, ErrInvalidHeightData
		return 0, nil // not exist
	}

	height := binary.BigEndian.Uint64(data)
	return height, nil
}

func (r *rocksDBWrap) SetHeight(height uint64) error {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], height)
	return r.db.Set(HeightKey, buf[:])
}
