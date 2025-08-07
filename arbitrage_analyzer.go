package main

import (
	"fmt"
	"math"
	"math/big"
	"time"

	"go.uber.org/zap"

	"github.com/ethereum/go-ethereum/common"
)

type ArbitrageAnalysis struct {
	Timestamp        time.Time      `json:"timestamp"`
	Pool1Address     common.Address `json:"pool1_address"`
	Pool2Address     common.Address `json:"pool2_address"`
	Pool1PriceUSD    *big.Float     `json:"pool_1_price_usd"`
	Pool2PriceUSD    *big.Float     `json:"pool_2_price_usd"`
	PriceDiff        *big.Float     `json:"price_diff"`
	PriceDiffPercent *big.Float     `json:"price_diff_percent"`

	// 详细分析
	OptimalTradeSizeUSD *big.Float `json:"optimal_trade_size_usd"` // 新增：最大可成交USD规模
	NonUSDTokenSymbol   string     `json:"non_usd_token_symbol"`   // 新增：非USD币种名
	NonUSDTokenAmount   *big.Float `json:"non_usd_token_amount"`   // 新增：最大可成交non-USD数量
	MaxProfit           *big.Float `json:"max_profit"`
	ProfitPercentage    *big.Float `json:"profit_percentage"`
	RiskLevel           string     `json:"risk_level"`

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
	if !pool1State.IsUSDPool() {
		return nil, fmt.Errorf("pool1 have no usd token")
	}
	pool2State, err := aa.poolStateGetter.GetPoolState(pool2Addr)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool2 state: %v", err)
	}
	if !pool2State.IsUSDPool() {
		return nil, fmt.Errorf("pool1 have no usd token")
	}

	pool1Tick := int32(pool1State.Global.Tick.Int64())
	pool2Tick := int32(pool2State.Global.Tick.Int64())
	pool1PriceUSD := calcUSDPrice(pool1Tick, pool1State.Token0.Address, pool1State.Token1.Address)
	pool2PriceUSD := calcUSDPrice(pool2Tick, pool2State.Token0.Address, pool2State.Token1.Address)

	priceDiff := new(big.Float).Sub(pool1PriceUSD, pool2PriceUSD)
	priceDiffAbs := new(big.Float).Abs(priceDiff)
	var priceDiffPercent *big.Float
	if pool1PriceUSD.Cmp(big.NewFloat(0)) == 0 {
		priceDiffPercent = big.NewFloat(0)
	} else {
		priceDiffPercent = new(big.Float).Quo(priceDiffAbs, pool1PriceUSD)
		priceDiffPercent.Mul(priceDiffPercent, big.NewFloat(100))
	}

	var tradeDirection string
	var buyPriceUSD, sellPriceUSD *big.Float
	var buyPool, sellPool *PoolState
	if pool1PriceUSD.Cmp(pool2PriceUSD) > 0 {
		tradeDirection = "pool2_to_pool1"
		buyPriceUSD = pool2PriceUSD
		sellPriceUSD = pool1PriceUSD
		buyPool = pool2State
		sellPool = pool1State
	} else {
		tradeDirection = "pool1_to_pool2"
		buyPriceUSD = pool1PriceUSD
		sellPriceUSD = pool2PriceUSD
		buyPool = pool1State
		sellPool = pool2State
	}

	// 计算最优交易规模（non-USD数量）和币种
	optimalTradeSize, nonUSDTokenSymbol := aa.calculateOptimalTradeSize(buyPool, sellPool)

	// 最大可成交USD规模（用买入池USD价格，保守）
	optimalTradeSizeUSD := new(big.Float).Mul(optimalTradeSize, buyPriceUSD)

	// 利润 = optimalTradeSize * (sellPriceUSD - buyPriceUSD)
	maxProfit := new(big.Float).Mul(optimalTradeSize, new(big.Float).Sub(sellPriceUSD, buyPriceUSD))
	if maxProfit.Cmp(optimalTradeSizeUSD) > 0 {
		maxProfit = new(big.Float).Set(optimalTradeSizeUSD)
	}

	// 利润率
	var profitPercentage *big.Float
	if optimalTradeSize.Cmp(big.NewFloat(0)) == 0 {
		profitPercentage = big.NewFloat(0)
	} else {
		profitPercentage = new(big.Float).Quo(maxProfit, optimalTradeSizeUSD)
		profitPercentage.Mul(profitPercentage, big.NewFloat(100))
	}

	// 评估风险等级
	pool1Liquidity := aa.calculateLiquidity(pool1State)
	pool2Liquidity := aa.calculateLiquidity(pool2State)
	liquidityRatio := new(big.Float)
	if pool2Liquidity.Cmp(big.NewFloat(0)) == 0 {
		liquidityRatio = big.NewFloat(0)
	} else {
		liquidityRatio = new(big.Float).Quo(pool1Liquidity, pool2Liquidity)
	}
	riskLevel := aa.assessRiskLevel(priceDiffPercent, pool1Liquidity, pool2Liquidity)

	recommendedAction := aa.generateRecommendation(priceDiffPercent, riskLevel, maxProfit)
	estimatedGasCost := big.NewFloat(20)

	return &ArbitrageAnalysis{
		Timestamp:           time.Now(),
		Pool1Address:        pool1Addr,
		Pool2Address:        pool2Addr,
		Pool1PriceUSD:       pool1PriceUSD,
		Pool2PriceUSD:       pool2PriceUSD,
		PriceDiff:           priceDiff,
		PriceDiffPercent:    priceDiffPercent,
		OptimalTradeSizeUSD: optimalTradeSizeUSD,
		NonUSDTokenSymbol:   nonUSDTokenSymbol,
		NonUSDTokenAmount:   optimalTradeSize,
		MaxProfit:           maxProfit,
		ProfitPercentage:    profitPercentage,
		RiskLevel:           riskLevel,
		Pool1Liquidity:      pool1Liquidity,
		Pool2Liquidity:      pool2Liquidity,
		LiquidityRatio:      liquidityRatio,
		RecommendedAction:   recommendedAction,
		TradeDirection:      tradeDirection,
		EstimatedGasCost:    estimatedGasCost,
	}, nil
}

