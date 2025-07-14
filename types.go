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
