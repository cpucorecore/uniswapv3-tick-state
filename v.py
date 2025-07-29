import math

def verify_calculation():
    tick_lower, tick_upper = 1000, 2000
    liquidity = 1000000  # 假设流动性为 1M

    # 方法1：累加
    total_amount0_1 = 0
    total_amount1_1 = 0

    for tick in range(tick_lower, tick_upper):
        sqrt_price_lower = 1.0001 ** (tick / 2)
        sqrt_price_upper = 1.0001 ** ((tick + 1) / 2)

        amount0 = liquidity * (sqrt_price_upper - sqrt_price_lower) / (sqrt_price_upper * sqrt_price_lower)
        amount1 = liquidity * (sqrt_price_upper - sqrt_price_lower)

        total_amount0_1 += amount0
        total_amount1_1 += amount1

    # 方法2：批量
    sqrt_price_lower = 1.0001 ** (tick_lower / 2)
    sqrt_price_upper = 1.0001 ** (tick_upper / 2)

    total_amount0_2 = liquidity * (sqrt_price_upper - sqrt_price_lower) / (sqrt_price_upper * sqrt_price_lower)
    total_amount1_2 = liquidity * (sqrt_price_upper - sqrt_price_lower)

    print(f"方法1 - amount0: {total_amount0_1:.6f}, amount1: {total_amount1_1:.6f}")
    print(f"方法2 - amount0: {total_amount0_2:.6f}, amount1: {total_amount1_2:.6f}")
    print(f"差异 - amount0: {abs(total_amount0_1 - total_amount0_2):.10f}")
    print(f"差异 - amount1: {abs(total_amount1_1 - total_amount1_2):.10f}")

verify_calculation()