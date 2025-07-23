package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetAllTicks(t *testing.T) {
	cc := NewContractCaller("https://bsc-testnet-dataseed.bnbchain.org")
	poolTicks, err := cc.CallGetAllTicks(common.HexToAddress("0x553700BD9eE66289f658Daf130bdf418EBB93324"))
	require.Nil(t, err, err)
	t.Log(poolTicks)
}
