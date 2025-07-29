package main

import (
	"math"
	"math/big"
)

type TickAmount struct {
	TickLower int32
	TickUpper int32
	Liquidity *big.Int
	Amount0   *big.Float
	Amount1   *big.Float
}

func CalcAmount(tickStates []*TickState, tickSpacing int32, tickLower, tickUpper int32, token0Decimals, token1Decimals int) ([]TickAmount, []TickAmount) {
	var allDetails []TickAmount
	if len(tickStates) == 0 {
		return allDetails, nil
	}

	// 构建所有tick边界
	tickBoundaries := make([]int32, len(tickStates))
	for i, t := range tickStates {
		tickBoundaries[i] = t.Tick
	}

	// 计算每个tick区间的liquidity前缀和
	prefixLiquidity := make([]*big.Int, len(tickStates))
	currentLiquidity := big.NewInt(0)
	for i, t := range tickStates {
		currentLiquidity = new(big.Int).Add(currentLiquidity, t.LiquidityNet)
		prefixLiquidity[i] = new(big.Int).Set(currentLiquidity)
	}

	var summary []TickAmount
	for i := 0; i < len(tickBoundaries)-1; i++ {
		segLower := tickBoundaries[i]
		segUpper := tickBoundaries[i+1]
		liquidity := prefixLiquidity[i]
		amount0Sum, amount1Sum, details := CalcAmountInRange(
			segLower, segUpper, liquidity, tickSpacing, token0Decimals, token1Decimals,
		)
		allDetails = append(allDetails, details...)
		summary = append(summary, TickAmount{
			TickLower: segLower,
			TickUpper: segUpper,
			Liquidity: new(big.Int).Set(liquidity),
			Amount0:   amount0Sum,
			Amount1:   amount1Sum,
		})
	}

	// 只保留视图区间内的明细和summary
	filteredDetails := []TickAmount{}
	for _, d := range allDetails {
		if d.TickLower >= tickLower && d.TickUpper <= tickUpper {
			filteredDetails = append(filteredDetails, d)
		}
	}
	filteredSummary := []TickAmount{}
	for _, s := range summary {
		if s.TickLower >= tickLower && s.TickUpper <= tickUpper {
			filteredSummary = append(filteredSummary, s)
		}
	}
	return filteredDetails, filteredSummary
}

// CalcAmountInRange 计算某个[tickLower, tickUpper)区间内的amount0/amount1总和和tickspace明细
func CalcAmountInRange(
	tickLower, tickUpper int32,
	liquidity *big.Int,
	tickSpacing int32, token0Decimals, token1Decimals int,
) (amount0Sum, amount1Sum *big.Float, details []TickAmount) {
	Q96 := new(big.Float).SetInt(new(big.Int).Lsh(big.NewInt(1), 96))
	pow10Token0 := new(big.Float).SetFloat64(math.Pow10(token0Decimals))
	pow10Token1 := new(big.Float).SetFloat64(math.Pow10(token1Decimals))

	amount0Sum = new(big.Float)
	amount1Sum = new(big.Float)
	details = []TickAmount{}

	liqF := new(big.Float).SetInt(liquidity)
	for t := tickLower; t < tickUpper; t += tickSpacing {
		tickA := t
		tickB := t + tickSpacing

		sqrtA := new(big.Float).Mul(
			new(big.Float).SetFloat64(math.Pow(1.0001, float64(tickA)/2)), Q96)
		sqrtB := new(big.Float).Mul(
			new(big.Float).SetFloat64(math.Pow(1.0001, float64(tickB)/2)), Q96)

		amount0 := new(big.Float).Mul(liqF, Q96)
		amount0.Mul(amount0, new(big.Float).Sub(sqrtB, sqrtA))
		amount0.Quo(amount0, sqrtB)
		amount0.Quo(amount0, sqrtA)
		amount0.Quo(amount0, pow10Token0)

		amount1 := new(big.Float).Mul(liqF, new(big.Float).Sub(sqrtB, sqrtA))
		amount1.Quo(amount1, Q96)
		amount1.Quo(amount1, pow10Token1)

		amount0Sum.Add(amount0Sum, amount0)
		amount1Sum.Add(amount1Sum, amount1)

		details = append(details, TickAmount{
			TickLower: tickA,
			TickUpper: tickB,
			Liquidity: new(big.Int).Set(liquidity),
			Amount0:   amount0,
			Amount1:   amount1,
		})
	}
	return amount0Sum, amount1Sum, details
}
