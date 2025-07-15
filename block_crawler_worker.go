package main

import (
	"context"
	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
	"sync"
)

type BlockReceiptIterable interface {
	NextBlockReceipt() *BlockReceipt
}

type BlockCrawlerWorker interface {
	Start(ctx context.Context)
	TaskCommiter
	BlockReceiptIterable
}

type blockCrawlerWorker struct {
	taskQueue       chan uint64
	ethClient       *ethclient.Client
	pool            *ants.Pool
	outputSequencer Sequencer[*BlockReceipt]
	outputBuffer    chan *BlockReceipt
}

func NewBlockCrawlerWorker(url string, poolSize int, brs Sequencer[*BlockReceipt]) BlockCrawlerWorker {
	pool, err := ants.NewPool(poolSize)
	if err != nil {
		panic(err)
	}

	ethClient, err := ethclient.Dial(url)
	if err != nil {
		Log.Fatal("failed to connect to Ethereum client", zap.Error(err))
	}

	return &blockCrawlerWorker{
		taskQueue:       make(chan uint64, 100),
		ethClient:       ethClient,
		pool:            pool,
		outputSequencer: brs,
		outputBuffer:    make(chan *BlockReceipt, 100),
	}
}

func (w *blockCrawlerWorker) getBlock(ctx context.Context, height uint64) (*BlockReceipt, error) {
	receipts, err := w.ethClient.BlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(height)))

	if err != nil {
		return nil, err
	}

	return &BlockReceipt{
		Height:   height,
		Receipts: receipts,
	}, nil
}

func (w *blockCrawlerWorker) getBlockRetry(ctx context.Context, height uint64) (*BlockReceipt, error) {
	return retry.DoWithData(func() (*BlockReceipt, error) {
		return w.getBlock(ctx, height)
	}, infiniteAttempts, retryDelay)
}

func (w *blockCrawlerWorker) NoMoreOutput() {
	close(w.outputBuffer)
}

func (w *blockCrawlerWorker) Start(ctx context.Context) {
	go func() {
		wg := &sync.WaitGroup{}
	tagFor:
		for {
			select {
			case height, ok := <-w.taskQueue:
				if !ok {
					Log.Info("blockCrawler queue closed")
					break tagFor
				}

				wg.Add(1)
				w.pool.Submit(func() {
					defer wg.Done()
					bw, err := w.getBlockRetry(ctx, height)
					if err != nil {
						Log.Error("get block err", zap.Uint64("headerHeight", height), zap.Error(err))
						return
					}
					w.outputSequencer.Commit(bw, w.outputBuffer)
				})
			}
		}

		taskNumber := w.pool.Waiting()
		Log.Debug("wait block getter task finish", zap.Int("taskNumber", taskNumber))
		wg.Wait()
		Log.Debug("all block getter task finish")
		w.NoMoreOutput()
	}()
}

func (w *blockCrawlerWorker) NextBlockReceipt() *BlockReceipt {
	return <-w.outputBuffer
}

func (w *blockCrawlerWorker) CommitTask(height uint64) {
	w.taskQueue <- height
}

func (w *blockCrawlerWorker) NoMoreTask() {
	close(w.taskQueue)
}

var _ TaskCommiter = (*blockCrawlerWorker)(nil)
var _ BlockCrawlerWorker = (*blockCrawlerWorker)(nil)
