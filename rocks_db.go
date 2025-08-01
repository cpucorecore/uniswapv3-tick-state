package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/linxGnu/grocksdb"
)

type EntryK interface {
	K() []byte
}

type EntryV interface {
	V() []byte
}
type KVEntry interface {
	EntryK
	EntryV
}

type bytesEntry struct {
	key []byte
	val []byte
}

func (e *bytesEntry) K() []byte { return e.key }
func (e *bytesEntry) V() []byte { return e.val }

type RocksDBOptions struct {
	EnableLog            bool
	BlockCacheSize       uint64
	WriteBufferSize      uint64
	MaxWriteBufferNumber int
}

type RocksDB struct {
	db *grocksdb.DB
	ro *grocksdb.ReadOptions
	wo *grocksdb.WriteOptions
}

func logRocksDBStats(db *grocksdb.DB) {
	stats := db.GetProperty("rocksdb.stats")
	fmt.Println(stats)

	blockCacheHitStr := db.GetProperty("rocksdb.block.cache.hit")
	blockCacheMissStr := db.GetProperty("rocksdb.block.cache.miss")

	blockCacheHit, _ := strconv.ParseUint(blockCacheHitStr, 10, 64)
	blockCacheMiss, _ := strconv.ParseUint(blockCacheMissStr, 10, 64)

	hitRate := float64(blockCacheHit) * 100 / float64(blockCacheHit+blockCacheMiss)
	fmt.Printf("BlockCache Hit: %d, Miss: %d, HitRate: %.2f%%\n", blockCacheHit, blockCacheMiss, hitRate)
}

func NewRocksDB(name string, optsConf *RocksDBOptions) (*RocksDB, error) {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	if optsConf.BlockCacheSize > 0 {
		options := grocksdb.NewDefaultBlockBasedTableOptions()
		cache := grocksdb.NewLRUCache(optsConf.BlockCacheSize)
		options.SetBlockCache(cache)
		opts.SetBlockBasedTableFactory(options)
	}

	if optsConf.WriteBufferSize > 0 {
		opts.SetWriteBufferSize(optsConf.WriteBufferSize)
	}
	if optsConf.MaxWriteBufferNumber > 0 {
		opts.SetMaxWriteBufferNumber(optsConf.MaxWriteBufferNumber)
	}

	db, err := grocksdb.OpenDb(opts, name)
	if err != nil {
		return nil, fmt.Errorf("failed to open RocksDB: %v", err)
	}

	if optsConf.EnableLog {
		go func() {
			for {
				logRocksDBStats(db)
				time.Sleep(time.Minute)
			}
		}()
	}

	return &RocksDB{
		db: db,
		ro: grocksdb.NewDefaultReadOptions(),
		wo: grocksdb.NewDefaultWriteOptions(),
	}, nil
}

func (r *RocksDB) Close() {
	r.ro.Destroy()
	r.wo.Destroy()
	r.db.Close()
}

func (r *RocksDB) Get(key []byte) ([]byte, error) {
	slice, err := r.db.Get(r.ro, key)
	if err != nil {
		return nil, err
	}
	defer slice.Free()

	if !slice.Exists() {
		return nil, nil
	}

	return append([]byte{}, slice.Data()...), nil
}

func (r *RocksDB) Set(key, value []byte) error {
	return r.db.Put(r.wo, key, value)
}

func (r *RocksDB) Del(key []byte) error {
	return r.db.Delete(r.wo, key)
}

func (r *RocksDB) GetRange(from, to []byte) ([]KVEntry, error) {
	it := r.db.NewIterator(r.ro)
	defer it.Close()

	var result []KVEntry
	for it.Seek(from); it.Valid(); it.Next() {
		key := it.Key()
		keyData := append([]byte{}, key.Data()...)
		if string(keyData) > string(to) {
			key.Free()
			break
		}

		value := it.Value()
		valData := append([]byte{}, value.Data()...)
		result = append(result, &bytesEntry{key: keyData, val: valData})
		key.Free()
		value.Free()
	}

	if err := it.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RocksDB) WriteBatch(batch *grocksdb.WriteBatch) error {
	return r.db.Write(r.wo, batch)
}
