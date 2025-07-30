package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-redis/redis/v8"
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
	db := NewDB(rocksDB)

	redisCli := redis.NewClient(&redis.Options{
		Addr:     G.Redis.Addr,
		Username: G.Redis.Username,
		Password: G.Redis.Password,
	})
	cache := NewTwoTierCache(redisCli)

	psg := NewPoolStateGetter(cache, db, G.EthRPC.HTTP)
	as := NewAPIServer(psg)
	as.Start()

	wg := &sync.WaitGroup{}
	wg.Add(1)
	reactor := NewEventReactor(wg, db, psg)
	parser := NewBlockParser()
	parser.MountOutput(reactor)

	finishedHeight, err := db.GetHeight()
	if err != nil {
		Log.Fatal("failed to get finished height", zap.Error(err))
	}

	Log.Info(fmt.Sprintf("finished height: %d", finishedHeight))
	dispatcher := NewTaskDispatcher(G.EthRPC.WS)
	fromHeight := dispatcher.GetFromHeight(ctx, G.BlockCrawler.FromHeight, finishedHeight)
	blockSequencer := NewSequencer[*BlockReceipt](fromHeight - 1)
	crawler := NewBlockCrawler(G.EthRPC.WS, 1, blockSequencer)
	crawler.MountOutput(parser)
	crawler.Start(ctx)

	dispatcher.MountOutput(crawler)
	dispatcher.Start(ctx, fromHeight, true)

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
