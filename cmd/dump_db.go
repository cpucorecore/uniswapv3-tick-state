package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"github.com/linxGnu/grocksdb"
)

func main() {
	var dbPath string
	flag.StringVar(&dbPath, "db", "/Users/sky/GolandProjects/uniswapv3-tick-state/.db", "RocksDB path")
	flag.Parse()

	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(false)
	db, err := grocksdb.OpenDb(opts, dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open rocksdb failed: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	readOpts := grocksdb.NewDefaultReadOptions()
	defer readOpts.Destroy()

	it := db.NewIterator(readOpts)
	defer it.Close()
	it.SeekToFirst()
	for ; it.Valid(); it.Next() {
		key := it.Key().Data()
		val := it.Value().Data()
		fmt.Printf("%s:%s\n", hex.EncodeToString(key), hex.EncodeToString(val))
		it.Key().Free()
		it.Value().Free()
	}
}
