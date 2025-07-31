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

## 配置文件说明

### 启动参数

```bash
# 显示版本信息
./uniswapv3-tick-state -v

# 指定配置文件
./uniswapv3-tick-state -c config.json

# 指定数据库路径（覆盖配置文件中的设置）
./uniswapv3-tick-state -db /path/to/database

# 同时指定配置文件和数据库路径
./uniswapv3-tick-state -c config.json -db /path/to/database
```

### 配置文件格式

配置文件为JSON格式，包含以下配置项：

#### 日志配置 (log)
```json
{
  "async": false,           // 是否异步写入日志
  "buffer_size": 1000000,   // 日志缓冲区大小
  "flush_interval": 1       // 日志刷新间隔（秒）
}
```

#### 以太坊RPC配置 (eth_rpc)
```json
{
  "http": "https://bsc-dataseed.binance.org/",     // HTTP RPC地址
  "archive": "https://bsc-dataseed.binance.org/",  // 归档节点地址
  "ws": "ws://bsc-dataseed.binance.org/"           // WebSocket地址
}
```

#### 区块爬虫配置 (block_crawler)
```json
{
  "pool_size": 1,        // 工作池大小
  "from_height": 0       // 起始区块高度
}
```

#### Redis配置 (redis)
```json
{
  "addr": "localhost:6379",  // Redis地址
  "username": "",            // Redis用户名
  "password": ""             // Redis密码
}
```

#### RocksDB配置 (rocksdb)
```json
{
  "enable_log": true,                    // 是否启用RocksDB日志
  "block_cache_size": 1073741824,        // 块缓存大小（字节），默认1GB
  "write_buffer_size": 134217728,        // 写缓冲区大小（字节），默认128MB
  "max_write_buffer_number": 2,          // 最大写缓冲区数量
  "db_path": ".db"                       // 数据库路径，默认为当前目录下的.db
}
```

### 配置建议

#### RocksDB性能调优
- **block_cache_size**: 建议设置为可用内存的25-30%
- **write_buffer_size**: 建议设置为64MB-256MB
- **max_write_buffer_number**: 建议设置为2-4

#### 内存使用估算
- RocksDB缓存: `block_cache_size + write_buffer_size * max_write_buffer_number`
- 示例: 1GB + 128MB × 2 = 1.25GB

### 示例配置文件

参考 `config.example.json` 文件获取完整的配置示例。

