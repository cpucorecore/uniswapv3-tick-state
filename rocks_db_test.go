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

	dbw := NewRepo(db)
	height, err := dbw.GetHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(0), height)

	testHeight := uint64(1000)
	require.NoError(t, dbw.SetHeight(testHeight))
	height, err = dbw.GetHeight()
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

	dbw := NewRepo(db)

	testTick := &TickState{
		LiquidityNet: big.NewInt(1),
	}

	addr := common.HexToAddress("0xffff")
	tick := int32(0)
	require.NoError(t, dbw.SetTickState(addr, tick, testTick))

	retrievedTick, err := dbw.GetTickState(addr, tick)
	require.NoError(t, err)
	require.True(t, retrievedTick.Equal(testTick), "retrieved tick should be equal to the original tick")
}

func TestGetTicks(t *testing.T) {
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

	r := NewRepo(db)

	addr := common.HexToAddress("0xffff")
	tn1 := int32(-1)
	tn2 := int32(-2)
	t1 := int32(1)
	t2 := int32(2)
	t3 := int32(3)

	tickTests := []struct {
		tick            int32
		tickState       *TickState
		expectTickState *TickState
	}{
		{tick: t1, tickState: &TickState{LiquidityNet: big.NewInt(1)}, expectTickState: &TickState{LiquidityNet: big.NewInt(-2)}},
		{tick: t3, tickState: &TickState{LiquidityNet: big.NewInt(3)}, expectTickState: &TickState{LiquidityNet: big.NewInt(-1)}},
		{tick: t2, tickState: &TickState{LiquidityNet: big.NewInt(2)}, expectTickState: &TickState{LiquidityNet: big.NewInt(1)}},
		{tick: tn2, tickState: &TickState{LiquidityNet: big.NewInt(-2)}, expectTickState: &TickState{LiquidityNet: big.NewInt(2)}},
		{tick: tn1, tickState: &TickState{LiquidityNet: big.NewInt(-1)}, expectTickState: &TickState{LiquidityNet: big.NewInt(3)}},
	}

	for _, tickTest := range tickTests {
		err = r.SetTickState(addr, tickTest.tick, tickTest.tickState)
		require.NoError(t, err)
	}

	ticks, err := r.GetTickStates(addr, tn2, t3)
	require.NoError(t, err)
	require.Len(t, ticks, 5, "should retrieve 5 ticks")
	for i, tick := range ticks {
		require.True(t, tick.Equal(tickTests[i].expectTickState), "tick state should match")
	}
}
