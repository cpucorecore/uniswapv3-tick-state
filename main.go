package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "show version information")

	var configFile string
	flag.StringVar(&configFile, "c", "config.json", "config file")

	var dbPath string
	flag.StringVar(&dbPath, "db", "", "database path (overrides config file)")

	// 套利监控参数
	var arbitrageMode bool
	flag.BoolVar(&arbitrageMode, "arbitrage", false, "启用套利监控模式")
	var pool1Addr string
	flag.StringVar(&pool1Addr, "pool1", "", "第一个池子地址")
	var pool2Addr string
	flag.StringVar(&pool2Addr, "pool2", "", "第二个池子地址")

	flag.Parse()

	if showVersion {
		fmt.Println(GetVersion())
		os.Exit(0)
	}

	err := LoadConfig(configFile)
	if err != nil {
		panic(err)
	}

	InitLogger()

	if dbPath != "" {
		G.RocksDB.DBPath = dbPath
	}

	ctx := context.Background()

	rocksDB, err := NewRocksDB(G.RocksDB.DBPath, &RocksDBOptions{
		EnableLog:            G.RocksDB.EnableLog,
		BlockCacheSize:       G.RocksDB.BlockCacheSize,
		WriteBufferSize:      G.RocksDB.WriteBufferSize,
		MaxWriteBufferNumber: G.RocksDB.MaxWriteBufferNumber,
	})
	if err != nil {
		panic(err)
	}
	db := NewDB(rocksDB)
	db = NewSafeDB(db)

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

	finishedHeight, err := db.GetFinishHeight()
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

	// 启动套利监控（如需要）
	var arbitrageStop func()
	if arbitrageMode {
		if pool1Addr == "" || pool2Addr == "" {
			Log.Fatal("套利监控模式需要提供两个池子地址 (-pool1 和 -pool2)")
		}
		pool1 := common.HexToAddress(pool1Addr)
		pool2 := common.HexToAddress(pool2Addr)
		Log.Info("启动套利监控", zap.String("pool1", pool1.Hex()), zap.String("pool2", pool2.Hex()))
		pools := []common.Address{pool1, pool2}
		arbitrageMonitor := NewArbitrageMonitor(psg, pools)
		arbitrageMonitor.Start()
		arbitrageStop = arbitrageMonitor.Stop
		go func() {
			for opportunity := range arbitrageMonitor.GetOpportunities() {
				opportunity.PrintOpportunity()
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		Log.Info("receive signal", zap.String("signal", sig.String()))
		dispatcher.Stop()
		if arbitrageStop != nil {
			arbitrageStop()
		}
	}()

	wg.Wait()
	Log.Info("done")
}
