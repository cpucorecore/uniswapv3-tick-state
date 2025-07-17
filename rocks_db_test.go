package main

import (
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

	dbw := NewDBWrap(db)
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

	dbw := NewDBWrap(db)

	testTick := &TickState{
		LiquidityNet: big.NewInt(1),
	}

	key := []byte("test_tick")
	require.NoError(t, dbw.SaveTickState(key, testTick))

	retrievedTick, err := dbw.GetTickState(key)
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

	dbw := NewDBWrap(db)

	k1 := genKey([]byte("mock"), 1)
	k2 := genKey([]byte("mock"), 2)
	k3 := genKey([]byte("mock"), 3)
	k4 := genKey([]byte("mock"), 4)
	tickTests := []struct {
		tickKey         []byte
		tickState       *TickState
		expectTickState *TickState
	}{
		{tickKey: k1, tickState: &TickState{LiquidityNet: big.NewInt(1)}, expectTickState: &TickState{LiquidityNet: big.NewInt(1)}},
		{tickKey: k3, tickState: &TickState{LiquidityNet: big.NewInt(3)}, expectTickState: &TickState{LiquidityNet: big.NewInt(2)}},
		{tickKey: k2, tickState: &TickState{LiquidityNet: big.NewInt(2)}, expectTickState: &TickState{LiquidityNet: big.NewInt(3)}},
	}

	for _, tickTest := range tickTests {
		err = dbw.SaveTickState(tickTest.tickKey, tickTest.tickState)
		require.NoError(t, err)
	}

	ticks, err := dbw.GetTickStates(k1, k4)
	require.NoError(t, err)
	require.Len(t, ticks, 3, "should retrieve 3 ticks")
	for i, tick := range ticks {
		require.True(t, tick.Equal(tickTests[i].expectTickState), "tick state should match")
	}
}
