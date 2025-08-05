#!/bin/bash

# 套利监控测试脚本

echo "=== 跨池套利监控测试 ==="
echo ""

# 检查可执行文件是否存在
if [ ! -f "./uniswapv3-tick-state" ]; then
    echo "错误: 可执行文件不存在，请先编译项目"
    echo "运行: go build -o uniswapv3-tick-state ."
    exit 1
fi

# 测试参数
POOL1="0xBe141893E4c6AD9272e8C04BAB7E6a10604501a5"
POOL2="0x9F599F3D64a9D99eA21e68127Bb6CE99f893DA61"

echo "池子1: $POOL1"
echo "池子2: $POOL2"
echo ""

echo "启动套利监控..."
echo "按 Ctrl+C 停止监控"
echo ""

# 启动套利监控
./uniswapv3-tick-state -arbitrage -pool1 $POOL1 -pool2 $POOL2 