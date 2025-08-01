package main

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"testing"
)

func newTestRepo(t *testing.T) DB {
	name := t.TempDir()
	db, err := NewRocksDB(name, &RocksDBOptions{
		EnableLog:            false,
		BlockCacheSize:       1024 * 1024 * 100,
		WriteBufferSize:      1024 * 1024 * 10,
		MaxWriteBufferNumber: 1,
	})
	if err != nil {
		t.Fatalf("failed to create rocksdb: %v", err)
	}
	return NewDB(db)
}

func Test_SetTickState_GetTickState_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x1000000000000000000000000000000000000001")
	for _, tick := range []int32{123, -123} {
		ts := &TickState{Tick: tick, LiquidityNet: big.NewInt(123456)}
		if err := repo.SetTickState(addr, ts); err != nil {
			t.Fatalf("SetTickState failed: %v", err)
		}
		ts2, err := repo.GetTickState(addr, tick)
		if err != nil {
			t.Fatalf("GetTickState failed: %v", err)
		}
		if ts2.Tick != ts.Tick || ts2.LiquidityNet.Cmp(ts.LiquidityNet) != 0 {
			t.Fatalf("GetTickState: want %+v, got %+v", ts, ts2)
		}
	}
}

func Test_GetPoolTicks_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x4000000000000000000000000000000000000004")
	_ = repo.SetTickState(addr, &TickState{Tick: 10, LiquidityNet: big.NewInt(10)})
	_ = repo.SetTickState(addr, &TickState{Tick: -10, LiquidityNet: big.NewInt(-10)})
	states, err := repo.GetTickStates(addr)
	if err != nil || len(states) != 2 {
		t.Fatalf("GetTickStates failed or empty")
	}
}

func Test_SetGetCurrentTick_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x5000000000000000000000000000000000000005")
	for _, v := range []int32{12345, -12345} {
		if err := repo.SetCurrentTick(addr, v); err != nil {
			t.Fatalf("SetCurrentTick failed: %v", err)
		}
		v2, err := repo.GetCurrentTick(addr)
		if err != nil {
			t.Fatalf("GetCurrentTick failed: %v", err)
		}
		if v != v2 {
			t.Fatalf("GetCurrentTick: want %d, got %d", v, v2)
		}
	}
}

func Test_SetGetTickSpacing_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x6000000000000000000000000000000000000006")
	for _, v := range []int32{60, -60} {
		if err := repo.SetTickSpacing(addr, v); err != nil {
			t.Fatalf("SetTickSpacing failed: %v", err)
		}
		v2, err := repo.GetTickSpacing(addr)
		if err != nil {
			t.Fatalf("GetTickSpacing failed: %v", err)
		}
		if v != v2 {
			t.Fatalf("GetTickSpacing: want %d, got %d", v, v2)
		}
	}
}

func Test_SetGetPoolHeight(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x8000000000000000000000000000000000000008")
	for _, h := range []uint64{123456, 0} {
		if err := repo.SetHeight(addr, h); err != nil {
			t.Fatalf("SetHeight failed: %v", err)
		}
		h2, err := repo.GetHeight(addr)
		if err != nil {
			t.Fatalf("GetHeight failed: %v", err)
		}
		if h != h2 {
			t.Fatalf("GetHeight: want %d, got %d", h, h2)
		}
	}
}

func Test_SetGetHeight(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	for _, h := range []uint64{987654321, 0} {
		if err := repo.SetFinishHeight(h); err != nil {
			t.Fatalf("SetFinishHeight failed: %v", err)
		}
		h2, err := repo.GetFinishHeight()
		if err != nil {
			t.Fatalf("GetFinishHeight failed: %v", err)
		}
		if h != h2 {
			t.Fatalf("GetFinishHeight: want %d, got %d", h, h2)
		}
	}
}

func Test_SetGetPoolState_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x9000000000000000000000000000000000000009")
	for _, tick := range []int32{100, -100} {
		poolTicks := &PoolState{
			Global: &PoolGlobalState{
				Height:      big.NewInt(100),
				TickSpacing: big.NewInt(10),
				Tick:        big.NewInt(int64(tick)),
			},
			TickStates: []*TickState{
				{Tick: tick, LiquidityNet: big.NewInt(int64(tick))},
			},
		}
		if err := repo.SetPoolState(addr, poolTicks); err != nil {
			t.Fatalf("SetPoolState failed: %v", err)
		}
		ps, err := repo.GetPoolState(addr)
		if err != nil {
			t.Fatalf("GetPoolState failed: %v", err)
		}
		if ps.Global.Tick.Cmp(poolTicks.Global.Tick) != 0 {
			t.Fatalf("GetPoolState: want tick %v, got %v", poolTicks.Global.Tick, ps.Global.Tick)
		}
	}
}

func Test_Close(t *testing.T) {
	repo := newTestRepo(t)
	repo.Close()
}
