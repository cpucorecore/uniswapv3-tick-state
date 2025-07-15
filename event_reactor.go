package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
)

/*
EventActor
BlockEvent enter this reactor, for every event, it will act as:

	if Event.Type == EventTypeMint {
		tick[TickLower] += Amount
		tick[TickUpper] -= Amount
	} else if Event.Type == EventTypeBurn {
		tick[TickLower] -= Amount
		tick[TickUpper] += Amount
	}
*/
type EventActor interface {
	ActBlockEvent(event *BlockEvent) error
}

type eventActor struct {
	db DBWrap
}

func NewEventActor(db DBWrap) EventActor {
	return &eventActor{
		db: db,
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

func (ea *eventActor) actEvent(event *Event) error {
	ks := genKey(event)

	switch event.Type {
	case EventTypeMint:
		ea.actTick(ks[0], event.Amount)
		ea.actTick(ks[1], new(big.Int).Neg(event.Amount))

	case EventTypeBurn:
		ea.actTick(ks[0], new(big.Int).Neg(event.Amount))
		ea.actTick(ks[1], event.Amount)

	default:
		panic(fmt.Sprintf("wrong event: %v", event.Type))
	}

	return nil
}

func IsNotExist(err error) bool {
	return errors.Is(err, ErrKeyNotFound)
}

func (ea *eventActor) getOrNewTick(k []byte) *Tick {
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

func (ea *eventActor) saveTick(k []byte, tick *Tick) error {
	return ea.db.SaveTick(k, tick)
}

func (ea *eventActor) actTick(k []byte, amount *big.Int) error {
	tick := ea.getOrNewTick(k)
	tick.AddLiquidity(amount)
	return ea.saveTick(k, tick)
}

func (ea *eventActor) ActBlockEvent(event *BlockEvent) error {
	for _, e := range event.Events {
		if err := ea.actEvent(e); err != nil {
			return err
		}
	}
	return nil
}
