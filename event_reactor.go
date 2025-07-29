package main

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
	"sync"
)

type EventReactor interface {
	ReactBlockEvent(event *BlockEvent) error
	Output[*BlockEvent]
}

type eventReactor struct {
	wg    *sync.WaitGroup
	db    Repo
	cache Cache
	cc    *ContractCaller
}

func (ea *eventReactor) ReactBlockEvent(blockEvent *BlockEvent) error {
	for _, e := range blockEvent.Events {
		pair, ok := ea.cache.GetPair(e.Address)
		if !ok {
			Log.Info("pool not cached", zap.String("addr", e.Address.String()))
			return nil
		}

		if pair.Filtered {
			Log.Info("pool filtered", zap.String("addr", e.Address.String()))
			return nil
		}

		if pair.ProtocolId != 3 {
			Log.Info("pool not v3", zap.String("addr", e.Address.String()))
			return nil
		}

		pts, err := GetPoolStateFromDBOrContractCaller(ea.db, ea.cc, pair.Address)
		if err != nil {
			Log.Info("pool get error", zap.String("addr", e.Address.String()))
			return err
		}

		if pts.GlobalState.Height.Uint64() >= blockEvent.Height {
			continue
		}

		if err := ea.reactEvent(e); err != nil {
			return err
		}

		ea.db.SetPoolHeight(e.Address, blockEvent.Height) // TODO once per block
	}
	return ea.db.SetHeight(blockEvent.Height)
}

func (ea *eventReactor) PutInput(blockEvent *BlockEvent) {
	// no buffer now
	err := ea.ReactBlockEvent(blockEvent)
	if err != nil {
		Log.Fatal("reactBlockEvent error", zap.Error(err), zap.Uint64("height", blockEvent.Height))
	}
}

func (ea *eventReactor) FinInput() {
	ea.shutdown()
}

func (ea *eventReactor) shutdown() {
	ea.db.Close()
	ea.wg.Done()
}

func NewEventReactor(wg *sync.WaitGroup, db Repo, cache Cache, url string) EventReactor {
	cc := NewContractCaller(url)
	return &eventReactor{
		wg:    wg,
		db:    db,
		cache: cache,
		cc:    cc,
	}
}

func (ea *eventReactor) reactEvent(event *Event) error {
	switch event.Type {
	case EventTypeMint:
		ea.reactTick(event.Address, int32(event.TickLower.Int64()), event.Amount)
		ea.reactTick(event.Address, int32(event.TickUpper.Int64()), new(big.Int).Neg(event.Amount))
		Log.Info("Mint Event", zap.String("addr", event.Address.String()))

	case EventTypeBurn:
		ea.reactTick(event.Address, int32(event.TickLower.Int64()), new(big.Int).Neg(event.Amount))
		ea.reactTick(event.Address, int32(event.TickUpper.Int64()), event.Amount)
		Log.Info("Burn Event", zap.String("addr", event.Address.String()))

	case EventTypeSwap:
		ea.db.SetCurrentTick(event.Address, int32(event.Tick.Int64()))
		Log.Info("Swap Event", zap.String("addr", event.Address.String()))

	default:
		panic(fmt.Sprintf("wrong event: %v", event.Type))
	}

	return nil
}

func IsNotExist(err error) bool {
	return errors.Is(err, ErrKeyNotFound)
}

func (ea *eventReactor) reactTick(addr common.Address, tick int32, amount *big.Int) error {
	tickState := ea.getOrNewTickState(addr, tick)
	tickState.AddLiquidity(amount)
	return ea.db.SetTickState(addr, tickState)
}

func (ea *eventReactor) getOrNewTickState(addr common.Address, tick int32) *TickState {
	tickState, err := ea.db.GetTickState(addr, tick)
	if err != nil {
		if IsNotExist(err) {
			return NewTickState(tick)
		} else {
			panic(fmt.Sprintf("GetTickState err: addr=%v,tick=%d, err=%v", addr.String(), tick, err))
		}
	}

	return tickState
}
