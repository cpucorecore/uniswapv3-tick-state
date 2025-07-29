package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/avast/retry-go/v4"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"strings"
	"time"
	"uniswapv3-tick-state/abi_instance"
)

type ContractCaller struct {
	ethClient *ethclient.Client
}

func NewContractCaller(url string) *ContractCaller {
	ethClient, err := ethclient.Dial(url)
	if err != nil {
		panic("failed to connect to Ethereum client: " + err.Error())
	}

	return &ContractCaller{
		ethClient: ethClient,
	}
}

func IsRetryableErr(err error) bool {
	errMsg := err.Error()
	if strings.Contains(errMsg, "execution reverted") ||
		strings.Contains(errMsg, "out of gas") ||
		strings.Contains(errMsg, "abi: cannot marshal in to go slice") {
		return false
	}
	return true
}

func (c *ContractCaller) callContract(ctx context.Context, req *CallContractReq) ([]byte, error) {
	bytes, err := c.ethClient.CallContract(
		ctx,
		ethereum.CallMsg{
			To:   &req.Address,
			Data: req.Data,
		},
		req.BlockNumber,
	)

	if err != nil {
		if IsRetryableErr(err) {
			return nil, err
		}
		return nil, nil
	}

	return bytes, nil
}

const (
	timeoutDuration = time.Minute * 5
)

func (c *ContractCaller) CallContract(ctx context.Context, req *CallContractReq) ([]byte, error) {
	ctxWithTimeout, _ := context.WithTimeout(ctx, timeoutDuration)
	return retry.DoWithData(func() ([]byte, error) {
		return c.callContract(ctx, req)
	}, infiniteAttempts, retryDelay, retry.Context(ctxWithTimeout))
}

var (
	getAllTicksMethod = abi_instance.LensABI.Methods["getAllTicks"]
)

var (
	ErrEmptyOutput = errors.New("empty output")
)

func (c *ContractCaller) GetPoolState(poolAddr common.Address) (*PoolState, error) {
	data, err := abi_instance.LensABI.Pack("getAllTicks", poolAddr)
	if err != nil {
		return nil, err
	}

	req := &CallContractReq{
		Address: abi_instance.LensAddress,
		Data:    data,
	}

	Log.Info(fmt.Sprintf("Calling getAllTicks: %s", req))
	bytes, err := c.CallContract(context.Background(), req)
	if err != nil {
		return nil, err
	}

	if len(bytes) == 0 {
		return nil, ErrEmptyOutput
	}

	outputs, err := getAllTicksMethod.Outputs.Unpack(bytes)
	if err != nil {
		return nil, err
	}

	poolState := outputs[0].(struct {
		Height       *big.Int `json:"height"`
		TickSpacing  *big.Int `json:"tickSpacing"`
		Tick         *big.Int `json:"tick"`
		Liquidity    *big.Int `json:"liquidity"`
		SqrtPriceX96 *big.Int `json:"sqrtPriceX96"`
	})

	ticks := outputs[1].([]struct {
		Index          *big.Int `json:"index"`
		LiquidityGross *big.Int `json:"liquidityGross"`
		LiquidityNet   *big.Int `json:"liquidityNet"`
	})

	var tickStates []*TickState
	for _, tick := range ticks {
		tickStates = append(tickStates, &TickState{
			Tick:         int32(tick.Index.Int64()),
			LiquidityNet: tick.LiquidityNet,
		})
	}

	return &PoolState{
		GlobalState: &PoolGlobalState{
			Height:       poolState.Height,
			TickSpacing:  poolState.TickSpacing,
			Tick:         poolState.Tick,
			Liquidity:    poolState.Liquidity,
			SqrtPriceX96: poolState.SqrtPriceX96,
		},
		TickStates: tickStates,
	}, nil
}
