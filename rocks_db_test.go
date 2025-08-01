package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"math/big"
	"path/filepath"
	"testing"
)

type testEntry struct {
	k []byte
	v []byte
}

func (e *testEntry) K() []byte { return e.k }
func (e *testEntry) V() []byte { return e.v }

func TestGetSetHeight(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	opts := &RocksDBOptions{
		BlockCacheSize:       8 * 1024 * 1024, // 8MB
		WriteBufferSize:      4 * 1024 * 1024, // 4MB
		MaxWriteBufferNumber: 2,
	}
	db, err := NewRocksDB(dbPath, opts)
	if err != nil {
		t.Fatalf("failed to create RocksDB: %v", err)
	}
	defer db.Close()

	dbw := NewDB(db)
	height, err := dbw.GetFinishHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(0), height)

	testHeight := uint64(1000)
	require.NoError(t, dbw.SetFinishHeight(testHeight))
	height, err = dbw.GetFinishHeight()
	require.NoError(t, err)
	require.Equal(t, testHeight, height)
}

func TestGetSetTick(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "testdb")
	opts := &RocksDBOptions{
		BlockCacheSize:       8 * 1024 * 1024, // 8MB
		WriteBufferSize:      4 * 1024 * 1024, // 4MB
		MaxWriteBufferNumber: 2,
	}
	db, err := NewRocksDB(dbPath, opts)
	if err != nil {
		t.Fatalf("failed to create RocksDB: %v", err)
	}
	defer db.Close()

	dbw := NewDB(db)

	testTick := &TickState{
		LiquidityNet: big.NewInt(1),
	}

	addr := common.HexToAddress("0xffff")
	tick := int32(0)
	require.NoError(t, dbw.SetTickState(addr, testTick))

	retrievedTick, err := dbw.GetTickState(addr, tick)
	require.NoError(t, err)
	require.True(t, retrievedTick.Equal(testTick), "retrieved tick should be equal to the original tick")
}

func TestGetTicks(t *testing.T) {
	opts := &RocksDBOptions{
		BlockCacheSize:       8 * 1024 * 1024,
		WriteBufferSize:      4 * 1024 * 1024,
		MaxWriteBufferNumber: 2,
	}
	name := t.TempDir()
	db, err := NewRocksDB(name, opts)
	if err != nil {
		t.Fatalf("failed to create RocksDB: %v", err)
	}
	defer db.Close()

	r := NewDB(db)

	addr := common.HexToAddress("0xffff")
	tn1 := int32(-1)
	tn2 := int32(-2)
	t1 := int32(1)
	t2 := int32(2)
	t3 := int32(3)

	tickTests := []struct {
		tickState       *TickState
		expectTickState *TickState
	}{
		{tickState: &TickState{Tick: t1, LiquidityNet: big.NewInt(100)}, expectTickState: &TickState{Tick: tn2, LiquidityNet: big.NewInt(-200)}},
		{tickState: &TickState{Tick: t3, LiquidityNet: big.NewInt(300)}, expectTickState: &TickState{Tick: tn1, LiquidityNet: big.NewInt(-100)}},
		{tickState: &TickState{Tick: t2, LiquidityNet: big.NewInt(200)}, expectTickState: &TickState{Tick: t1, LiquidityNet: big.NewInt(100)}},
		{tickState: &TickState{Tick: tn2, LiquidityNet: big.NewInt(-200)}, expectTickState: &TickState{Tick: t2, LiquidityNet: big.NewInt(200)}},
		{tickState: &TickState{Tick: tn1, LiquidityNet: big.NewInt(-100)}, expectTickState: &TickState{Tick: t3, LiquidityNet: big.NewInt(300)}},
	}

	for _, tickTest := range tickTests {
		err = r.SetTickState(addr, tickTest.tickState)
		require.NoError(t, err)
	}

	ticks, err := r.GetTickStates(addr, tn2, t3)
	require.NoError(t, err)
	require.Len(t, ticks, 5, "should retrieve 5 ticks")
	for i, tick := range ticks {
		require.True(t, tick.Equal(tickTests[i].expectTickState), "tick state should match")
	}
}
