# Uniswap V3 Pool State API 接口文档

## 接口概述

该API提供Uniswap V3池子状态查询功能，支持获取流动性信息、token数量计算等。

## 基础信息

- **服务地址**: `http://192.168.100.16:29292`
- **接口路径**: `/pool_state`
- **请求方法**: `GET`
- **Content-Type**: `application/json` 或 `text/html`

## 请求参数

### 必需参数

| 参数名 | 类型 | 必填 | 说明 | 示例 |
|--------|------|------|------|------|
| `address` | string | 是 | Uniswap V3池子合约地址 | `0x172fcD41E0913e95784454622d1c3724f546f849` |
| `tick_offset` | integer | 是 | 当前tick前后查询的tickSpacing区间数量 | `100` |

### 可选参数

| 参数名 | 类型 | 默认值 | 说明 | 示例 |
|--------|------|--------|------|------|
| `type` | string | `"2"` | 查询类型，详见下方说明 | `"1"`, `"2"`, `"3"` |
| `format` | string | `"html"` | 响应格式 | `"json"`, `"html"` |

## 参数详细说明

### type 参数说明

| 值 | 说明 | 适用format |
|----|------|------------|
| `"1"` | 返回原始poolState数据 | 仅支持 `json` |
| `"2"` | 使用原始tick区间计算token数量 | `json`, `html` |
| `"3"` | 根据tickSpacing计算更细粒度的token数量 | `json`, `html` |

### format 参数说明

| 值 | 说明 | Content-Type |
|----|------|--------------|
| `"json"` | 返回JSON格式数据 | `application/json` |
| `"html"` | 返回HTML图表页面 | `text/html` |

## 请求示例

### 1. 获取原始poolState数据
```bash
GET /pool_state?address=0x172fcD41E0913e95784454622d1c3724f546f849&tick_offset=100&type=1&format=json
```

### 2. 使用原始tick区间计算，返回JSON
```bash
GET /pool_state?address=0x172fcD41E0913e95784454622d1c3724f546f849&tick_offset=100&type=2&format=json
```

### 3. 使用原始tick区间计算，返回HTML图表
```bash
GET /pool_state?address=0x172fcD41E0913e95784454622d1c3724f546f849&tick_offset=100&type=2&format=html
```

### 4. 使用tickSpacing细粒度计算，返回JSON
```bash
GET /pool_state?address=0x172fcD41E0913e95784454622d1c3724f546f849&tick_offset=100&type=3&format=json
```

### 5. 使用tickSpacing细粒度计算，返回HTML图表
```bash
GET /pool_state?address=0x172fcD41E0913e95784454622d1c3724f546f849&tick_offset=100&type=3&format=html
```

## 响应格式

### JSON 响应格式

#### type=1 响应示例
```json
{
  "Token0": {
    "Symbol": "USDC",
    "Decimals": 6
  },
  "Token1": {
    "Symbol": "ETH",
    "Decimals": 18
  },
  "Global": {
    "Height": "12345678",
    "TickSpacing": "60",
    "Tick": "12345",
    "Liquidity": "1000000000000000000",
    "SqrtPriceX96": "123456789012345678901234567890"
  },
  "TickStates": [
    {
      "Tick": 12000,
      "LiquidityNet": "1000000000000000000"
    }
  ]
}
```

#### type=2/3 响应示例
```json
[
  {
    "TickLower": 12000,
    "TickUpper": 12060,
    "Liquidity": "1000000000000000000",
    "Amount0": "123.456",
    "Amount1": "0.789"
  },
  {
    "TickLower": 12060,
    "TickUpper": 12120,
    "Liquidity": "2000000000000000000",
    "Amount0": "234.567",
    "Amount1": "0.890"
  }
]
```

### HTML 响应格式

HTML响应包含一个交互式图表，显示：
- 区块高度
- 交易对信息
- 当前tick位置
- tickSpacing
- 当前tick价格
- 柱状图显示各tick区间的token数量

## 错误响应

### 400 Bad Request
```json
{
  "error": "missing parameter: address"
}
```

### 500 Internal Server Error
```json
{
  "error": "get tick states error: connection failed"
}
```

## 常见错误码

| HTTP状态码 | 错误信息 | 说明 |
|------------|----------|------|
| 400 | `missing parameter: address` | 缺少必需参数address |
| 400 | `missing parameter: tick_offset` | 缺少必需参数tick_offset |
| 400 | `invalid tick_offset format` | tick_offset格式错误 |
| 400 | `no pool info` | 池子信息不存在 |
| 400 | `pool filtered` | 池子被过滤 |
| 400 | `type=1 only supports format=json` | type=1只支持json格式 |
| 400 | `unsupported format` | 不支持的format值 |
| 500 | `get tick states error: *` | 获取tick状态失败 |
| 500 | `json marshal error` | JSON序列化失败 |
| 500 | `render error` | HTML渲染失败 |

## 性能说明

- **type=2**: 使用原始tick区间计算，性能较好
- **type=3**: 按tickSpacing细粒度计算，数据更详细但性能稍慢
- **tick_offset**: 影响查询的tick范围，值越大查询范围越大，性能消耗越高

## 使用建议

1. **调试阶段**: 使用 `type=1&format=json` 查看原始数据
2. **数据分析**: 使用 `type=2&format=json` 获取计算结果
3. **可视化展示**: 使用 `type=2&format=html` 或 `type=3&format=html` 查看图表
4. **tick_offset选择**: 根据需要的精度选择合适的值，通常10-100之间

## 完整示例URL

```bash
# 获取USDC/ETH池子的细粒度token数量图表
http://192.168.100.16:29292/pool_state?address=0x172fcD41E0913e95784454622d1c3724f546f849&tick_offset=100&type=3&format=html
```

