def calculate_uniswap_v3_amounts(tick, liquidity, token0_decimals, token1_decimals):
    """
    Calculate the amounts of token0 and token1 for a Uniswap v3 position
    given the current tick and liquidity.
    
    Args:
        tick (int): Current tick of the pool
        liquidity (int): Liquidity of the position
        token0_decimals (int): Decimals of token0 (default: 6 for USDC)
        token1_decimals (int): Decimals of token1 (default: 18 for WETH)
    
    Returns:
        tuple: (amount0, amount1) in human-readable units
    """
    # Constants
    TICK_SPACING = 10
    Q96 = 2**96
    
    # Calculate tickA and tickB (lower and upper ticks of the position)
    tickA = (int(tick / TICK_SPACING)) * TICK_SPACING
    tickB = tickA + TICK_SPACING
    
    # Calculate sqrt ratios
    sqrtA = (1.0001 ** (tickA / 2)) * Q96
    sqrtB = (1.0001 ** (tickB / 2)) * Q96
    
    # Calculate amount0 (token0 - USDC)
    amount0 = (liquidity * Q96 * (sqrtB - sqrtA) / sqrtB / sqrtA) / (10 ** token0_decimals)
    
    # Calculate amount1 (token1 - WETH)
    amount1 = liquidity * (sqrtB - sqrtA) / Q96 / (10 ** token1_decimals)
    
    return amount0, amount1

# Example usage
tick = -66625
liquidity =203160713452353941752068

amount0, amount1 = calculate_uniswap_v3_amounts(tick, liquidity,18,18)
print(f"Amount of token0 (USD1): {amount0:.18f}")
print(f"Amount of token1 (WBNB): {amount1:.18f}")
