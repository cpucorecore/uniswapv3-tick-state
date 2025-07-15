package main

import (
	"errors"
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
type Entry interface {
	EntryK
	EntryV
}

type kvEntry struct {
	key []byte
	val []byte
}

func (e *kvEntry) K() []byte { return e.key }
func (e *kvEntry) V() []byte { return e.val }

type RocksDBOptions struct {
	BlockCacheSize       uint64
	WriteBufferSize      uint64
	MaxWriteBufferNumber int
}

type RocksDB struct {
	db *grocksdb.DB
	ro *grocksdb.ReadOptions
	wo *grocksdb.WriteOptions
}

func NewRocksDB(path string, optsConf *RocksDBOptions) (*RocksDB, error) {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	if optsConf != nil && optsConf.BlockCacheSize > 0 {
		options := grocksdb.NewDefaultBlockBasedTableOptions()
		cache := grocksdb.NewLRUCache(optsConf.BlockCacheSize)
		options.SetBlockCache(cache)
		opts.SetBlockBasedTableFactory(options)
	}

	if optsConf != nil && optsConf.WriteBufferSize > 0 {
		opts.SetWriteBufferSize(optsConf.WriteBufferSize)
	}
	if optsConf != nil && optsConf.MaxWriteBufferNumber > 0 {
		opts.SetMaxWriteBufferNumber(optsConf.MaxWriteBufferNumber)
	}

	db, err := grocksdb.OpenDb(opts, path)
	if err != nil {
		return nil, fmt.Errorf("failed to open RocksDB: %v", err)
	}

	stats := db.GetProperty("rocksdb.stats")
	fmt.Println(stats)

	blockCacheHitStr := db.GetProperty("rocksdb.block.cache.hit")
	blockCacheMissStr := db.GetProperty("rocksdb.block.cache.miss")

	blockCacheHit, _ := strconv.ParseUint(blockCacheHitStr, 10, 64)
	blockCacheMiss, _ := strconv.ParseUint(blockCacheMissStr, 10, 64)

	hitRate := float64(blockCacheHit) * 100 / float64(blockCacheHit+blockCacheMiss)
	fmt.Printf("BlockCache Hit: %d, Miss: %d, HitRate: %.2f%%\n", blockCacheHit, blockCacheMiss, hitRate)

	go func() {
		for {
			time.Sleep(time.Minute)
			stats := db.GetProperty("rocksdb.stats")
			blockCacheHitStr := db.GetProperty("rocksdb.block.cache.hit")
			blockCacheMissStr := db.GetProperty("rocksdb.block.cache.miss")

			blockCacheHit, _ := strconv.ParseUint(blockCacheHitStr, 10, 64)
			blockCacheMiss, _ := strconv.ParseUint(blockCacheMissStr, 10, 64)

			hitRate := float64(blockCacheHit) * 100 / float64(blockCacheHit+blockCacheMiss)
			fmt.Printf("[RocksDB] BlockCache Hit: %d, Miss: %d, HitRate: %.2f%%\n", blockCacheHit, blockCacheMiss, hitRate)
			fmt.Println(stats)
		}
	}()

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

var ErrKeyNotFound = errors.New("key not found")

func (r *RocksDB) Get(key []byte) ([]byte, error) {
	slice, err := r.db.Get(r.ro, key)
	if err != nil {
		return nil, err
	}
	defer slice.Free()

	if !slice.Exists() {
		return nil, ErrKeyNotFound
	}

	return slice.Data(), nil
}

func (r *RocksDB) Set(key, value []byte) error {
	return r.db.Put(r.wo, key, value)
}

func (r *RocksDB) Del(key []byte) error {
	return r.db.Delete(r.wo, key)
}

func (r *RocksDB) GetRange(start, end []byte) ([]Entry, error) {
	it := r.db.NewIterator(r.ro)
	defer it.Close()
	var result []Entry
	for it.Seek(start); it.Valid(); it.Next() {
		key := it.Key()
		keyData := append([]byte{}, key.Data()...)
		if string(keyData) >= string(end) {
			key.Free()
			break
		}
		value := it.Value()
		valData := append([]byte{}, value.Data()...)
		result = append(result, &kvEntry{key: keyData, val: valData})
		key.Free()
		value.Free()
	}
	if err := it.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RocksDB) SetAll(data map[string][]byte) error {
	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	for k, v := range data {
		batch.Put([]byte(k), v)
	}
	return r.db.Write(r.wo, batch)
}

func (r *RocksDB) SetAll2(data []Entry) error {
	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	for _, e := range data {
		batch.Put(e.K(), e.V())
	}
	return r.db.Write(r.wo, batch)
}
