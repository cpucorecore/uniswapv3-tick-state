package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetAllTicks(t *testing.T) {
	t.Skip()
	cc := NewContractCaller("https://bsc-testnet-dataseed.bnbchain.org")
	poolState, err := cc.GetPoolState(common.HexToAddress("0x172fcD41E0913e95784454622d1c3724f546f849"))
	require.Nil(t, err, err)
	t.Log(poolState)
}
