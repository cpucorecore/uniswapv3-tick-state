package abi_instance

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"strings"
)

const (
	LensABIJson    = `[{"inputs":[{"internalType":"contract IPool","name":"pool","type":"address"}],"name":"getAllTicks","outputs":[{"components":[{"internalType":"uint256","name":"height","type":"uint256"},{"internalType":"int24","name":"tickSpacing","type":"int24"},{"internalType":"int24","name":"tick","type":"int24"},{"internalType":"uint128","name":"liquidity","type":"uint128"},{"internalType":"uint160","name":"sqrtPriceX96","type":"uint160"}],"internalType":"struct PoolState","name":"poolState","type":"tuple"},{"components":[{"internalType":"int24","name":"index","type":"int24"},{"internalType":"uint128","name":"liquidityGross","type":"uint128"},{"internalType":"int128","name":"liquidityNet","type":"int128"}],"internalType":"struct Tick[]","name":"ticks","type":"tuple[]"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"contract IPool","name":"pool","type":"address"}],"name":"getPoolState","outputs":[{"components":[{"internalType":"uint256","name":"height","type":"uint256"},{"internalType":"int24","name":"tickSpacing","type":"int24"},{"internalType":"int24","name":"tick","type":"int24"},{"internalType":"uint128","name":"liquidity","type":"uint128"},{"internalType":"uint160","name":"sqrtPriceX96","type":"uint160"}],"internalType":"struct PoolState","name":"poolState","type":"tuple"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"contract IPool","name":"pool","type":"address"},{"internalType":"int24","name":"tickStart","type":"int24"},{"internalType":"uint256","name":"numTicks","type":"uint256"}],"name":"getTicks","outputs":[{"components":[{"internalType":"int24","name":"index","type":"int24"},{"internalType":"uint128","name":"liquidityGross","type":"uint128"},{"internalType":"int128","name":"liquidityNet","type":"int128"}],"internalType":"struct Tick[]","name":"ticks","type":"tuple[]"}],"stateMutability":"view","type":"function"}]`
	LensAddressHex = "0x2511107146BB1908434E92FF7D985C4B7e2Fb08a" // TODO: Replace with the actual Lens contract address
)

var (
	LensABI     *abi.ABI
	LensAddress = common.HexToAddress(LensAddressHex)
)

func init() {
	lensAbi, err := abi.JSON(strings.NewReader(LensABIJson))
	if err != nil {
		panic(err)
	}
	LensABI = &lensAbi
}
