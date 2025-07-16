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

type BlockCrawlerWorker interface {
	Start(ctx context.Context)
	Output[uint64]
	OutputMountable[*BlockReceipt]
}

type blockCrawler struct {
	inputQueue           chan uint64
	ethClient            *ethclient.Client
	pool                 *ants.Pool
	outputSequencer      Sequencer[*BlockReceipt]
	outputBuffer         chan *BlockReceipt
	blockReceiptReceiver Output[*BlockReceipt]
}

func (c *blockCrawler) PutInput(height uint64) {
	c.inputQueue <- height
}

func (c *blockCrawler) FinInput() {
	close(c.inputQueue)
}

func (c *blockCrawler) MountOutput(blockReceiptReceiver Output[*BlockReceipt]) {
	c.blockReceiptReceiver = blockReceiptReceiver
}

func NewBlockCrawler(url string, poolSize int, sequencer Sequencer[*BlockReceipt]) BlockCrawlerWorker {
	pool, err := ants.NewPool(poolSize)
	if err != nil {
		panic(err)
	}

	ethClient, err := ethclient.Dial(url)
	if err != nil {
		Log.Fatal("failed to connect to Ethereum client", zap.Error(err))
	}

	return &blockCrawler{
		inputQueue:      make(chan uint64, 1),
		ethClient:       ethClient,
		pool:            pool,
		outputSequencer: sequencer,
		outputBuffer:    make(chan *BlockReceipt, 1),
	}
}

func (c *blockCrawler) getBlock(ctx context.Context, height uint64) (*BlockReceipt, error) {
	receipts, err := c.ethClient.BlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(height)))

	if err != nil {
		return nil, err
	}

	return &BlockReceipt{
		Height:   height,
		Receipts: receipts,
	}, nil
}

func (c *blockCrawler) getBlockRetry(ctx context.Context, height uint64) (*BlockReceipt, error) {
	return retry.DoWithData(func() (*BlockReceipt, error) {
		return c.getBlock(ctx, height)
	}, infiniteAttempts, retryDelay)
}

func (c *blockCrawler) startCommitOutput() {
	go func() {
		for {
			select {
			case blockReceipt, ok := <-c.outputBuffer:
				if !ok {
					Log.Info("no more block receipt")
					c.blockReceiptReceiver.FinInput()
					return
				}

				c.blockReceiptReceiver.PutInput(blockReceipt)
			}
		}
	}()
}

func (c *blockCrawler) Start(ctx context.Context) {
	c.startCommitOutput()

	go func() {
		wg := &sync.WaitGroup{}
	tagFor:
		for {
			select {
			case height, ok := <-c.inputQueue:
				if !ok {
					Log.Info("no more task")
					break tagFor
				}

				wg.Add(1)
				c.pool.Submit(func() {
					defer wg.Done()
					blockReceipt, err := c.getBlockRetry(ctx, height)
					if err != nil {
						Log.Error("get block err", zap.Uint64("headerHeight", height), zap.Error(err))
						return
					}
					Log.Info("get block success", zap.Uint64("headerHeight", height))
					c.outputSequencer.Commit(blockReceipt, c.outputBuffer)
				})
			}
		}

		wg.Wait()
		Log.Info("block crawler finished all tasks")
		close(c.outputBuffer)
	}()
}
