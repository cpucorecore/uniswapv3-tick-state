package main

import (
	"encoding/binary"
	"github.com/ethereum/go-ethereum/common"
)

const (
	TickStateKeyLen = 26
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
	copy(key[:2], KeyPrefixTickState)
	copy(key[2:22], addr[:])
	copy(key[22:], int32ToOrderedBytes(tick))
	return key
}

func BytesToTickStateKey(bytes []byte) TickStateKey {
	if len(bytes) != TickStateKeyLen {
		panic("unexpected bytes length") // TODO check
	}

	var key TickStateKey
	key.setBytes(bytes)
	return key
}

func int32ToOrderedBytes(n int32) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(n)+0x80000000)
	return buf
}

func orderedBytesToInt32(bytes []byte) int32 {
	if len(bytes) != 4 {
		return 0
	}
	return int32(binary.BigEndian.Uint32(bytes) - 0x80000000)
}

const (
	MinTick  = int32(-887272) // uniswap v3 core: ./contracts/libraries/TickMath.sol:9: int24 internal constant MIN_TICK = -887272;
	MaxTick  = int32(887272)
	MinInt24 = int32(-8388608)
	MaxInt24 = int32(8388607)
)

var (
	minAddr = common.Address{}
	maxAddr = common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	MinKey  = GetTickStateKey(minAddr, MinInt24)
	MaxKey  = GetTickStateKey(maxAddr, MaxInt24)
)
