package main

import (
	"context"
	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/panjf2000/ants/v2"
	"go.uber.org/zap"
	"sync"
	"time"
)

type BlockCrawler interface {
	Start()
	GetStartHeight(startHeight uint64) uint64
	StartDispatch(startHeight uint64)
	Stop()
	GetBlockAsync(height uint64)
	Iterable
}

type blockCrawler struct {
	ctx              context.Context
	ethClient        *ethclient.Client
	wsEthClient      *ethclient.Client
	queue            chan uint64
	buffer           chan *BlockReceipt
	workPool         *ants.Pool
	stopped          bool
	stoppedLock      sync.RWMutex
	blockHeaderChan  chan *ethtypes.Header
	sequencer        Sequencer[*BlockReceipt]
	headerHeightLock sync.RWMutex
	headerHeight     uint64
}

func NewBlockCrawler(
	ethClient *ethclient.Client,
	wsEthClient *ethclient.Client,
	sequencer Sequencer[*BlockReceipt],
) BlockCrawler {
	workPool, err := ants.NewPool(10) // TODO
	if err != nil {
		panic(err)
	}

	return &blockCrawler{
		ctx:             context.Background(),
		ethClient:       ethClient,
		wsEthClient:     wsEthClient,
		queue:           make(chan uint64, 10),        // TODO
		buffer:          make(chan *BlockReceipt, 10), // TODO
		workPool:        workPool,
		blockHeaderChan: make(chan *ethtypes.Header, 100),
		sequencer:       sequencer,
	}
}

func (bc *blockCrawler) getBlock(height uint64) (*BlockReceipt, error) {
	receipts, err := bc.ethClient.BlockReceipts(bc.ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(height)))

	if err != nil {
		return nil, err
	}

	return &BlockReceipt{
		Height:   height,
		Receipts: receipts,
	}, nil
}

var (
	infiniteAttempts = retry.Attempts(0) // infinite attempts
	retryDelay       = retry.Delay(time.Millisecond * 100)
)

func (bc *blockCrawler) getBlockRetry(height uint64) (*BlockReceipt, error) {
	return retry.DoWithData(func() (*BlockReceipt, error) {
		return bc.getBlock(height)
	}, infiniteAttempts, retryDelay)
}

func (bc *blockCrawler) GetBlockAsync(blockNumber uint64) {
	bc.queue <- blockNumber
}

func (bc *blockCrawler) Next() *BlockReceipt {
	return <-bc.buffer
}

func (bc *blockCrawler) latestHeight() (uint64, error) {
	blockNumber, err := bc.ethClient.BlockNumber(bc.ctx)
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

func (bc *blockCrawler) latestHeightRetry() (uint64, error) {
	return retry.DoWithData(func() (uint64, error) {
		return bc.latestHeight()
	}, infiniteAttempts, retryDelay)
}

func (bc *blockCrawler) Start() {
	go func() {
		wg := &sync.WaitGroup{}
	tagFor:
		for {
			select {
			case blockNumber, ok := <-bc.queue:
				if !ok {
					Log.Info("blockCrawler queue closed")
					break tagFor
				}

				wg.Add(1)
				bc.workPool.Submit(func() {
					defer wg.Done()
					bw, err := bc.getBlockRetry(blockNumber)
					if err != nil {
						Log.Error("get block err", zap.Uint64("headerHeight", blockNumber), zap.Error(err))
						return
					}
					bc.sequencer.Commit(bw, bc.buffer)
				})
			}
		}

		taskNumber := bc.workPool.Waiting()
		Log.Debug("wait block getter task finish", zap.Int("taskNumber", taskNumber))
		wg.Wait()
		Log.Debug("all block getter task finish")
		close(bc.buffer)
	}()
}

func (bc *blockCrawler) GetStartHeight(startBlockNumber uint64) uint64 {
	newestBlockNumber, err := bc.ethClient.BlockNumber(bc.ctx)
	if err != nil {
		Log.Fatal("ethClient.BigIntHeight() err", zap.Error(err))
	}

	if startBlockNumber == 0 {
		//startBlockNumber = bc.cache.GetFinishedBlock()
	}

	if startBlockNumber == 0 {
		startBlockNumber = newestBlockNumber
	}

	return startBlockNumber
}

func (bc *blockCrawler) setHeaderHeight(headerHeight uint64) {
	bc.headerHeightLock.Lock()
	defer bc.headerHeightLock.Unlock()
	if headerHeight > bc.headerHeight {
		bc.headerHeight = headerHeight
	}
}

func (bc *blockCrawler) getHeaderHeight() uint64 {
	bc.headerHeightLock.RLock()
	defer bc.headerHeightLock.RUnlock()
	return bc.headerHeight
}

func (bc *blockCrawler) subscribeNewHead() (ethereum.Subscription, <-chan error, error) {
	sub, err := bc.wsEthClient.SubscribeNewHead(bc.ctx, bc.blockHeaderChan)
	if err != nil {
		return nil, nil, err
	}
	return sub, sub.Err(), nil
}

func (bc *blockCrawler) startSubscribeNewHead() {
	headerHeight, err := bc.ethClient.BlockNumber(bc.ctx)
	if err != nil {
		Log.Fatal("BigIntHeight() err", zap.Error(err))
	}
	bc.setHeaderHeight(headerHeight)

	sub, subErrChan, subErr := bc.subscribeNewHead()
	if subErr != nil {
		Log.Fatal("subscribeNewHead() err", zap.Error(subErr))
	}

	go func() {
		for {
			select {
			case err = <-subErrChan:
				Log.Error("receive block err", zap.Error(err))
				sub.Unsubscribe()
				for {
					sub, subErrChan, subErr = bc.subscribeNewHead()
					if subErr != nil {
						Log.Error("subscribeNewHead() err", zap.Error(subErr))
						time.Sleep(time.Second * 1)
						continue
					}
					Log.Info("subscribeNewHead() success")
					break
				}

			case blockHeader := <-bc.blockHeaderChan:
				Log.Info("receive block header", zap.Any("headerHeight", blockHeader.Number))
				headerHeight = blockHeader.Number.Uint64()
				bc.setHeaderHeight(headerHeight)
			}
		}
	}()
}

func (bc *blockCrawler) dispatchRange(from, to uint64) (stopped bool, nextBlock uint64) {
	for i := from; i <= to; i++ {
		if bc.isStopped() {
			return true, i
		}
		bc.GetBlockAsync(i)
	}
	return false, 0
}

func (bc *blockCrawler) StartDispatch(startBlockNumber uint64) {
	bc.startSubscribeNewHead()

	go func() {
		cur := startBlockNumber
		for {
			headerHeight := bc.getHeaderHeight()
			if headerHeight < cur {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			stopped, nextBlockHeight := bc.dispatchRange(cur, headerHeight)
			if stopped {
				Log.Info("dispatch interrupted", zap.Uint64("nextBlockHeight", nextBlockHeight))
				bc.doStop()
				return
			}

			cur = headerHeight + 1
		}
	}()
}

func (bc *blockCrawler) Stop() {
	bc.stoppedLock.Lock()
	defer bc.stoppedLock.Unlock()
	bc.stopped = true
}

func (bc *blockCrawler) isStopped() bool {
	bc.stoppedLock.RLock()
	defer bc.stoppedLock.RUnlock()
	return bc.stopped
}

func (bc *blockCrawler) doStop() {
	close(bc.queue)
}
