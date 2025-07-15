package main

import (
	"path/filepath"
	"testing"
)

type testEntry struct {
	k []byte
	v []byte
}

func (e *testEntry) K() []byte { return e.k }
func (e *testEntry) V() []byte { return e.v }

func TestRocksDB_Basic(t *testing.T) {
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

	// Test Set & Get
	key := []byte("foo")
	val := []byte("bar")
	if err := db.Set(key, val); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	got, err := db.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if string(got) != string(val) {
		t.Errorf("Get value mismatch: got %s, want %s", got, val)
	}

	// Test Del
	if err := db.Del(key); err != nil {
		t.Fatalf("Del failed: %v", err)
	}
	_, err = db.Get(key)
	if err == nil {
		t.Errorf("Get after Del should fail, but got value")
	}

	// Test SetAll
	batch := map[string][]byte{
		"a": []byte("1"),
		"b": []byte("2"),
		"c": []byte("3"),
	}
	if err := db.SetAll(batch); err != nil {
		t.Fatalf("SetAll failed: %v", err)
	}
	for k, v := range batch {
		got, err := db.Get([]byte(k))
		if err != nil || string(got) != string(v) {
			t.Errorf("SetAll/Get mismatch for key %s: got %s, want %s", k, got, v)
		}
	}

	// Test SetAll2
	entries := []Entry{
		&testEntry{k: []byte("x"), v: []byte("100")},
		&testEntry{k: []byte("y"), v: []byte("200")},
	}
	if err := db.SetAll2(entries); err != nil {
		t.Fatalf("SetAll2 failed: %v", err)
	}
	for _, e := range entries {
		got, err := db.Get(e.K())
		if err != nil || string(got) != string(e.V()) {
			t.Errorf("SetAll2/Get mismatch for key %s: got %s, want %s", e.K(), got, e.V())
		}
	}

	// Test GetRange (有序)
	all, err := db.GetRange([]byte("a"), []byte("z"))
	if err != nil {
		t.Fatalf("GetRange failed: %v", err)
	}
	found := map[string]bool{}
	for _, e := range all {
		found[string(e.K())] = true
	}
	for _, k := range []string{"a", "b", "c", "x", "y"} {
		if !found[k] {
			t.Errorf("GetRange missing key: %s", k)
		}
	}
}
