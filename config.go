package main

import (
	"encoding/json"
	"os"
)

type LogConf struct {
	Async         bool   `json:"async"`
	BufferSize    int    `json:"buffer_size"`
	FlushInterval int    `json:"flush_interval"`
	Level         string `json:"level"`
}

type EthRPCConf struct {
	HTTP    string `json:"http"`
	Archive string `json:"archive"`
	WS      string `json:"ws"`
}

type BlockCrawlerConf struct {
	PoolSize   int    `json:"pool_size"`
	FromHeight uint64 `json:"from_height"`
}

type RedisConf struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type RocksDBConf struct {
	EnableLog            bool   `json:"enable_log"`
	BlockCacheSize       uint64 `json:"block_cache_size"`
	WriteBufferSize      uint64 `json:"write_buffer_size"`
	MaxWriteBufferNumber int    `json:"max_write_buffer_number"`
	DBPath               string `json:"db_path"`
}

type Config struct {
	Log          *LogConf          `json:"log"`
	EthRPC       *EthRPCConf       `json:"eth_rpc"`
	BlockCrawler *BlockCrawlerConf `json:"block_crawler"`
	Redis        *RedisConf        `json:"redis"`
	RocksDB      *RocksDBConf      `json:"rocksdb"`
}

var (
	defaultConfig = Config{
		Log: &LogConf{
			Async:         false,
			BufferSize:    1000000,
			FlushInterval: 1,
			Level:         "info",
		},
		EthRPC: &EthRPCConf{
			HTTP:    "https://bsc-dataseed.binance.org/",
			Archive: "https://bsc-dataseed.binance.org/",
			WS:      "ws://bsc-dataseed.binance.org/",
		},
		BlockCrawler: &BlockCrawlerConf{
			PoolSize:   1,
			FromHeight: 0,
		},
		Redis: &RedisConf{
			Addr:     "localhost:6379",
			Username: "",
			Password: "",
		},
		RocksDB: &RocksDBConf{
			EnableLog:            true,
			BlockCacheSize:       uint64(1024 * 1024 * 1024 * 1), // 1GB
			WriteBufferSize:      uint64(1024 * 1024 * 128),      // 128MB
			MaxWriteBufferNumber: 2,
			DBPath:               ".db",
		},
	}

	G = defaultConfig
)

func LoadConfig(name string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&G); err != nil {
		return err
	}

	return nil
}
