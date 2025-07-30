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
	for _, e := range blockEvent.Events {
		exist, err := r.db.PoolExists(e.Address)
		if err != nil {
			Log.Fatal("PoolExists error", zap.String("addr", e.Address.String()), zap.Error(err)) // TODO check Fatal?
		}

		if !exist {
			_, err = r.poolStateGetter.GetPoolState(e.Address)
			if err != nil {
				Log.Info("GetPoolState error", zap.String("addr", e.Address.String()), zap.Error(err))
				if IsIgnorantError(err) {
					continue
				}
				return err
			}
		}

		height, err := r.db.GetPoolHeight(e.Address)
		if err != nil {
			Log.Fatal("GetPoolHeight error", zap.String("addr", e.Address.String()), zap.Error(err)) // TODO check Fatal?
		}

		if height >= blockEvent.Height {
			continue
		}

		if err := r.reactEvent(e); err != nil {
			return err
		}

		r.db.SetPoolHeight(e.Address, blockEvent.Height) // TODO once per block
	}
	return r.db.SetHeight(blockEvent.Height)
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
		Log.Info("Mint Event", zap.String("addr", event.Address.String()))

	case EventTypeBurn:
		r.reactTick(event.Address, int32(event.TickLower.Int64()), new(big.Int).Neg(event.Amount))
		r.reactTick(event.Address, int32(event.TickUpper.Int64()), event.Amount)
		Log.Info("Burn Event", zap.String("addr", event.Address.String()))

	case EventTypeSwap:
		r.db.SetCurrentTick(event.Address, int32(event.Tick.Int64()))
		Log.Info("Swap Event", zap.String("addr", event.Address.String()))

	default:
		panic(fmt.Sprintf("wrong event: %v", event.Type))
	}

	return nil
}

func IsNotExist(err error) bool {
	return errors.Is(err, ErrKeyNotFound)
}

func (r *eventReactor) reactTick(addr common.Address, tick int32, amount *big.Int) error {
	tickState := r.getOrNewTickState(addr, tick)
	tickState.AddLiquidity(amount)
	return r.db.SetTickState(addr, tickState)
}

func (r *eventReactor) getOrNewTickState(addr common.Address, tick int32) *TickState {
	tickState, err := r.db.GetTickState(addr, tick)
	if err != nil {
		if IsNotExist(err) {
			return NewTickState(tick)
		} else {
			panic(fmt.Sprintf("GetTickState err: addr=%v,tick=%d, err=%v", addr.String(), tick, err))
		}
	}

	return tickState
}
