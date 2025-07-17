package main

import (
	"errors"
	"fmt"
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
	db DBWrap
	wg *sync.WaitGroup
}

func (ea *eventReactor) ReactBlockEvent(blockEvent *BlockEvent) error {
	for _, e := range blockEvent.Events {
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
	ea.db.close()
	ea.wg.Done()
}

func NewEventReactor(db DBWrap, wg *sync.WaitGroup) EventReactor {
	return &eventReactor{
		db: db,
		wg: wg,
	}
}

func (ea *eventReactor) reactEvent(event *Event) error {
	ks := event.GetTickStateKeys()

	switch event.Type {
	case EventTypeMint:
		ea.reactTick(ks[0], uint32(event.TickLower.Uint64()), event.Amount)
		ea.reactTick(ks[1], uint32(event.TickUpper.Uint64()), new(big.Int).Neg(event.Amount))

	case EventTypeBurn:
		ea.reactTick(ks[0], uint32(event.TickLower.Uint64()), new(big.Int).Neg(event.Amount))
		ea.reactTick(ks[1], uint32(event.TickUpper.Uint64()), event.Amount)

	default:
		panic(fmt.Sprintf("wrong event: %v", event.Type))
	}

	return nil
}

func IsNotExist(err error) bool {
	return errors.Is(err, ErrKeyNotFound)
}

func (ea *eventReactor) getOrNewTickState(k []byte, tick uint32) *TickState {
	tickState, err := ea.db.GetTickState(k)
	if err != nil {
		if IsNotExist(err) {
			return NewTickState(tick)
		} else {
			panic(fmt.Sprintf("GetTickState err: k=%s, err=%v", k, err)) // TODO
		}
	}

	return tickState
}

func (ea *eventReactor) saveTickState(k []byte, tick *TickState) error {
	return ea.db.SaveTickState(k, tick)
}

func (ea *eventReactor) reactTick(k []byte, tick uint32, amount *big.Int) error {
	tickState := ea.getOrNewTickState(k, tick)
	tickState.AddLiquidity(amount)
	return ea.saveTickState(k, tickState)
}
