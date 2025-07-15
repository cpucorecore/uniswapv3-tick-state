package main

import (
	"context"
	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"time"
)

type TaskCommiter interface {
	CommitTask(uint64)
	NoMoreTask()
}

type TaskDispatcher interface {
	MountTaskCommiter(TaskCommiter)
	Start(ctx context.Context, fromHeight uint64)
	Stop()
}

type taskDispatcher struct {
	ethClient    *ethclient.Client
	taskCommiter TaskCommiter
	headerHeight MutexValue[uint64]
	ethHeaders   chan *ethtypes.Header
	stopped      MutexValue[bool]
}

func (d *taskDispatcher) MountTaskCommiter(taskCommiter TaskCommiter) {
	d.taskCommiter = taskCommiter
}

func (d *taskDispatcher) Stop() {
	d.stopped.Set(true)
}

func (d *taskDispatcher) subEthHeader(ctx context.Context) (ethereum.Subscription, <-chan error, error) {
	sub, err := d.ethClient.SubscribeNewHead(ctx, d.ethHeaders)
	if err != nil {
		return nil, nil, err
	}
	return sub, sub.Err(), nil
}

func (d *taskDispatcher) startSubEthHeader(ctx context.Context) {
	height, err := d.ethClient.BlockNumber(ctx)
	if err != nil {
		panic(err)
	}

	d.headerHeight.Set(height)

	sub, subErrChan, err := d.subEthHeader(ctx)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case err = <-subErrChan:
				Log.Error("receive block err", zap.Error(err))
				sub.Unsubscribe()
				for {
					sub, subErrChan, err = d.subEthHeader(ctx)
					if err != nil {
						Log.Error("subscribeNewHead() err", zap.Error(err))
						time.Sleep(time.Second * 1)
						continue
					}
					Log.Info("subscribeNewHead() success")
					break
				}

			case ethHeader := <-d.ethHeaders:
				height = ethHeader.Number.Uint64()
				d.headerHeight.Set(height)
			}
		}
	}()
}

func (d *taskDispatcher) dispatchRange(from, to uint64) (stopped bool, nextBlock uint64) {
	for i := from; i <= to; i++ {
		if d.stopped.Get() {
			return true, i
		}
		d.taskCommiter.CommitTask(i)
	}
	return false, 0
}

func (d *taskDispatcher) Start(ctx context.Context, fromHeight uint64) {
	d.startSubEthHeader(ctx)

	go func() {
		height := fromHeight
		for {
			headerHeight := d.headerHeight.Get()
			if headerHeight < height {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			stopped, nextBlockHeight := d.dispatchRange(height, headerHeight)
			if stopped {
				Log.Info("dispatch interrupted", zap.Uint64("nextBlockHeight", nextBlockHeight))
				d.taskCommiter.NoMoreTask()
				return
			}

			height = headerHeight + 1
		}
	}()
}

func NewTaskDispatcher(url string) TaskDispatcher {
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		panic(err)
	}

	return &taskDispatcher{
		ethClient:  ethClient,
		ethHeaders: make(chan *ethtypes.Header, 100),
	}
}
