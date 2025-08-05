package main

import (
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// ArbitrageOpportunity 套利机会结构
type ArbitrageOpportunity struct {
	Pool1Address    common.Address
	Pool2Address    common.Address
	Pool1Price      *big.Float
	Pool2Price      *big.Float
	PriceDifference *big.Float
	ProfitEstimate  *big.Float
	TradeAmount     *big.Float // 新增字段
	Pool1Liquidity  *big.Float // 新增
	Pool2Liquidity  *big.Float // 新增
	Pool1Amount0    *big.Float // 新增: 当前tick区间token0数量
	Pool1Amount1    *big.Float // 新增: 当前tick区间token1数量
	Pool2Amount0    *big.Float
	Pool2Amount1    *big.Float
	Timestamp       time.Time
	Direction       string // "pool1_to_pool2" 或 "pool2_to_pool1"
	Token0Symbol    string
	Token1Symbol    string
}

// ArbitrageMonitor 套利监控器
type ArbitrageMonitor struct {
	poolStateGetter PoolStateGetter
	pools           []common.Address
	opportunities   chan *ArbitrageOpportunity
	stopChan        chan struct{}
	wg              sync.WaitGroup
}

// NewArbitrageMonitor 创建新的套利监控器
func NewArbitrageMonitor(poolStateGetter PoolStateGetter, pools []common.Address) *ArbitrageMonitor {
	return &ArbitrageMonitor{
		poolStateGetter: poolStateGetter,
		pools:           pools,
		opportunities:   make(chan *ArbitrageOpportunity, 100),
		stopChan:        make(chan struct{}),
	}
}

// Start 启动监控
func (am *ArbitrageMonitor) Start() {
	am.wg.Add(1)
	go am.monitorLoop()
}

// Stop 停止监控
func (am *ArbitrageMonitor) Stop() {
	close(am.stopChan)
	am.wg.Wait()
}

// GetOpportunities 获取套利机会通道
func (am *ArbitrageMonitor) GetOpportunities() <-chan *ArbitrageOpportunity {
	return am.opportunities
}

// monitorLoop 监控循环
func (am *ArbitrageMonitor) monitorLoop() {
	defer am.wg.Done()
	ticker := time.NewTicker(1 * time.Second) // 每秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-am.stopChan:
			return
		case <-ticker.C:
			am.checkArbitrageOpportunities()
		}
	}
}

// checkArbitrageOpportunities 检查套利机会
func (am *ArbitrageMonitor) checkArbitrageOpportunities() {
	if len(am.pools) < 2 {
		return
	}

	// 获取两个池子的状态
	pool1State, err := am.poolStateGetter.GetPoolState(am.pools[0])
	if err != nil {
		Log.Error(fmt.Sprintf("Failed to get pool1 state: %v", err))
		return
	}

	pool2State, err := am.poolStateGetter.GetPoolState(am.pools[1])
	if err != nil {
		Log.Error(fmt.Sprintf("Failed to get pool2 state: %v", err))
		return
	}

	// 计算价格
	pool1Price := am.calculatePrice(pool1State)
	pool2Price := am.calculatePrice(pool2State)

	// 检测套利机会
	opportunity := am.detectArbitrageOpportunity(pool1State, pool2State, pool1Price, pool2Price)
	if opportunity != nil {
		select {
		case am.opportunities <- opportunity:
		default:
			// 通道满了，丢弃这个机会
		}
	}
}

// calculatePrice 计算池子价格
func (am *ArbitrageMonitor) calculatePrice(poolState *PoolState) *big.Float {
	if poolState == nil || poolState.Global == nil {
		return big.NewFloat(0)
	}

	currentTick := poolState.Global.Tick.Int64()
	// 使用Uniswap V3的价格公式: price = 1.0001^tick
	// 由于big.Float没有Exp方法，我们使用math.Pow
	price := new(big.Float).SetFloat64(math.Pow(1.0001, float64(currentTick)))

	return price
}

