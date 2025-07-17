package main

import (
	"encoding/binary"
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

func int32ToBigEndianBytes(n int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(n))
	return buf
}

func genKey(base []byte, value int32) []byte {
	buf := make([]byte, len(base)+4)
	copy(buf, base)
	binary.BigEndian.PutUint32(buf[len(base):], uint32(value))
	return buf
}

/*
genKeys generates a key for the tick based on the event's address and tick values.
tickLower and tickUpper in Uniswap V3 contract are represented as int24 values.
tickLower and tickUpper are converted to int32 and appended to the address bytes.
*/
func genKeys(event *Event) [2][]byte {
	return [2][]byte{
		genKey(event.Address.Bytes(), int32(event.TickLower.Int64())),
		genKey(event.Address.Bytes(), int32(event.TickUpper.Int64())),
	}
}

func (ea *eventReactor) reactEvent(event *Event) error {
	ks := genKeys(event)

	switch event.Type {
	case EventTypeMint:
		ea.reactTick(ks[0], event.Amount)
		ea.reactTick(ks[1], new(big.Int).Neg(event.Amount))

	case EventTypeBurn:
		ea.reactTick(ks[0], new(big.Int).Neg(event.Amount))
		ea.reactTick(ks[1], event.Amount)

	default:
		panic(fmt.Sprintf("wrong event: %v", event.Type))
	}

	return nil
}

func IsNotExist(err error) bool {
	return errors.Is(err, ErrKeyNotFound)
}

func (ea *eventReactor) getOrNewTickState(k []byte) *TickState {
	tick, err := ea.db.GetTickState(k)
	if err != nil {
		if IsNotExist(err) {
			return NewTick()
		} else {
			panic(fmt.Sprintf("GetTickState err: k=%s, err=%v", k, err)) // TODO
		}
	}

	return tick
}

func (ea *eventReactor) saveTickState(k []byte, tick *TickState) error {
	return ea.db.SaveTickState(k, tick)
}

func (ea *eventReactor) reactTick(k []byte, amount *big.Int) error {
	tickState := ea.getOrNewTickState(k)
	tickState.AddLiquidity(amount)
	return ea.saveTickState(k, tickState)
}
