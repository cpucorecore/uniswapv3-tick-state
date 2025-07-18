package main

import (
	"context"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	rocksDB, err := NewRocksDB(".db", &RocksDBOptions{
		EnableLog:            true,
		BlockCacheSize:       1024 * 1024 * 1024 * 1,
		WriteBufferSize:      1024 * 1024 * 128,
		MaxWriteBufferNumber: 2,
	})
	if err != nil {
		panic(err)
	}
	dbWrap := NewRepo(rocksDB)

	as := NewAPIServer(dbWrap)
	as.Start()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	reactor := NewEventReactor(dbWrap, wg)
	parser := NewBlockParser()
	parser.MountOutput(reactor)

	finishedHeight, err := dbWrap.GetHeight()
	if err != nil {
		Log.Fatal("failed to get finished height", zap.Error(err))
	}

	Log.Info(fmt.Sprintf("finished height: %d", finishedHeight))
	dispatcher := NewTaskDispatcher(G.EthRPC.WS)
	fromHeight := dispatcher.GetFromHeight(ctx, G.BlockCrawler.FromHeight, finishedHeight)
	blockSequencer := NewSequencer[*BlockReceipt](fromHeight)
	crawler := NewBlockCrawler(G.EthRPC.WS, 10, blockSequencer)
	crawler.MountOutput(parser)
	crawler.Start(ctx)

	dispatcher.MountOutput(crawler)
	dispatcher.Start(ctx, fromHeight)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		Log.Info("receive signal", zap.String("signal", sig.String()))
		dispatcher.Stop()
	}()

	Log.Info("waiting for retirement...")
	wg.Wait()
	Log.Info("now retire")
}
