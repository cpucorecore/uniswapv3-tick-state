package main

import (
	"github.com/shopspring/decimal"
	"math/big"
)

type TickAmount struct {
	TickIndex int32
	Liquidity *big.Int
	Amount0   *big.Int
	Amount1   *big.Int
}

// tickToSqrtPrice returns sqrt(1.0001^tick) as decimal.Decimal
func tickToSqrtPrice(tick int32) *big.Int {
	exp := decimal.NewFromFloat(float64(tick) / 2.0)
	base := decimal.NewFromFloat(1.0001)
	return base.Pow(exp).BigInt()
}

func CalcAmount(poolState *PoolState, ticks []*TickState) []TickAmount {
	results := []TickAmount{}
	L := big.NewInt(0)
	for i := 0; i < len(ticks)-1; i++ {
		t := ticks[i]
		nextT := ticks[i+1]
		L.Add(L, t.LiquidityNet)

		sqrtPa := tickToSqrtPrice(t.Tick)
		sqrtPb := tickToSqrtPrice(nextT.Tick)

		invSqrtPa := big.NewInt(0).Div(big.NewInt(1), sqrtPa)
		invSqrtPb := big.NewInt(0).Div(big.NewInt(1), sqrtPb)

		amount0 := big.NewInt(0).Mul(L, big.NewInt(0).Sub(invSqrtPa, invSqrtPb))
		amount1 := big.NewInt(0).Mul(L, big.NewInt(0).Sub(sqrtPb, sqrtPa))

		results = append(results, TickAmount{
			TickIndex: t.Tick,
			Liquidity: L,
			Amount0:   amount0,
			Amount1:   amount1,
		})
	}
	return results
}
