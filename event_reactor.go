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

/*
genKey generates a key for the tick based on the event's address and tick values.
tickLower and tickUpper in Uniswap V3 contract are represented as int24 values.
tickLower and tickUpper are converted to int32 and appended to the address bytes.
*/
func genKey(event *Event) [2][]byte {
	tickLower := int32(event.TickLower.Int64())
	tickUpper := int32(event.TickUpper.Int64())
	return [2][]byte{
		append(event.Address.Bytes(), int32ToBigEndianBytes(tickLower)...),
		append(event.Address.Bytes(), int32ToBigEndianBytes(tickUpper)...),
	}
}

func (ea *eventReactor) reactEvent(event *Event) error {
	ks := genKey(event)

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

func (ea *eventReactor) getOrNewTick(k []byte) *Tick {
	tick, err := ea.db.GetTick(k)
	if err != nil {
		if IsNotExist(err) {
			return NewTick()
		} else {
			panic(fmt.Sprintf("GetTick err: k=%s, err=%v", k, err)) // TODO
		}
	}

	return tick
}

func (ea *eventReactor) saveTick(k []byte, tick *Tick) error {
	return ea.db.SaveTick(k, tick)
}

func (ea *eventReactor) reactTick(k []byte, amount *big.Int) error {
	tick := ea.getOrNewTick(k)
	tick.AddLiquidity(amount)
	return ea.saveTick(k, tick)
}
