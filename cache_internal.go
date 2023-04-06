package main

import (
	"github.com/ReneKroon/ttlcache"
	"time"
)

type CacheInternal struct {
	cacher *ttlcache.Cache
}

func NewCacheInternal() (cache *CacheInternal) {
	cacher := ttlcache.NewCache()
	cacher.SkipTtlExtensionOnHit(true)
	cache = &CacheInternal{cacher: cacher}
	return
}

func (cache *CacheInternal) Get(key string) (val interface{}, ok bool) {
	val, ok = cache.cacher.Get(key)
	return
}

func (cache *CacheInternal) Set(key string, val interface{}, ttl uint32) {
	cache.cacher.SetWithTTL(key, val, time.Second*time.Duration(ttl))
}
