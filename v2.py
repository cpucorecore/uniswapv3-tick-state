import math

def verify_tick_spacing_10():
    tick_lower, tick_upper = 1000, 2000
    tick_spacing = 10
    liquidity = 1000000  # 1M 流动性

    print(f"测试区间: [{tick_lower}, {tick_upper}], tickSpacing={tick_spacing}")
    print("=" * 60)

    # 方法1：逐个计算
    total_amount0_1 = 0
    total_amount1_1 = 0
    interval_count = 0

    print("方法1 - 逐个区间计算:")
    for tick in range(tick_lower, tick_upper, tick_spacing):
        tick_upper_interval = tick + tick_spacing
        interval_count += 1

        sqrt_price_lower = 1.0001 ** (tick / 2)
        sqrt_price_upper = 1.0001 ** (tick_upper_interval / 2)

        amount0 = liquidity * (sqrt_price_upper - sqrt_price_lower) / (sqrt_price_upper * sqrt_price_lower)
        amount1 = liquidity * (sqrt_price_upper - sqrt_price_lower)

        total_amount0_1 += amount0
        total_amount1_1 += amount1

        if interval_count <= 3:  # 只显示前3个区间
            print(f"  区间 [{tick}, {tick_upper_interval}]: amount0={amount0:.6f}, amount1={amount1:.6f}")

    print(f"  总共计算了 {interval_count} 个区间")
    print(f"  最终结果: amount0={total_amount0_1:.6f}, amount1={total_amount1_1:.6f}")
    print()

    # 方法2：批量计算
    print("方法2 - 批量计算整个区间:")
    sqrt_price_lower = 1.0001 ** (tick_lower / 2)
    sqrt_price_upper = 1.0001 ** (tick_upper / 2)

    total_amount0_2 = liquidity * (sqrt_price_upper - sqrt_price_lower) / (sqrt_price_upper * sqrt_price_lower)
    total_amount1_2 = liquidity * (sqrt_price_upper - sqrt_price_lower)

    print(f"  直接计算 [{tick_lower}, {tick_upper}]")
    print(f"  最终结果: amount0={total_amount0_2:.6f}, amount1={total_amount1_2:.6f}")
    print()

    # 对比结果
    print("结果对比:")
    print(f"  amount0 差异: {abs(total_amount0_1 - total_amount0_2):.10f}")
    print(f"  amount1 差异: {abs(total_amount1_1 - total_amount1_2):.10f}")
    print(f"  计算次数对比: {interval_count} vs 1 (优化 {interval_count}x)")

    return total_amount0_1, total_amount1_1, total_amount0_2, total_amount1_2

# 运行验证
verify_tick_spacing_10()