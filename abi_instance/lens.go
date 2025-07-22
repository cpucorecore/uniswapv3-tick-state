package abi_instance

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"strings"
)

const (
	LensABIJson    = `[{"inputs":[{"internalType":"contract IPool","name":"pool","type":"address"}],"name":"getAllTicks","outputs":[{"internalType":"uint256","name":"height","type":"uint256"},{"internalType":"int24","name":"tickSpacing","type":"int24"},{"components":[{"internalType":"int24","name":"index","type":"int24"},{"internalType":"uint128","name":"liquidityGross","type":"uint128"},{"internalType":"int128","name":"liquidityNet","type":"int128"}],"internalType":"struct Tick[]","name":"ticks","type":"tuple[]"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"contract IPool","name":"pool","type":"address"},{"internalType":"int24","name":"tickStart","type":"int24"},{"internalType":"uint256","name":"numTicks","type":"uint256"}],"name":"getTicks","outputs":[{"internalType":"uint256","name":"height","type":"uint256"},{"internalType":"int24","name":"tickSpacing","type":"int24"},{"components":[{"internalType":"int24","name":"index","type":"int24"},{"internalType":"uint128","name":"liquidityGross","type":"uint128"},{"internalType":"int128","name":"liquidityNet","type":"int128"}],"internalType":"struct Tick[]","name":"ticks","type":"tuple[]"}],"stateMutability":"view","type":"function"}]`
	LensAddressHex = "" // TODO
)

var (
	LensABI *abi.ABI
)

func init() {
	lensAbi, err := abi.JSON(strings.NewReader(LensABIJson))
	if err != nil {
		panic(err)
	}
	LensABI = &lensAbi
}
