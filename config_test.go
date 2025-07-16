package main

import (
	"encoding/json"
	"testing"
)

func TestConfig(t *testing.T) {
	bytes, err := json.Marshal(G)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	Log.Info(string(bytes))
}
