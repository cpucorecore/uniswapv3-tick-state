import json
from calc_range import calculate_uniswap_v3_amounts_range
import matplotlib.pyplot as plt

# 读取json文件
with open('input.json', 'r') as f:
    input_data = json.load(f)

# 计算结果
results = calculate_uniswap_v3_amounts_range(input_data)

# 拆分x和y
ticks = [tick for tick, amount0, amount1 in results]
amount0s = [amount0 for tick, amount0, amount1 in results]

# 画图
plt.figure(figsize=(16, 6))
plt.bar(ticks, amount0s, width=8)  # width可以根据tickSpacing调整
plt.xlabel('Tick')
plt.ylabel('Amount0')
plt.title('Amount0 per Tick')
plt.tight_layout()
plt.show()