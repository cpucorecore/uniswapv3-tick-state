package main

import (
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type ArbitrageAnalysis struct {
	Timestamp        time.Time      `json:"timestamp"`
	Pool1Address     common.Address `json:"pool1_address"`
	Pool2Address     common.Address `json:"pool2_address"`
	Pool1Price       *big.Float     `json:"pool1_price"`
	Pool2Price       *big.Float     `json:"pool2_price"`
	PriceDiff        *big.Float     `json:"price_diff"`
	PriceDiffPercent *big.Float     `json:"price_diff_percent"`

	// 详细分析
	OptimalTradeSize *big.Float `json:"optimal_trade_size"`
	MaxProfit        *big.Float `json:"max_profit"`
	ProfitPercentage *big.Float `json:"profit_percentage"`
	RiskLevel        string     `json:"risk_level"`

	// 流动性分析
	Pool1Liquidity *big.Float `json:"pool1_liquidity"`
	Pool2Liquidity *big.Float `json:"pool2_liquidity"`
	LiquidityRatio *big.Float `json:"liquidity_ratio"`

	// 交易建议
	RecommendedAction string     `json:"recommended_action"`
	TradeDirection    string     `json:"trade_direction"`
	EstimatedGasCost  *big.Float `json:"estimated_gas_cost"`
}

type ArbitrageAnalyzer struct {
	poolStateGetter PoolStateGetter
}

func NewArbitrageAnalyzer(poolStateGetter PoolStateGetter) *ArbitrageAnalyzer {
	return &ArbitrageAnalyzer{
		poolStateGetter: poolStateGetter,
	}
}

func (aa *ArbitrageAnalyzer) AnalyzeArbitrage(pool1Addr, pool2Addr common.Address) (*ArbitrageAnalysis, error) {
	pool1State, err := aa.poolStateGetter.GetPoolState(pool1Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool1 state: %v", err)
	}

	if !pool1State.IsMonitor() {
		return nil, fmt.Errorf("pool1 is not monitor")
	}

	pool2State, err := aa.poolStateGetter.GetPoolState(pool2Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool2 state: %v", err)
	}

	if !pool2State.IsMonitor() {
		return nil, fmt.Errorf("pool2 is not monitor")
	}

	pool1Price := aa.calculatePrice(pool1State)
	pool2Price := aa.calculatePrice(pool2State)

	pool1Liquidity := aa.calculateLiquidity(pool1State)
	pool2Liquidity := aa.calculateLiquidity(pool2State)

	priceDiff := new(big.Float).Sub(pool1Price, pool2Price)
	priceDiffAbs := new(big.Float).Abs(priceDiff)
	var priceDiffPercent *big.Float
	if pool1Price.Cmp(big.NewFloat(0)) == 0 {
		priceDiffPercent = big.NewFloat(0)
	} else {
		priceDiffPercent = new(big.Float).Quo(priceDiffAbs, pool1Price)
		priceDiffPercent.Mul(priceDiffPercent, big.NewFloat(100))
	}

	var tradeDirection string
	var buyPrice, sellPrice *big.Float
	var buyPool, sellPool *PoolState

	if pool1Price.Cmp(pool2Price) > 0 {
		tradeDirection = "pool2_to_pool1"
		buyPrice = pool2Price
		sellPrice = pool1Price
		buyPool = pool2State
		sellPool = pool1State
	} else {
		tradeDirection = "pool1_to_pool2"
		buyPrice = pool1Price
		sellPrice = pool2Price
		buyPool = pool1State
		sellPool = pool2State
	}

	// 计算最优交易规模
	optimalTradeSize := aa.calculateOptimalTradeSize(buyPool, sellPool)

	// 计算最大利润
	maxProfit := aa.calculateMaxProfit(optimalTradeSize, buyPrice, sellPrice)

	// 计算利润率
	var profitPercentage *big.Float
	if optimalTradeSize.Cmp(big.NewFloat(0)) == 0 {
		profitPercentage = big.NewFloat(0)
	} else {
		profitPercentage = new(big.Float).Quo(maxProfit, optimalTradeSize)
		profitPercentage.Mul(profitPercentage, big.NewFloat(100))
	}

	// 评估风险等级
	riskLevel := aa.assessRiskLevel(priceDiffPercent, pool1Liquidity, pool2Liquidity)

	// 计算流动性比率
	var liquidityRatio *big.Float
	if pool2Liquidity.Cmp(big.NewFloat(0)) == 0 {
		liquidityRatio = big.NewFloat(0)
	} else {
		liquidityRatio = new(big.Float).Quo(pool1Liquidity, pool2Liquidity)
	}

	// 生成交易建议
	recommendedAction := aa.generateRecommendation(priceDiffPercent, riskLevel, maxProfit)

	// 估算Gas成本（简化估算）
	estimatedGasCost := big.NewFloat(0.005) // 假设0.005 ETH

	return &ArbitrageAnalysis{
		Timestamp:         time.Now(),
		Pool1Address:      pool1Addr,
		Pool2Address:      pool2Addr,
		Pool1Price:        pool1Price,
		Pool2Price:        pool2Price,
		PriceDiff:         priceDiffAbs,
		PriceDiffPercent:  priceDiffPercent,
		OptimalTradeSize:  optimalTradeSize,
		MaxProfit:         maxProfit,
		ProfitPercentage:  profitPercentage,
		RiskLevel:         riskLevel,
		Pool1Liquidity:    pool1Liquidity,
		Pool2Liquidity:    pool2Liquidity,
		LiquidityRatio:    liquidityRatio,
		RecommendedAction: recommendedAction,
		TradeDirection:    tradeDirection,
		EstimatedGasCost:  estimatedGasCost,
	}, nil
}

func (aa *ArbitrageAnalyzer) calculatePrice(poolState *PoolState) *big.Float {
	if poolState == nil || poolState.Global == nil {
		return big.NewFloat(0)
	}

	currentTick := poolState.Global.Tick.Int64()
	price := new(big.Float).SetFloat64(math.Pow(1.0001, float64(currentTick)))
	return price
}

func (aa *ArbitrageAnalyzer) calculateLiquidity(poolState *PoolState) *big.Float {
	if poolState == nil || poolState.Global == nil || len(poolState.TickStates) == 0 {
		return big.NewFloat(0)
	}
	currentTick := int32(poolState.Global.Tick.Int64())
	tickSpacing := int32(poolState.Global.TickSpacing.Int64())
	tickIndex := (currentTick / tickSpacing) * tickSpacing

	prefixSum := big.NewInt(0)
	for _, tickState := range poolState.TickStates {
		if tickState.Tick > tickIndex {
			break
		}
		prefixSum.Add(prefixSum, tickState.LiquidityNet)
	}
	return new(big.Float).SetInt(prefixSum)
}

func (aa *ArbitrageAnalyzer) calculateOptimalTradeSize(pool1State, pool2State *PoolState) *big.Float {
	// 计算池子1当前tick区间的ETH数量
	currentTick1 := int32(pool1State.Global.Tick.Int64())
	tickSpacing1 := int32(pool1State.Global.TickSpacing.Int64())
	fromTick1, toTick1 := CalculateTickRange(currentTick1, 0, tickSpacing1)
	rangeLiquidityArray1 := BuildRangeLiquidityArray(pool1State.TickStates)
	rangeLiquidityArray1 = FilterRangeLiquidityArray(rangeLiquidityArray1, fromTick1, toTick1)
	rangeAmountArray1 := CalcRangeAmountArray(rangeLiquidityArray1, int(pool1State.Token0.Decimals), int(pool1State.Token1.Decimals))

	// 计算池子2当前tick区间的ETH数量
	currentTick2 := int32(pool2State.Global.Tick.Int64())
	tickSpacing2 := int32(pool2State.Global.TickSpacing.Int64())
	fromTick2, toTick2 := CalculateTickRange(currentTick2, 0, tickSpacing2)
	rangeLiquidityArray2 := BuildRangeLiquidityArray(pool2State.TickStates)
	rangeLiquidityArray2 = FilterRangeLiquidityArray(rangeLiquidityArray2, fromTick2, toTick2)
	rangeAmountArray2 := CalcRangeAmountArray(rangeLiquidityArray2, int(pool2State.Token0.Decimals), int(pool2State.Token1.Decimals))

	// 取两个池子ETH数量的较小值
	var pool1ETH, pool2ETH *big.Float
	if len(rangeAmountArray1) > 0 {
		pool1ETH = rangeAmountArray1[0].Amount1
	} else {
		pool1ETH = big.NewFloat(0)
	}
	if len(rangeAmountArray2) > 0 {
		pool2ETH = rangeAmountArray2[0].Amount1
	} else {
		pool2ETH = big.NewFloat(0)
	}

	optimalSize := pool1ETH
	if pool2ETH.Cmp(pool1ETH) < 0 {
		optimalSize = pool2ETH
	}

	return optimalSize
}

func (aa *ArbitrageAnalyzer) calculateMaxProfit(tradeSize, buyPrice, sellPrice *big.Float) *big.Float {
	// 在低价池买入获得的token数量
	tokensBought := new(big.Float).Quo(tradeSize, buyPrice)

	// 在高价池卖出获得的ETH数量
	ethReceived := new(big.Float).Mul(tokensBought, sellPrice)

	// 利润 = 卖出获得的ETH - 买入花费的ETH
	profit := new(big.Float).Sub(ethReceived, tradeSize)

	return profit
}

func (aa *ArbitrageAnalyzer) assessRiskLevel(priceDiff, pool1Liquidity, pool2Liquidity *big.Float) string {
	// 基于价格差异和流动性评估风险
	priceDiffFloat, _ := priceDiff.Float64()
	pool1LiquidityFloat, _ := pool1Liquidity.Float64()
	pool2LiquidityFloat, _ := pool2Liquidity.Float64()

	if priceDiffFloat > 1.0 && pool1LiquidityFloat > 1000 && pool2LiquidityFloat > 1000 {
		return "LOW"
	} else if priceDiffFloat > 0.5 && pool1LiquidityFloat > 500 && pool2LiquidityFloat > 500 {
		return "MEDIUM"
	} else {
		return "HIGH"
	}
}

func (aa *ArbitrageAnalyzer) generateRecommendation(priceDiff *big.Float, riskLevel string, maxProfit *big.Float) string {
	priceDiffFloat, _ := priceDiff.Float64()
	maxProfitFloat, _ := maxProfit.Float64()

	if priceDiffFloat < 0.1 {
		return "价格差异太小，不建议交易"
	} else if riskLevel == "HIGH" {
		return "风险较高，建议谨慎交易"
	} else if maxProfitFloat < 0.01 {
		return "利润太小，不建议交易"
	} else {
		return "建议执行套利交易"
	}
}

func (aa *ArbitrageAnalysis) PrintAnalysis() {
	fmt.Printf("=== 详细套利分析 ===\n")
	fmt.Printf("时间: %s\n", aa.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("池子1: %s (价格: %.6f, 流动性: %.2f)\n",
		aa.Pool1Address.Hex(), aa.Pool1Price, aa.Pool1Liquidity)
	fmt.Printf("池子2: %s (价格: %.6f, 流动性: %.2f)\n",
		aa.Pool2Address.Hex(), aa.Pool2Price, aa.Pool2Liquidity)
	fmt.Printf("价格差异: %.4f%%\n", aa.PriceDiffPercent)
	fmt.Printf("交易方向: %s\n", aa.TradeDirection)
	fmt.Printf("最优交易规模: %.4f ETH\n", aa.OptimalTradeSize)
	fmt.Printf("最大利润: %.6f ETH\n", aa.MaxProfit)
	fmt.Printf("利润率: %.4f%%\n", aa.ProfitPercentage)
	fmt.Printf("风险等级: %s\n", aa.RiskLevel)
	fmt.Printf("流动性比率: %.2f\n", aa.LiquidityRatio)
	fmt.Printf("预估Gas成本: %.4f ETH\n", aa.EstimatedGasCost)
	fmt.Printf("交易建议: %s\n", aa.RecommendedAction)
	fmt.Printf("==================\n\n")
}
