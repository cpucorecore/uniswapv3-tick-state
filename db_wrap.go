package main

import (
	"encoding/binary"
	"errors"
)

type TickDB interface {
	GetTick(k []byte) (*Tick, error)
	SaveTick(k []byte, tick *Tick) error
}

type HeightDB interface {
	GetHeight() (uint64, error)
	SetHeight(height uint64) error
}

type DBWrap interface {
	TickDB
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

func (r *rocksDBWrap) GetTick(k []byte) (*Tick, error) {
	data, err := r.db.Get(k)
	if err != nil {
		return nil, err
	}

	tick := NewTick()
	if err := tick.UnmarshalBinary(data); err != nil {
		return nil, err
	}

	return tick, nil
}

func (r *rocksDBWrap) SaveTick(k []byte, tick *Tick) error {
	data, err := tick.MarshalBinary()
	if err != nil {
		return err
	}

	return r.db.Set(k, data)
}

var HeightKey = []byte("height")

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

var ErrInvalidHeightData = errors.New("invalid headerHeight data")
