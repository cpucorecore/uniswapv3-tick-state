package main

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
	"sync"
)

/*
EventReactor
BlockEvent enter this reactor, for every event, it will act as:

	if Event.Type == EventTypeMint {
		tick[TickLower] += Amount
		tick[TickUpper] -= Amount
	} else if Event.Type == EventTypeBurn {
		tick[TickLower] -= Amount
		tick[TickUpper] += Amount
	}
*/
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
			return nil
		}

		if pair.Filtered {
			return nil
		}

		if pair.ProtocolId != 3 {
			return nil
		}

		ok, _ = ea.db.TickExists(e.Address)
		if !ok {
			ticks, err := ea.cc.CallGetAllTicks(e.Address)
			if err != nil {
				// TODO check
				continue
			}
			ea.db.SetCurrentTick(e.Address, int32(ticks.State.Tick.Int64()))
			ea.db.SetTickSpacing(e.Address, int32(ticks.State.Tick.Int64()))
			for _, tick := range ticks.Ticks {
				ea.db.SetTickState(e.Address, tick.Tick, tick)
			}
			ea.db.SetPoolHeight(e.Address, ticks.State.Height.Uint64())
		}

		h, _ := ea.db.GetPoolHeight(e.Address)
		if h >= blockEvent.Height {
			continue
		}

		if err := ea.reactEvent(e); err != nil {
			return err
		}
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

func NewEventReactor(wg *sync.WaitGroup, db Repo, cache Cache, cc *ContractCaller) EventReactor {
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

	case EventTypeBurn:
		ea.reactTick(event.Address, int32(event.TickLower.Int64()), new(big.Int).Neg(event.Amount))
		ea.reactTick(event.Address, int32(event.TickUpper.Int64()), event.Amount)

	case EventTypeSwap:
		// TODO

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
	return ea.db.SetTickState(addr, tick, tickState)
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
