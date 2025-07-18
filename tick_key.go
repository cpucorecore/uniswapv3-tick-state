package main

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
)

const (
	TickStateKeyLen = 26
)

var (
	TickStateKeyPrefix = []byte("2:")
)

type TickStateKey [TickStateKeyLen]byte

func (k *TickStateKey) setBytes(b []byte) {
	copy(k[:], b)
}

func (k TickStateKey) GetAddress() common.Address {
	return common.BytesToAddress(k[2:22])
}

func (k TickStateKey) GetTick() int32 {
	return orderedBytesToInt32(k[22:26])
}

func (k TickStateKey) GetKey() []byte {
	return k[:]
}

func GetTickStateKey(addr common.Address, tick int32) TickStateKey {
	var key TickStateKey
	copy(key[:2], TickStateKeyPrefix)
	copy(key[2:22], addr[:])
	copy(key[22:], int32ToOrderedBytes(tick))
	return key
}

func BytesToTickStateKey(b []byte) TickStateKey {
	if len(b) != TickStateKeyLen {
		panic("unexpected bytes length") // TODO check
	}

	var key TickStateKey
	key.setBytes(b)
	return key
}

func int32ToOrderedBytes(n int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(n)+0x80000000)
	return buf
}

func orderedBytesToInt32(b []byte) int32 {
	if len(b) != 4 {
		return 0
	}
	return int32(binary.BigEndian.Uint32(b) - 0x80000000)
}

const (
	minTick = int32(-8388608) // int24
	maxTick = int32(8388607)  // int24
)

var (
	minAddr = common.Address{}
	maxAddr = common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	MinKey  = GetTickStateKey(minAddr, minTick)
	MaxKey  = GetTickStateKey(maxAddr, maxTick)
)
