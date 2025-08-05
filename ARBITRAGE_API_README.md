# 套利检测API接口文档

## 接口概述

新增的套利检测API接口，用于检测两个Uniswap V3池子之间的套利机会，提供详细的分析结果。

## 接口信息

- **接口路径**: `/arbitrage_check`
- **请求方法**: `GET`
- **Content-Type**: `application/json`

## 请求参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| `pool1` | string | 是 | 第一个池子合约地址 | `0xBe141893E4c6AD9272e8C04BAB7E6a10604501a5` |
| `pool2` | string | 是 | 第二个池子合约地址 | `0x9F599F3D64a9D99eA21e68127Bb6CE99f893DA61` |

## 请求示例

```bash
GET /arbitrage_check?pool1=0xBe141893E4c6AD9272e8C04BAB7E6a10604501a5&pool2=0x9F599F3D64a9D99eA21e68127Bb6CE99f893DA61
```

## 响应格式

### 成功响应

```json
{
  "pool1_address": "0xBe141893E4c6AD9272e8C04BAB7E6a10604501a5",
  "pool2_address": "0x9F599F3D64a9D99eA21e68127Bb6CE99f893DA61",
  "pool1_price": "3554.576500",
  "pool2_price": "3554.931958",
  "price_difference": "0.0100",
  "timestamp": "2025-08-04T18:03:52Z",
  "optimal_trade_size": "0.956593",
  "max_profit": "0.001000",
  "profit_percentage": "0.1046",
  "risk_level": "LOW",
  "pool1_liquidity": "817221464503315399537967.000000",
  "pool2_liquidity": "13265305795458620581239.000000",
  "liquidity_ratio": "61.58",
  "recommended_action": "建议执行套利交易",
  "trade_direction": "pool1_to_pool2",
  "estimated_gas_cost": "0.005000"
}
```

### 字段说明

| 字段名 | 类型 | 说明 |
|--------|------|------|
| `pool1_address` | string | 第一个池子地址 |
| `pool2_address` | string | 第二个池子地址 |
| `pool1_price` | string | 池子1当前价格 |
| `pool2_price` | string | 池子2当前价格 |
| `price_difference` | string | 价格差异百分比 |
| `timestamp` | string | 检测时间 |
| `optimal_trade_size` | string | 最优交易规模（ETH） |
| `max_profit` | string | 最大预估利润（ETH） |
| `profit_percentage` | string | 利润率百分比 |
| `risk_level` | string | 风险等级（LOW/MEDIUM/HIGH） |
| `pool1_liquidity` | string | 池子1当前tick流动性 |
| `pool2_liquidity` | string | 池子2当前tick流动性 |
| `liquidity_ratio` | string | 流动性比率 |
| `recommended_action` | string | 交易建议 |
| `trade_direction` | string | 套利方向 |
| `estimated_gas_cost` | string | 预估Gas成本（ETH） |

### 错误响应

#### 400 Bad Request
```json
{
  "error": "missing parameter: pool1 or pool2"
}
```

#### 500 Internal Server Error
```json
{
  "error": "get pool1 state error: no pair info"
}
```

## 使用示例

### cURL示例
```bash
curl "http://localhost:29292/arbitrage_check?pool1=0xBe141893E4c6AD9272e8C04BAB7E6a10604501a5&pool2=0x9F599F3D64a9D99eA21e68127Bb6CE99f893DA61"
```

### JavaScript示例
```javascript
fetch('http://localhost:29292/arbitrage_check?pool1=0xBe141893E4c6AD9272e8C04BAB7E6a10604501a5&pool2=0x9F599F3D64a9D99eA21e68127Bb6CE99f893DA61')
  .then(response => response.json())
  .then(data => {
    console.log('套利分析结果:', data);
    if (data.recommended_action === '建议执行套利交易') {
      console.log('发现套利机会！');
      console.log('最优交易规模:', data.optimal_trade_size, 'ETH');
      console.log('预估利润:', data.max_profit, 'ETH');
    }
  });
```

### Python示例
```python
import requests

url = "http://localhost:29292/arbitrage_check"
params = {
    "pool1": "0xBe141893E4c6AD9272e8C04BAB7E6a10604501a5",
    "pool2": "0x9F599F3D64a9D99eA21e68127Bb6CE99f893DA61"
}

response = requests.get(url, params=params)
if response.status_code == 200:
    data = response.json()
    print("套利分析结果:", data)
    if data["recommended_action"] == "建议执行套利交易":
        print("发现套利机会！")
        print("最优交易规模:", data["optimal_trade_size"], "ETH")
        print("预估利润:", data["max_profit"], "ETH")
else:
    print("请求失败:", response.text)
```

## 注意事项

1. **实时性**: API返回的是当前时刻的套利分析结果
2. **准确性**: 基于Uniswap V3的真实价格和流动性数据
3. **风险提示**: 套利建议仅供参考，实际交易需考虑手续费、滑点等因素
4. **频率限制**: 建议合理控制API调用频率，避免对服务造成压力

## 错误码说明

| HTTP状态码 | 错误信息 | 说明 |
|------------|----------|------|
| 400 | `missing parameter: pool1 or pool2` | 缺少必需参数 |
| 500 | `get pool1/2 state error: *` | 获取池子状态失败 |
| 500 | `analyze arbitrage error: *` | 套利分析失败 |
| 500 | `json marshal error` | JSON序列化失败 | 