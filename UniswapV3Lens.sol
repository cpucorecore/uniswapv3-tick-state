// SPDX-License-Identifier: MIT
pragma solidity ^0.8.27;

interface IPool {
    function tickSpacing() external view returns (int24);
    function tickBitmap(int16 wordPos) external view returns (uint256);
    function ticks(int24 tick) external view returns (uint128 liquidityGross, int128 liquidityNet);
}

struct Tick {
    int24 index;
    uint128 liquidityGross;
    int128 liquidityNet;
}

contract UniswapV3Lens {
    int24 internal constant MIN_TICK = -887272;
    int24 internal constant MAX_TICK = -MIN_TICK;

    function getAllTicks(IPool pool) external view returns (uint256 height, int24 tickSpacing, Tick[] memory ticks) {
        tickSpacing = pool.tickSpacing();
        int256 minWord = int16((MIN_TICK / tickSpacing) >> 8);
        int256 maxWord = int16((MAX_TICK / tickSpacing) >> 8);

        uint256 numTicks = 0;
        for (int256 word = minWord; word <= maxWord; word++) {
            uint256 bitmap = pool.tickBitmap(int16(word));
            if (bitmap == 0) continue;
            for (uint256 bit; bit < 256; bit++) if (bitmap & (1 << bit) > 0) numTicks++;
        }

        height = block.number;
        ticks = new Tick[](numTicks);
        uint256 idx = 0;
        for (int256 word = minWord; word <= maxWord; word++) {
            uint256 bitmap = pool.tickBitmap(int16(word));
            if (bitmap == 0) continue;
            for (uint256 bit; bit < 256; bit++) {
                if (bitmap & (1 << bit) == 0) continue;
                ticks[idx].index = int24((word << 8) + int256(bit)) * tickSpacing;
                (ticks[idx].liquidityGross, ticks[idx].liquidityNet) = pool.ticks(ticks[idx].index);
                idx++;
            }
        }
    }

    function getTicks(IPool pool, int24 tickStart, uint256 numTicks) external view returns (uint256 height, int24 tickSpacing, Tick[] memory ticks) {
        tickSpacing = pool.tickSpacing();
        int256 maxWord = int16((MAX_TICK / tickSpacing) >> 8);
        tickStart /= tickSpacing;
        int256 wordStart = int16(tickStart >> 8);
        uint256 bitStart = uint8(uint24(tickStart % 256));

        height = block.number;
        ticks = new Tick[](numTicks);
        uint256 idx = 0;
        for (int256 word = wordStart; word <= maxWord; word++) {
            uint256 bitmap = pool.tickBitmap(int16(word));
            if (bitmap == 0) continue;
            for (uint256 bit = word == wordStart ? bitStart : 0; bit < 256; bit++) {
                if (bitmap & (1 << bit) == 0) continue;
                ticks[idx].index = int24((word << 8) + int256(bit)) * tickSpacing;
                (ticks[idx].liquidityGross, ticks[idx].liquidityNet) = pool.ticks(ticks[idx].index);
                if (++idx >= numTicks) return (height, tickSpacing, ticks);
            }
        }
    }
}
