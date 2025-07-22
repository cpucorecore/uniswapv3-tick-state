"""
{
  "data": {
    "pools": [
      {
        "liquidity": "1345094269907828626",
        "tick": "193905",
        "ticks": [
          {
            "liquidityGross": "2127118770244",
            "liquidityNet": "2127118770244",
            "tickIdx": "190020"
          },
          {
            "liquidityGross": "721001902721267",
            "liquidityNet": "721001902721267",
            "tickIdx": "190080"
          },
          {
            "liquidityGross": "331229290891174",
            "liquidityNet": "331229290891174",
            "tickIdx": "190140"
          },
          {
            "liquidityGross": "57832540361293711",
            "liquidityNet": "-56750035488392889",
            "tickIdx": "190200"
          },
          {
            "liquidityGross": "261012893385367",
            "liquidityNet": "261012893385367",
            "tickIdx": "190260"
          },
          {
            "liquidityGross": "26564542835069",
            "liquidityNet": "26564542835069",
            "tickIdx": "190320"
          },
          {
            "liquidityGross": "17447684854142",
            "liquidityNet": "17447684854142",
            "tickIdx": "190380"
          },
          {
            "liquidityGross": "244028592779770",
            "liquidityNet": "244028592779770",
            "tickIdx": "190440"
          },
          {
            "liquidityGross": "235716921815710",
            "liquidityNet": "235716921815710",
            "tickIdx": "190500"
          },
          {
            "liquidityGross": "22916900530797",
            "liquidityNet": "22916900530797",
            "tickIdx": "190560"
          }
        ]
      }
    ]
  }
}
"""
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
    TICK_SPACING = 1
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
tick = 102106
liquidity =1527488668366266481406253 

amount0, amount1 = calculate_uniswap_v3_amounts(tick, liquidity,18,18)
print(f"Amount of token0 (USDC): {amount0:.6f}")
print(f"Amount of token1 (WETH): {amount1:.18f}")
