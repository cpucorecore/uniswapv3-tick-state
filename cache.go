package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
	"time"
)

type PairCache interface {
	GetPair(address common.Address) (*Pair, bool)
}

type Cache interface {
	PairCache
}

type twoTierCache struct {
	ctx    context.Context
	memory *cache.Cache
	redis  *redis.Client
}

func NewTwoTierCache(redis *redis.Client) Cache {
	return &twoTierCache{
		ctx:    context.Background(),
		memory: cache.New(time.Hour*24, time.Hour),
		redis:  redis,
	}
}

func PairCacheKey(address common.Address) string {
	return fmt.Sprintf("npr:%s", address.Hex())
}

func (c *twoTierCache) GetPair(address common.Address) (*Pair, bool) {
	k := PairCacheKey(address)
	pair, ok := c.memory.Get(k)
	if ok {
		return pair.(*Pair), true
	}

	v := &Pair{}
	err := c.redis.Get(c.ctx, k).Scan(v)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			Log.Error("redis get err", zap.Error(err))
		}
		return nil, false
	}

	c.memory.Set(k, v, 0)
	return v, true
}
