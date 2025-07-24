import json

def calculate_uniswap_v3_amounts_range(inputJson, token0_decimals=18, token1_decimals=18):
    """
    inputJson: dict or str, must contain 'State' and 'Ticks'
    返回: [(tickLower, amount0, amount1), ...]
    """
    if isinstance(inputJson, str):
        data = json.loads(inputJson)
    else:
        data = inputJson
    
    tick_spacing = data['State']['tickSpacing']
    ticks = data['Ticks']
    # 按Tick升序排序
    ticks = sorted(ticks, key=lambda x: x['Tick'])
    
    # 构建所有tick边界
    tick_boundaries = [tick['Tick'] for tick in ticks]
    
    # 计算每个tick区间的liquidity前缀和
    prefix_liquidity = []
    current_liquidity = 0
    for tick in ticks:
        current_liquidity += tick['LiquidityNet']
        prefix_liquidity.append(current_liquidity)
    
    Q96 = 2**96
    results = []
    
    for i in range(len(tick_boundaries) - 1):
        tick_lower = tick_boundaries[i]
        tick_upper = tick_boundaries[i+1]
        liquidity = prefix_liquidity[i]
        # 以tickSpacing为步长遍历区间
        for t in range(tick_lower, tick_upper, tick_spacing):
            tickA = t
            tickB = t + tick_spacing
            # sqrtA, sqrtB 参考单区间公式
            sqrtA = (1.0001 ** (tickA / 2)) * Q96
            sqrtB = (1.0001 ** (tickB / 2)) * Q96
            # amount0
            amount0 = (liquidity * Q96 * (sqrtB - sqrtA) / sqrtB / sqrtA) / (10 ** token0_decimals)
            # amount1
            amount1 = liquidity * (sqrtB - sqrtA) / Q96 / (10 ** token1_decimals)
            results.append((tickA, amount0, amount1))
    return results