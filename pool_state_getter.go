package main

import (
	"errors"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrPairNotFound = errors.New("no pair info")
	ErrPairFiltered = errors.New("pair is filtered")
	ErrNotV3Pool    = errors.New("not a v3 pool")
)

type PoolStateGetter interface {
	GetPoolState(addr common.Address) (*PoolState, error)
}

type poolStateGetter struct {
	cache          Cache
	db             DB
	contractCaller *ContractCaller
}

func NewPoolStateGetter(cache Cache, db DB, url string) PoolStateGetter {
	contractCaller := NewContractCaller(url)
	return &poolStateGetter{
		cache:          cache,
		db:             db,
		contractCaller: contractCaller,
	}
}

func decoratePoolState(poolState *PoolState, pair *Pair) *PoolState {
	poolState.Token0 = &Token{
		Symbol:   pair.Token0Core.Symbol,
		Decimals: pair.Token0Core.Decimals,
	}
	poolState.Token1 = &Token{
		Symbol:   pair.Token1Core.Symbol,
		Decimals: pair.Token1Core.Decimals,
	}
	if pair.TokensReversed {
		poolState.Token0, poolState.Token1 = poolState.Token1, poolState.Token0
	}

	return poolState
}

func (g *poolStateGetter) GetPoolState(addr common.Address) (*PoolState, error) {
	pair, ok := g.cache.GetPair(addr)
	if !ok {
		return nil, ErrPairNotFound
	}

	if pair.Filtered {
		return nil, ErrPairFiltered
	}

	if pair.ProtocolId != 3 {
		return nil, ErrNotV3Pool
	}

	ok, err := g.db.PoolExists(addr)
	if ok && err == nil {
		poolState, err := g.db.GetPoolState(addr)
		if err != nil {
			return nil, err
		}

		return decoratePoolState(poolState, pair), nil
	}

	poolState, err := g.contractCaller.GetPoolState(addr)
	if err != nil {
		return nil, err
	}

	err = g.db.SetPoolState(addr, poolState)
	if err != nil {
		return nil, err
	}

	return decoratePoolState(poolState, pair), nil
}
