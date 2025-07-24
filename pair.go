package main

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"time"
)

func IsSameAddress(address1, address2 common.Address) bool {
	return address1.Cmp(address2) == 0
}

type TokenCore struct {
	Address  common.Address `json:"-"`
	Symbol   string
	Decimals int8
}

func (t *TokenCore) MarshalJSON() ([]byte, error) {
	type Alias TokenCore
	return json.Marshal(&struct {
		AddressString string `json:"Address"`
		*Alias
	}{
		AddressString: t.Address.String(),
		Alias:         (*Alias)(t),
	})
}

func (t *TokenCore) UnmarshalJSON(data []byte) error {
	type Alias TokenCore
	aux := &struct {
		AddressString string `json:"Address"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.Address = common.HexToAddress(aux.AddressString)
	return nil
}

func (t *TokenCore) Equal(token *TokenCore) bool {
	if !IsSameAddress(t.Address, token.Address) {
		return false
	}

	if t.Symbol != token.Symbol {
		return false
	}

	if t.Decimals != token.Decimals {
		return false
	}

	return true
}

type Pair struct {
	Address          common.Address `json:"-"`
	TokensReversed   bool
	Token0Core       *TokenCore
	Token1Core       *TokenCore
	Token0InitAmount decimal.Decimal
	Token1InitAmount decimal.Decimal
	Block            uint64
	BlockAt          time.Time
	ProtocolId       int
	Filtered         bool
	FilterCode       int
	Timestamp        time.Time
}

func (p *Pair) MarshalBinary() ([]byte, error) {
	type Alias Pair
	return json.Marshal(&struct {
		AddressString string `json:"Address"`
		*Alias
	}{
		AddressString: p.Address.String(),
		Alias:         (*Alias)(p),
	})
}

func (p *Pair) UnmarshalBinary(data []byte) error {
	type Alias Pair
	aux := &struct {
		AddressString string `json:"Address"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.Address = common.HexToAddress(aux.AddressString)
	return nil
}
