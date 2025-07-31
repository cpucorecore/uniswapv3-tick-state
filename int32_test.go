package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInt32ToBytes(t *testing.T) {
	v := int32(-1000)
	bytes := int32ToBytes(v)
	require.Equal(t, bytesToInt32(bytes), v)
}
