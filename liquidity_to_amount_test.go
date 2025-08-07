package main

import (
	"math/big"
	"testing"
)

func TestCalcRangeAmount_Basic(t *testing.T) {
	liquidity := big.NewInt(1000000)
	tickLower := int32(0)
	tickUpper := int32(100)
	token0Decimals := 18
	token1Decimals := 18

	rangeAmount := CalcRangeAmount(liquidity, tickLower, tickUpper, token0Decimals, token1Decimals)
	if rangeAmount.Amount0.Cmp(big.NewFloat(0)) <= 0 {
		t.Errorf("Amount0 should be positive, got %s", rangeAmount.Amount0.String())
	}
	if rangeAmount.Amount1.Cmp(big.NewFloat(0)) <= 0 {
		t.Errorf("Amount1 should be positive, got %s", rangeAmount.Amount1.String())
	}
}

func TestCalcRangeAmountArray_Empty(t *testing.T) {
	arr := CalcRangeAmountArray(nil, 18, 18)
	if arr != nil && len(arr) != 0 {
		t.Errorf("Expected empty array, got %v", arr)
	}
}

func TestCalcRangeAmountArray_Single(t *testing.T) {
	rangeLiquidity := &RangeLiquidity{
		TickLower: 0,
		TickUpper: 100,
		Liquidity: big.NewInt(1000000),
	}
	arr := CalcRangeAmountArray([]*RangeLiquidity{rangeLiquidity}, 18, 18)
	if len(arr) != 1 {
		t.Fatalf("Expected 1 element, got %d", len(arr))
	}
	if arr[0].Amount0.Cmp(big.NewFloat(0)) <= 0 {
		t.Errorf("Amount0 should be positive, got %s", arr[0].Amount0.String())
	}
	if arr[0].Amount1.Cmp(big.NewFloat(0)) <= 0 {
		t.Errorf("Amount1 should be positive, got %s", arr[0].Amount1.String())
	}
}
