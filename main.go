package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version information")
	var configFile string
	flag.StringVar(&configFile, "c", "config.json", "config file")
	flag.Parse()

	if showVersion {
		fmt.Println(GetVersion())
		os.Exit(0)
	}

	err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	bc := NewBlockCrawler(nil, nil, brs)
	sh := bc.GetStartHeight(0)
	brs := NewSequencer[*BlockReceipt](sh)
	//_ := NewSequencer[*BlockEvent](sh)

	dispatcher := NewTaskDispatcher(G.Bsc.WsEndpoint)
	worker := NewBlockCrawlerWorker(G.Bsc.WsEndpoint, 10, brs)
	worker.Start(ctx)
	dispatcher.MountTaskCommiter(worker)
	dispatcher.Start(ctx, sh)

	rocksDB, err := NewRocksDB("./db", &RocksDBOptions{
		BlockCacheSize:       1024 * 1024 * 1024 * 4,
		WriteBufferSize:      1024 * 1024 * 128,
		MaxWriteBufferNumber: 2,
	})

	dbWrap := NewDBWrap(rocksDB)
	actor := NewEventActor(dbWrap)

	for {
		br := worker.NextBlockReceipt()
		be := ParseBlock(br)
		err = actor.ActBlockEvent(be)
		if err != nil {
			Log.Fatal("TODO")
		}
	}
}
