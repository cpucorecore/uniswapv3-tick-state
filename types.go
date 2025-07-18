package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
)

type BlockReceipt struct {
	Height   uint64
	Receipts []*types.Receipt
}

func (b *BlockReceipt) Sequence() uint64 {
	return b.Height
}

const (
	EventTypeMint = iota + 1
	EventTypeBurn
)

type Event struct {
	Address   common.Address
	Type      int
	TickLower *big.Int
	TickUpper *big.Int
	Amount    *big.Int
}

type BlockEvent struct {
	Height uint64
	Events []*Event
}

func (b *BlockEvent) Sequence() uint64 {
	return b.Height
}

type TickState struct {
	Tick         int32
	LiquidityNet *big.Int
}

func NewTickState(tick int32) *TickState {
	return &TickState{
		Tick:         tick,
		LiquidityNet: new(big.Int),
	}
}

func (t *TickState) AddLiquidity(amount *big.Int) {
	t.LiquidityNet.Add(t.LiquidityNet, amount)
}

func (t *TickState) Equal(other *TickState) bool {
	if other == nil {
		return false
	}
	return t.LiquidityNet.Cmp(other.LiquidityNet) == 0
}

func (t *TickState) MarshalBinary() ([]byte, error) {
	return t.LiquidityNet.GobEncode()
}

func (t *TickState) UnmarshalBinary(data []byte) error {
	return t.LiquidityNet.GobDecode(data)
}
