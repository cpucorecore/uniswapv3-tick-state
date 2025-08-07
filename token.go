package main

import "github.com/ethereum/go-ethereum/common"

const (
	WBNBAddrStr = "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"
	WETHAddrStr = "0x2170Ed0880ac9A755fd29B2688956BD959F933F8"
	BTCBAddrStr = "0x7130d2A12B9BCbFAe4f2634d864A1Ee1Ce3Ead9c"
	USDTAddrStr = "0x55d398326f99059fF775485246999027B3197955"
	USDCAddrStr = "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"
	USD1AddrStr = "0x8d0D000Ee44948FC98c9B98A4FA4921476f08B0d"
	BUSDAddrStr = "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56"
)

var (
	WBNBAddr = common.HexToAddress(WBNBAddrStr)
	WETHAddr = common.HexToAddress(WETHAddrStr)
	BTCBAddr = common.HexToAddress(BTCBAddrStr)
	USDTAddr = common.HexToAddress(USDTAddrStr)
	USDCAddr = common.HexToAddress(USDCAddrStr)
	USD1Addr = common.HexToAddress(USD1AddrStr)
	BUSDAddr = common.HexToAddress(BUSDAddrStr)
)

func IsUSD(addr common.Address) bool {
	return IsSameAddress(addr, USDTAddr) ||
		IsSameAddress(addr, USDCAddr) ||
		IsSameAddress(addr, USD1Addr) ||
		IsSameAddress(addr, BUSDAddr)
}

func IsMonitorToken(addr common.Address) bool {
	return IsUSD(addr) // 明确监控的基准代币就是稳定币
}

func IsMonitorPool(token0, token1 common.Address) bool {
	return IsMonitorToken(token0) || IsMonitorToken(token1)
}
