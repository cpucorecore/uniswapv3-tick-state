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
	wg              *sync.WaitGroup
	db              DB
	poolStateGetter PoolStateGetter
}

func IsIgnorantError(err error) bool {
	if errors.Is(err, ErrPairNotFound) ||
		errors.Is(err, ErrPairFiltered) ||
		errors.Is(err, ErrNotV3Pool) {
		return true
	}
	return false
}

func (r *eventReactor) ReactBlockEvent(blockEvent *BlockEvent) error {
	Log.Debug("ReactBlockEvent begin", zap.Any("height", blockEvent.Height))

	for _, event := range blockEvent.Events {
		height, err := r.db.GetHeight(event.Address)
		if err != nil {
			return err
		}

		if height == 0 {
			poolState, err := r.poolStateGetter.GetPoolState(event.Address)
			if err != nil {
				if IsIgnorantError(err) {
					continue
				}

				return err
			}
			height = poolState.Global.Height.Uint64()
		}

		if height >= blockEvent.Height {
			continue
		}

		if err = r.reactEvent(event); err != nil {
			return err
		}

		r.db.SetHeight(event.Address, blockEvent.Height)
	}

	Log.Info("ReactBlockEvent end", zap.Any("height", blockEvent.Height))
	return r.db.SetFinishHeight(blockEvent.Height)
}

func (r *eventReactor) PutInput(blockEvent *BlockEvent) {
	// no buffer now
	err := r.ReactBlockEvent(blockEvent)
	if err != nil {
		Log.Fatal("ReactBlockEvent error", zap.Error(err), zap.Uint64("height", blockEvent.Height))
	}
}

func (r *eventReactor) FinInput() {
	r.shutdown()
}

func (r *eventReactor) shutdown() {
	r.db.Close()
	r.wg.Done()
}

func NewEventReactor(wg *sync.WaitGroup, db DB, poolStateGetter PoolStateGetter) EventReactor {
	return &eventReactor{
		wg:              wg,
		db:              db,
		poolStateGetter: poolStateGetter,
	}
}

func (r *eventReactor) reactEvent(event *Event) error {
	switch event.Type {
	case EventTypeMint:
		r.reactTick(event.Address, int32(event.TickLower.Int64()), event.Amount)
		r.reactTick(event.Address, int32(event.TickUpper.Int64()), new(big.Int).Neg(event.Amount))
		Log.Debug("Mint Event", zap.String("addr", event.Address.String()))

	case EventTypeBurn:
		r.reactTick(event.Address, int32(event.TickLower.Int64()), new(big.Int).Neg(event.Amount))
		r.reactTick(event.Address, int32(event.TickUpper.Int64()), event.Amount)
		Log.Debug("Burn Event", zap.String("addr", event.Address.String()))

	case EventTypeSwap:
		r.db.SetCurrentTick(event.Address, int32(event.Tick.Int64()))
		Log.Debug("Swap Event", zap.String("addr", event.Address.String()))

	default:
		panic(fmt.Sprintf("wrong event: %v", event.Type))
	}

	return nil
}

func (r *eventReactor) reactTick(addr common.Address, tick int32, amount *big.Int) error {
	tickState := r.getOrNewTickState(addr, tick)
	tickState.AddLiquidity(amount)
	return r.db.SetTickState(addr, tickState)
}

func (r *eventReactor) getOrNewTickState(addr common.Address, tick int32) *TickState {
	tickState, err := r.db.GetTickState(addr, tick)
	if err != nil {
		panic(fmt.Sprintf("GetTickState err: addr=%v,tick=%d, err=%v", addr.String(), tick, err))
	}

	if tickState == nil {
		return NewTickState(tick)
	}

	return tickState
}