var (
	bigFloat0 = big.NewFloat(0)
)

func calcTickPrice(tick int32) *big.Float {
	return new(big.Float).SetFloat64(math.Pow(1.0001, float64(tick)))
}

func calcUSDPrice(tick int32, addr0, addr1 common.Address) *big.Float {
	if IsUSD(addr0) {
		return calcTickPrice(tick)
	} else if IsUSD(addr1) {
		return new(big.Float).Quo(big.NewFloat(1), calcTickPrice(tick))
	} else {
		return bigFloat0
	}
}

func (aa *ArbitrageAnalyzer) calculateUniswapV3Price(poolState *PoolState) *big.Float {
	if poolState == nil || poolState.Global == nil {
		return big.NewFloat(0)
	}

	currentTick := poolState.Global.Tick.Int64()
	return new(big.Float).SetFloat64(math.Pow(1.0001, float64(currentTick)))
}

func (aa *ArbitrageAnalyzer) calculatePriceInUSD(poolState *PoolState) (*big.Float, bool) {
	if poolState == nil || poolState.Global == nil {
		return big.NewFloat(0), false
	}

	currentTick := poolState.Global.Tick.Int64()
	price := new(big.Float).SetFloat64(math.Pow(1.0001, float64(currentTick)))
	token0IsUSD := IsUSD(poolState.Token0.Address)
	if token0IsUSD {
		return price, true
	}

	token1IsUSD := IsUSD(poolState.Token1.Address)
	if token1IsUSD {
		return big.NewFloat(0).Quo(big.NewFloat(1), price), true
	}

	return big.NewFloat(0), false
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

// getTradableAmountInCurrentTickRange 计算当前tick区间的token0和token1可交易数量
func (aa *ArbitrageAnalyzer) getTradableAmountInCurrentTickRange(poolState *PoolState) (*big.Float, *big.Float) {
	if poolState == nil || poolState.Global == nil {
		return big.NewFloat(0), big.NewFloat(0)
	}
	currentTick := int32(poolState.Global.Tick.Int64())
	tickSpacing := int32(poolState.Global.TickSpacing.Int64())
	fromTick, toTick := CalculateTickRange(currentTick, 0, tickSpacing)

	rangeLiquidityArray := BuildRangeLiquidityArray(poolState.TickStates)
	rangeLiquidityArray = FilterRangeLiquidityArray(rangeLiquidityArray, fromTick, toTick)
	rangeLiquidityArray = SplitRangeLiquidityArray(rangeLiquidityArray, tickSpacing)
	rangeLiquidityArray = FilterRangeLiquidityArray(rangeLiquidityArray, fromTick, toTick)
	rangeAmountArray := CalcRangeAmountArray(rangeLiquidityArray, int(poolState.Token0.Decimals), int(poolState.Token1.Decimals))

	var token0Amount, token1Amount *big.Float
	if len(rangeAmountArray) > 0 {
		token0Amount = rangeAmountArray[0].Amount0
		token1Amount = rangeAmountArray[0].Amount1
		Log.Info("[getTradableAmountInCurrentTickRange] matched tick range", zap.Int32("TickLower", rangeAmountArray[0].TickLower), zap.Int32("TickUpper", rangeAmountArray[0].TickUpper), zap.String("Amount0", rangeAmountArray[0].Amount0.Text('f', 18)), zap.String("Amount1", rangeAmountArray[0].Amount1.Text('f', 18)))
	} else {
		token0Amount = big.NewFloat(0)
		token1Amount = big.NewFloat(0)
		Log.Warn("[getTradableAmountInCurrentTickRange] no matching tick range found for currentTick", zap.Int32("currentTick", currentTick))
	}
	Log.Info("[getTradableAmountInCurrentTickRange] final token0 amount", zap.String("value", token0Amount.Text('f', 18)))
	Log.Info("[getTradableAmountInCurrentTickRange] final token1 amount", zap.String("value", token1Amount.Text('f', 18)))
	return token0Amount, token1Amount
}

func (aa *ArbitrageAnalyzer) calculateOptimalTradeSize(buyPool, sellPool *PoolState) (*big.Float, string) {
	buyToken0, buyToken1 := aa.getTradableAmountInCurrentTickRange(buyPool)
	sellToken0, sellToken1 := aa.getTradableAmountInCurrentTickRange(sellPool)

	var buyNonUSD, sellNonUSD *big.Float
	var nonUSDTokenSymbol string
	if IsUSD(buyPool.Token0.Address) {
		buyNonUSD = buyToken1
		nonUSDTokenSymbol = buyPool.Token1.Symbol
	} else {
		buyNonUSD = buyToken0
		nonUSDTokenSymbol = buyPool.Token0.Symbol
	}
	if IsUSD(sellPool.Token0.Address) {
		sellNonUSD = sellToken1
	} else {
		sellNonUSD = sellToken0
	}

	optimalSize := buyNonUSD
	if sellNonUSD.Cmp(buyNonUSD) < 0 {
		optimalSize = sellNonUSD
	}
	return optimalSize, nonUSDTokenSymbol
}

func (aa *ArbitrageAnalyzer) calculateMaxProfit(tradeSize, buyPrice, sellPrice *big.Float) *big.Float {
	// 新的利润计算逻辑 (以USD为基准)
	// tradeSize: 最优交易规模，单位是非稳定币的数量
	// buyPrice, sellPrice: 单位是 USD / 非稳定币
	// 成本 = tradeSize * buyPrice (USD)
	// 收入 = tradeSize * sellPrice (USD)
	// 利润 = 收入 - 成本 = tradeSize * (sellPrice - buyPrice)
	priceDiff := new(big.Float).Sub(sellPrice, buyPrice)
	profit := new(big.Float).Mul(tradeSize, priceDiff)
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
		aa.Pool1Address.Hex(), aa.Pool1PriceUSD, aa.Pool1Liquidity)
	fmt.Printf("池子2: %s (价格: %.6f, 流动性: %.2f)\n",
		aa.Pool2Address.Hex(), aa.Pool2PriceUSD, aa.Pool2Liquidity)
	fmt.Printf("价格差异: %.4f%%\n", aa.PriceDiffPercent)
	fmt.Printf("交易方向 (买入->卖出): %s\n", aa.TradeDirection)
	fmt.Printf("最优交易规模 (非稳定币): %.4f\n", aa.OptimalTradeSizeUSD)
	fmt.Printf("非稳定币数量: %.4f\n", aa.NonUSDTokenAmount)
	fmt.Printf("非稳定币符号: %s\n", aa.NonUSDTokenSymbol)
	fmt.Printf("最大利润 (USD): %.6f\n", aa.MaxProfit)
	fmt.Printf("利润率: %.4f%%\n", aa.ProfitPercentage)
	fmt.Printf("风险等级: %s\n", aa.RiskLevel)
	fmt.Printf("流动性比率: %.2f\n", aa.LiquidityRatio)
	fmt.Printf("预估Gas成本 (USD): %.4f\n", aa.EstimatedGasCost)
	fmt.Printf("交易建议: %s\n", aa.RecommendedAction)
	fmt.Printf("==================\n\n")
}
