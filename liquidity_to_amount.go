package main

import (
	"math"
	"math/big"
)

type RangeAmount struct {
	TickLower int32
	TickUpper int32
	Liquidity *big.Int
	Amount0   *big.Float
	Amount1   *big.Float
}

func CalcAmount(tickStates []*TickState, tickSpacing int32, tickLower, tickUpper int32, token0Decimals, token1Decimals int) ([]RangeAmount, []RangeAmount) {
	var allDetails []RangeAmount
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

	var summary []RangeAmount
	for i := 0; i < len(tickBoundaries)-1; i++ {
		segLower := tickBoundaries[i]
		segUpper := tickBoundaries[i+1]
		liquidity := prefixLiquidity[i]
		amount0Sum, amount1Sum, details := CalcAmountsForTickRange(
			segLower, segUpper, liquidity, tickSpacing, token0Decimals, token1Decimals,
		)
		allDetails = append(allDetails, details...)
		summary = append(summary, RangeAmount{
			TickLower: segLower,
			TickUpper: segUpper,
			Liquidity: new(big.Int).Set(liquidity),
			Amount0:   amount0Sum,
			Amount1:   amount1Sum,
		})
	}

	// 只保留视图区间内的明细和summary
	filteredDetails := []RangeAmount{}
	for _, d := range allDetails {
		if d.TickLower >= tickLower && d.TickUpper <= tickUpper {
			filteredDetails = append(filteredDetails, d)
		}
	}
	filteredSummary := []RangeAmount{}
	for _, s := range summary {
		if s.TickLower >= tickLower && s.TickUpper <= tickUpper {
			filteredSummary = append(filteredSummary, s)
		}
	}
	return filteredDetails, filteredSummary
}

type RangeLiquidity struct {
	TickLower int32
	TickUpper int32
	Liquidity *big.Int
}