// detectArbitrageOpportunity 检测套利机会
func (am *ArbitrageMonitor) detectArbitrageOpportunity(pool1State, pool2State *PoolState, pool1Price, pool2Price *big.Float) *ArbitrageOpportunity {
	Log.Info("to detect")
	if pool1Price.Cmp(big.NewFloat(0)) == 0 || pool2Price.Cmp(big.NewFloat(0)) == 0 {
		return nil
	}

	analyzer := NewArbitrageAnalyzer(am.poolStateGetter)
	pool1Liquidity := analyzer.calculateLiquidity(pool1State)
	pool2Liquidity := analyzer.calculateLiquidity(pool2State)

	// 计算当前tick区间token数量，复用API核心逻辑
	currentTick1 := int32(pool1State.Global.Tick.Int64())
	tickSpacing1 := int32(pool1State.Global.TickSpacing.Int64())
	fromTick1, toTick1 := CalculateTickRange(currentTick1, 0, tickSpacing1)
	rangeLiquidityArray1 := BuildRangeLiquidityArray(pool1State.TickStates)
	rangeLiquidityArray1 = FilterRangeLiquidityArray(rangeLiquidityArray1, fromTick1, toTick1)
	rangeAmountArray1 := CalcRangeAmountArray(rangeLiquidityArray1, int(pool1State.Token0.Decimals), int(pool1State.Token1.Decimals))
	var amount1Amount0, amount1Amount1 *big.Float
	if len(rangeAmountArray1) > 0 {
		amount1Amount0 = rangeAmountArray1[0].Amount0
		amount1Amount1 = rangeAmountArray1[0].Amount1
	} else {
		amount1Amount0 = big.NewFloat(0)
		amount1Amount1 = big.NewFloat(0)
	}

	currentTick2 := int32(pool2State.Global.Tick.Int64())
	tickSpacing2 := int32(pool2State.Global.TickSpacing.Int64())
	fromTick2, toTick2 := CalculateTickRange(currentTick2, 0, tickSpacing2)
	rangeLiquidityArray2 := BuildRangeLiquidityArray(pool2State.TickStates)
	rangeLiquidityArray2 = FilterRangeLiquidityArray(rangeLiquidityArray2, fromTick2, toTick2)
	rangeAmountArray2 := CalcRangeAmountArray(rangeLiquidityArray2, int(pool2State.Token0.Decimals), int(pool2State.Token1.Decimals))
	var amount2Amount0, amount2Amount1 *big.Float
	if len(rangeAmountArray2) > 0 {
		amount2Amount0 = rangeAmountArray2[0].Amount0
		amount2Amount1 = rangeAmountArray2[0].Amount1
	} else {
		amount2Amount0 = big.NewFloat(0)
		amount2Amount1 = big.NewFloat(0)
	}

	// 取两个池子的token1数量的最小值作为tradeAmount，最大不超过10
	//maxTrade := big.NewFloat(10.0)
	tradeAmount := amount1Amount1
	if amount2Amount1.Cmp(amount1Amount1) < 0 {
		tradeAmount = amount2Amount1
	}
	//if tradeAmount.Cmp(maxTrade) > 0 {
	//	tradeAmount = maxTrade
	//}

	// 计算价格差异百分比
	priceDiff := new(big.Float).Sub(pool1Price, pool2Price)
	priceDiffAbs := new(big.Float).Abs(priceDiff)
	priceDiffPercent := new(big.Float).Quo(priceDiffAbs, pool1Price)
	priceDiffPercent.Mul(priceDiffPercent, big.NewFloat(100))

	// 设置最小套利阈值 (0.0001%)
	minArbitrageThreshold := big.NewFloat(0.0001)
	if priceDiffPercent.Cmp(minArbitrageThreshold) < 0 {
		return nil
	}

	// 确定套利方向
	var direction string
	var profitEstimate *big.Float

	if pool1Price.Cmp(pool2Price) > 0 {
		direction = "pool2_to_pool1"
		profitEstimate = am.calculateProfitEstimate(pool2State, pool1State, pool2Price, pool1Price, tradeAmount)
	} else {
		direction = "pool1_to_pool2"
		profitEstimate = am.calculateProfitEstimate(pool1State, pool2State, pool1Price, pool2Price, tradeAmount)
	}

	minTrade := big.NewFloat(0.0001)
	if tradeAmount == nil || tradeAmount.Cmp(minTrade) < 0 {
		return nil
	}

	return &ArbitrageOpportunity{
		Pool1Address:    am.pools[0],
		Pool2Address:    am.pools[1],
		Pool1Price:      pool1Price,
		Pool2Price:      pool2Price,
		PriceDifference: priceDiffPercent,
		ProfitEstimate:  profitEstimate,
		TradeAmount:     tradeAmount,
		Pool1Liquidity:  pool1Liquidity,
		Pool2Liquidity:  pool2Liquidity,
		Pool1Amount0:    amount1Amount0,
		Pool1Amount1:    amount1Amount1,
		Pool2Amount0:    amount2Amount0,
		Pool2Amount1:    amount2Amount1,
		Timestamp:       time.Now(),
		Direction:       direction,
		Token0Symbol:    pool1State.Token0.Symbol,
		Token1Symbol:    pool1State.Token1.Symbol,
	}
}

// calculateProfitEstimate 计算利润估算
func (am *ArbitrageMonitor) calculateProfitEstimate(buyPool, sellPool *PoolState, buyPrice, sellPrice, tradeAmount *big.Float) *big.Float {
	if tradeAmount == nil || tradeAmount.Cmp(big.NewFloat(0)) <= 0 {
		return big.NewFloat(0)
	}
	// 在低价池买入获得的token数量
	tokensBought := new(big.Float).Quo(tradeAmount, buyPrice)
	// 在高价池卖出获得的ETH数量
	ethReceived := new(big.Float).Mul(tokensBought, sellPrice)
	// 利润 = 卖出获得的ETH - 买入花费的ETH
	profit := new(big.Float).Sub(ethReceived, tradeAmount)
	return profit
}

// PrintOpportunity 打印套利机会
func (ao *ArbitrageOpportunity) PrintOpportunity() {
	fmt.Printf("=== 套利机会检测 ===\n")
	fmt.Printf("时间: %s\n", ao.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("池子1: %s (价格: %.6f)\n", ao.Pool1Address.Hex(), ao.Pool1Price)
	fmt.Printf("池子2: %s (价格: %.6f)\n", ao.Pool2Address.Hex(), ao.Pool2Price)
	fmt.Printf("价格差异: %.4f%%\n", ao.PriceDifference)
	fmt.Printf("套利方向: %s\n", ao.Direction)
	fmt.Printf("池子1当前tick区间最大可swap: %s %s, %s %s\n", ao.Pool1Amount0.Text('f', 6), ao.Token0Symbol, ao.Pool1Amount1.Text('f', 6), ao.Token1Symbol)
	fmt.Printf("池子2当前tick区间最大可swap: %s %s, %s %s\n", ao.Pool2Amount0.Text('f', 6), ao.Token0Symbol, ao.Pool2Amount1.Text('f', 6), ao.Token1Symbol)
	fmt.Printf("最终套利交易数量: %s %s\n", ao.TradeAmount.Text('f', 6), ao.Token1Symbol)
	fmt.Printf("预估利润: %s %s\n", ao.ProfitEstimate.Text('f', 6), ao.Token1Symbol)
	fmt.Printf("==================\n\n")
}
