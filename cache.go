package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-redis/redis/v8"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

type PairCache interface {
	GetPair(addr common.Address) (*Pair, bool)
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

func PairCacheKey(addr common.Address) string {
	return fmt.Sprintf("npr:%s", addr.Hex())
}

func (c *twoTierCache) GetPair(addr common.Address) (*Pair, bool) {
	k := PairCacheKey(addr)
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
