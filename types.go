package main

import (
	"encoding/binary"
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

func int32ToOrderedBytes(n int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(n)+0x80000000)
	return buf
}

const (
	TickStateKeySize = 26 // 2 bytes prefix("2:") + 20 bytes address + 4 bytes tick
)

var (
	TickStateKeyPrefix = []byte("2:")
)

func GetTickStateKey(address common.Address, tick int32) []byte {
	key := make([]byte, TickStateKeySize)
	copy(key[:2], TickStateKeyPrefix)
	copy(key[2:22], address[:])
	copy(key[22:], int32ToOrderedBytes(tick))
	return key
}

/*
GetTickStateKeys generates a key for the tick based on the event's address and tick values.
tickLower and tickUpper in Uniswap V3 contract are represented as int24 values.
tickLower and tickUpper are converted to int32 and appended to the address bytes.
*/
func (e *Event) GetTickStateKeys() [2][]byte {
	return [2][]byte{
		GetTickStateKey(e.Address, int32(e.TickLower.Int64())),
		GetTickStateKey(e.Address, int32(e.TickUpper.Int64())),
	}
}

type BlockEvent struct {
	Height uint64
	Events []*Event
}

func (b *BlockEvent) Sequence() uint64 {
	return b.Height
}

type TickState struct {
	Tick         uint32 // int24 in Uniswap V3, but we use uint32 for simplicity
	LiquidityNet *big.Int
}

func (t *TickState) V() []byte {
	return t.LiquidityNet.Bytes()
}

func NewTickState(tick uint32) *TickState {
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
	lnBytes, err := t.LiquidityNet.GobEncode()
	if err != nil {
		return nil, err
	}

	buf := make([]byte, 4+len(lnBytes))
	binary.BigEndian.PutUint32(buf[:4], uint32(t.Tick))
	copy(buf[4:], lnBytes)
	return buf, nil
}

func (t *TickState) UnmarshalBinary(data []byte) error {
	t.Tick = binary.BigEndian.Uint32(data[:4])
	return t.LiquidityNet.GobDecode(data[4:])
}
