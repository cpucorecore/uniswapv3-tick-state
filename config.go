package main

import (
	"encoding/json"
	"os"
)

type LogConf struct {
	Async         bool `json:"async"`
	BufferSize    int  `json:"buffer_size"`
	FlushInterval int  `json:"flush_interval"`
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

type Config struct {
	Log          *LogConf          `json:"log"`
	EthRPC       *EthRPCConf       `json:"eth_rpc"`
	BlockCrawler *BlockCrawlerConf `json:"block_crawler"`
}

var (
	defaultConfig = Config{
		Log: &LogConf{
			Async:         false,
			BufferSize:    1000000,
			FlushInterval: 1,
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
