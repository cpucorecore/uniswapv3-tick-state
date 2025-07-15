package main

import (
	"errors"
	"fmt"
	"github.com/linxGnu/grocksdb"
	"strconv"
	"time"
)

// Entry represents a key-value pair for batch operations (interface)
type Entry interface {
	K() []byte
	V() []byte
}

// kvEntry is a simple implementation of Entry for internal use
// (unexported, but returned as Entry interface)
type kvEntry struct {
	key []byte
	val []byte
}

func (e *kvEntry) K() []byte { return e.key }
func (e *kvEntry) V() []byte { return e.val }

// RocksDBOptions 用于配置RocksDB实例的缓存等参数
// BlockCacheSize/WriteBufferSize 单位为字节
// MaxWriteBufferNumber 为MemTable个数
// 其他参数可按需扩展

type RocksDBOptions struct {
	BlockCacheSize       int    // 块缓存大小，单位字节
	WriteBufferSize      uint64 // 单个MemTable大小，单位字节
	MaxWriteBufferNumber int    // MemTable个数
}

// RocksDB represents a wrapper around the RocksDB database
type RocksDB struct {
	db *grocksdb.DB
	ro *grocksdb.ReadOptions
	wo *grocksdb.WriteOptions
}

// NewRocksDB initializes a new RocksDB instance with the given path and options.
// It returns a pointer to the RocksDB instance or an error if initialization fails.
// 新增 options 参数用于配置缓存大小等
func NewRocksDB(path string, optsConf *RocksDBOptions) (*RocksDB, error) {
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	// 配置 Block Cache
	if optsConf != nil && optsConf.BlockCacheSize > 0 {
		bbto := grocksdb.NewDefaultBlockBasedTableOptions()
		cache := grocksdb.NewLRUCache(uint64(optsConf.BlockCacheSize))
		bbto.SetBlockCache(cache)
		opts.SetBlockBasedTableFactory(bbto)
	}

	// 配置 Write Buffer
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

	// 获取统计字符串
	stats := db.GetProperty("rocksdb.stats")
	fmt.Println(stats)

	// 获取命中/未命中次数
	blockCacheHitStr := db.GetProperty("rocksdb.block.cache.hit")
	blockCacheMissStr := db.GetProperty("rocksdb.block.cache.miss")

	// 转换为整数
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

			// 转换为整数
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

// Close closes the RocksDB instance
func (r *RocksDB) Close() {
	r.ro.Destroy()
	r.wo.Destroy()
	r.db.Close()
}

// Get retrieves the value for the given key ([]byte)
func (r *RocksDB) Get(key []byte) ([]byte, error) {
	slice, err := r.db.Get(r.ro, key)
	if err != nil {
		return nil, err
	}
	defer slice.Free()

	if !slice.Exists() {
		return nil, errors.New("key not found")
	}
	return append([]byte{}, slice.Data()...), nil
}

// Set sets the value for the given key ([]byte)
func (r *RocksDB) Set(key, value []byte) error {
	return r.db.Put(r.wo, key, value)
}

// Del deletes the value for the given key ([]byte)
func (r *RocksDB) Del(key []byte) error {
	return r.db.Delete(r.wo, key)
}

// GetRange retrieves all key-value pairs where keys are in the range [start, end)
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

// SetAll sets multiple key-value pairs from a map (key, value都是[]byte)
func (r *RocksDB) SetAll(data map[string][]byte) error {
	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	for k, v := range data {
		batch.Put([]byte(k), v)
	}
	return r.db.Write(r.wo, batch)
}

// SetAll2 sets multiple key-value pairs from a slice of Entry接口
func (r *RocksDB) SetAll2(data []Entry) error {
	batch := grocksdb.NewWriteBatch()
	defer batch.Destroy()
	for _, e := range data {
		batch.Put(e.K(), e.V())
	}
	return r.db.Write(r.wo, batch)
}
