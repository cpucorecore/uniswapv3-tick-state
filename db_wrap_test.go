package main

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func newTestRepo(t *testing.T) Repo {
	db, err := NewRocksDB("/tmp/testdb_wrap_rocksdb", &RocksDBOptions{
		EnableLog:            false,
		BlockCacheSize:       1024 * 1024 * 100,
		WriteBufferSize:      1024 * 1024 * 10,
		MaxWriteBufferNumber: 1024 * 1024 * 10,
	})
	if err != nil {
		t.Fatalf("failed to create rocksdb: %v", err)
	}
	return NewRepo(db)
}

func Test_SetTickState_GetTickState_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x1000000000000000000000000000000000000001")
	for _, tick := range []int32{123, -123} {
		ts := &TickState{Tick: tick, LiquidityNet: big.NewInt(123456)}
		if err := repo.SetTickState(addr, tick, ts); err != nil {
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

func Test_GetTickStates_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x2000000000000000000000000000000000000002")
	for _, tick := range []int32{-200, -100, 0, 100, 200} {
		ts := &TickState{Tick: tick, LiquidityNet: big.NewInt(int64(tick))}
		_ = repo.SetTickState(addr, tick, ts)
	}
	states, err := repo.GetTickStates(addr, 0, 201)
	if err != nil || len(states) == 0 {
		t.Fatalf("GetTickStates (pos) failed or empty")
	}
	states, err = repo.GetTickStates(addr, -201, 0)
	if err != nil || len(states) == 0 {
		t.Fatalf("GetTickStates (neg) failed or empty")
	}
}

func Test_GetAllTicks(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x3000000000000000000000000000000000000003")
	_ = repo.SetTickState(addr, 1, &TickState{Tick: 1, LiquidityNet: big.NewInt(1)})
	_ = repo.SetTickState(addr, -1, &TickState{Tick: -1, LiquidityNet: big.NewInt(-1)})
	all, err := repo.GetAllTicks()
	if err != nil || len(all) == 0 {
		t.Fatalf("GetAllTicks failed or empty")
	}
}

func Test_GetPoolTicks_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x4000000000000000000000000000000000000004")
	_ = repo.SetTickState(addr, 10, &TickState{Tick: 10, LiquidityNet: big.NewInt(10)})
	_ = repo.SetTickState(addr, -10, &TickState{Tick: -10, LiquidityNet: big.NewInt(-10)})
	states, err := repo.GetPoolTicks(addr)
	if err != nil || len(states) != 2 {
		t.Fatalf("GetPoolTicks failed or empty")
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

func Test_PoolExists_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x7000000000000000000000000000000000000007")
	_ = repo.SetTickSpacing(addr, 10)
	ok, err := repo.PoolExists(addr)
	if err != nil || !ok {
		t.Fatalf("PoolExists failed or should exist")
	}
	addr2 := common.HexToAddress("0x7000000000000000000000000000000000000008")
	ok, err = repo.PoolExists(addr2)
	if err != nil {
		t.Fatalf("PoolExists (not exist) failed: %v", err)
	}
	if ok {
		t.Fatalf("PoolExists: should not exist")
	}
}

func Test_SetGetPoolHeight(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x8000000000000000000000000000000000000008")
	for _, h := range []uint64{123456, 0} {
		if err := repo.SetPoolHeight(addr, h); err != nil {
			t.Fatalf("SetPoolHeight failed: %v", err)
		}
		h2, err := repo.GetPoolHeight(addr)
		if err != nil {
			t.Fatalf("GetPoolHeight failed: %v", err)
		}
		if h != h2 {
			t.Fatalf("GetPoolHeight: want %d, got %d", h, h2)
		}
	}
}

func Test_SetGetHeight(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	for _, h := range []uint64{987654321, 0} {
		if err := repo.SetHeight(h); err != nil {
			t.Fatalf("SetHeight failed: %v", err)
		}
		h2, err := repo.GetHeight()
		if err != nil {
			t.Fatalf("GetHeight failed: %v", err)
		}
		if h != h2 {
			t.Fatalf("GetHeight: want %d, got %d", h, h2)
		}
	}
}

func Test_SetGetPoolState_PositiveNegative(t *testing.T) {
	repo := newTestRepo(t)
	defer repo.Close()
	addr := common.HexToAddress("0x9000000000000000000000000000000000000009")
	for _, tick := range []int32{100, -100} {
		poolTicks := &PoolTicks{
			State: &PoolState{
				Height:      big.NewInt(100),
				TickSpacing: big.NewInt(10),
				Tick:        big.NewInt(int64(tick)),
			},
			Ticks: []*TickState{
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
		if ps.State.Tick.Cmp(poolTicks.State.Tick) != 0 {
			t.Fatalf("GetPoolState: want tick %v, got %v", poolTicks.State.Tick, ps.State.Tick)
		}
	}
}

func Test_Close(t *testing.T) {
	repo := newTestRepo(t)
	repo.Close()
	repo.Close() // 再次关闭不应panic
}