// BuildRangeLiquidityArray 根据tickStates构建RangeLiquidity数组
func BuildRangeLiquidityArray(tickStates []*TickState) []*RangeLiquidity {
	if len(tickStates) == 0 {
		return nil
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

	// 构建RangeLiquidity数组
	var rangeLiquidity []*RangeLiquidity
	for i := 0; i < len(tickBoundaries)-1; i++ {
		rangeLiquidity = append(rangeLiquidity, &RangeLiquidity{
			TickLower: tickBoundaries[i],
			TickUpper: tickBoundaries[i+1],
			Liquidity: new(big.Int).Set(prefixLiquidity[i]),
		})
	}

	return rangeLiquidity
}

func CalcRangeAmountArray(rangeLiquidityArray []*RangeLiquidity, token0Decimals, token1Decimals int) []*RangeAmount {
	if len(rangeLiquidityArray) == 0 {
		return nil
	}

	var rangeAmountArray []*RangeAmount
	for _, rangeLiquidity := range rangeLiquidityArray {
		rangeAmount := CalcRangeAmount(rangeLiquidity.Liquidity, rangeLiquidity.TickLower, rangeLiquidity.TickUpper, token0Decimals, token1Decimals)
		rangeAmountArray = append(rangeAmountArray, rangeAmount)
	}

	return rangeAmountArray
}

func SplitRangeLiquidityArray(rangeLiquidityArray []*RangeLiquidity, tickSpacing int32) []*RangeLiquidity {
	var result []*RangeLiquidity

	for _, rangeLiquidity := range rangeLiquidityArray {
		tickRanges := SplitToTickSpacingRanges(rangeLiquidity.TickLower, rangeLiquidity.TickUpper, tickSpacing)

		for _, tickRange := range tickRanges {
			result = append(result, &RangeLiquidity{
				TickLower: tickRange[0],
				TickUpper: tickRange[1],
				Liquidity: new(big.Int).Set(rangeLiquidity.Liquidity),
			})
		}
	}

	return result
}

// FilterRangeLiquidityArray 根据tickFrom和tickTo筛选RangeLiquidity数组
// 只保留与指定区间有重叠的RangeLiquidity
func FilterRangeLiquidityArray(rangeLiquidityArray []*RangeLiquidity, fromTick, toTick int32) []*RangeLiquidity {
	if len(rangeLiquidityArray) == 0 {
		return nil
	}

	var filteredArray []*RangeLiquidity
	for _, rl := range rangeLiquidityArray {
		// 只添加与指定区间有重叠的RangeLiquidity
		if rl.TickUpper > fromTick && rl.TickLower < toTick {
			filteredArray = append(filteredArray, rl)
		}
	}

	return filteredArray
}

// CalcAmountsForTickRange 计算某个[tickLower, tickUpper)区间内的amount0/amount1总和和tickspace明细
func CalcAmountsForTickRange(
	tickLower, tickUpper int32,
	liquidity *big.Int,
	tickSpacing int32, token0Decimals, token1Decimals int,
) (amount0Sum, amount1Sum *big.Float, details []RangeAmount) {
	details = []RangeAmount{}
	amount0Sum = new(big.Float)
	amount1Sum = new(big.Float)
	for t := tickLower; t < tickUpper; t += tickSpacing {
		tickA := t
		tickB := t + tickSpacing
		tickAmt := CalcRangeAmount(liquidity, tickA, tickB, token0Decimals, token1Decimals)
		amount0Sum.Add(amount0Sum, tickAmt.Amount0)
		amount1Sum.Add(amount1Sum, tickAmt.Amount1)
		details = append(details, *tickAmt)
	}
	return amount0Sum, amount1Sum, details
}

// SplitToTickSpacingRanges 将大区间[tickLower, tickUpper)按tickSpacing拆分为若干小区间
func SplitToTickSpacingRanges(tickLower, tickUpper, tickSpacing int32) [][2]int32 {
	var ranges [][2]int32
	for t := tickLower; t < tickUpper; t += tickSpacing {
		next := t + tickSpacing
		if next > tickUpper {
			next = tickUpper
		}
		ranges = append(ranges, [2]int32{t, next})
	}
	return ranges
}

// CalcAmountsByTickSpacing 按tickSpacing分段计算所有小区间的token0、token1
func CalcAmountsByTickSpacing(liquidity *big.Int, tickLower, tickUpper, tickSpacing int32, token0Decimals, token1Decimals int) []RangeAmount {
	ranges := SplitToTickSpacingRanges(tickLower, tickUpper, tickSpacing)
	results := make([]RangeAmount, 0, len(ranges))
	for _, r := range ranges {
		tickAmt := CalcRangeAmount(liquidity, r[0], r[1], token0Decimals, token1Decimals)
		results = append(results, *tickAmt)
	}
	return results
}

// CalcRangeAmount
// 直接计算整个tick区间的token0、token1总量，不分段
//
// 公式来源：Uniswap V3 Whitepaper
//
// amount0 = liquidity * (sqrtB - sqrtA) / (sqrtB * sqrtA)
// amount1 = liquidity * (sqrtB - sqrtA)
// 其中：
//
//	sqrtA = sqrt(PA) * Q96
//	sqrtB = sqrt(PB) * Q96
//	PA = 1.0001^tickLower
//	PB = 1.0001^tickUpper
//	Q96 = 2^96
//	liquidity 单位同Uniswap合约
//	amount0, amount1 需除以对应token的10^decimals
func CalcRangeAmount(liquidity *big.Int, tickLower, tickUpper int32, token0Decimals, token1Decimals int) *RangeAmount {
	Q96 := new(big.Float).SetInt(new(big.Int).Lsh(big.NewInt(1), 96))
	pow10Token0 := new(big.Float).SetFloat64(math.Pow10(token0Decimals))
	pow10Token1 := new(big.Float).SetFloat64(math.Pow10(token1Decimals))

	liqF := new(big.Float).SetInt(liquidity)

	sqrtA := new(big.Float).Mul(
		new(big.Float).SetFloat64(math.Pow(1.0001, float64(tickLower)/2)), Q96)
	sqrtB := new(big.Float).Mul(
		new(big.Float).SetFloat64(math.Pow(1.0001, float64(tickUpper)/2)), Q96)

	amount0 := new(big.Float).Mul(liqF, Q96)
	amount0.Mul(amount0, new(big.Float).Sub(sqrtB, sqrtA))
	amount0.Quo(amount0, sqrtB)
	amount0.Quo(amount0, sqrtA)
	amount0.Quo(amount0, pow10Token0)

	amount1 := new(big.Float).Mul(liqF, new(big.Float).Sub(sqrtB, sqrtA))
	amount1.Quo(amount1, Q96)
	amount1.Quo(amount1, pow10Token1)

	return &RangeAmount{
		TickLower: tickLower,
		TickUpper: tickUpper,
		Liquidity: new(big.Int).Set(liquidity),
		Amount0:   amount0,
		Amount1:   amount1,
	}
}

// CalculateTickRange 根据tickOffset、tickSpacing和当前tick计算fromTick和toTick
// tickOffset表示当前tick前后多少个tickSpacing区间
func CalculateTickRange(currentTick, tickOffset, tickSpacing int32) (fromTick, toTick int32) {
	centerTick := (currentTick / tickSpacing) * tickSpacing
	fromTick = centerTick - tickOffset*tickSpacing
	toTick = centerTick + (tickOffset+1)*tickSpacing
	return fromTick, toTick
}
