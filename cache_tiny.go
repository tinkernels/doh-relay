package main

import (
	"github.com/ReneKroon/ttlcache"
	"time"
)

type DohCacheItem struct {
	SetTimeUnix         int64
	DohResolverResponse DohResolverRsp
	GeoCountry          string
}

func GetDohCache(cache *ttlcache.Cache, key string) (rsp DohResolverRsp, ok bool) {
	if cache == nil {
		return nil, false
	}
	var cacheItem_ *DohCacheItem
	if c_, ok_ := cache.Get(key); c_ != nil && ok_ {
		cacheItem_ = c_.(*DohCacheItem)
	} else {
		return nil, false
	}
	if cacheItem_.DohResolverResponse == nil {
		return nil, true
	}
	ttl2Subtract_ := time.Now().Unix() - cacheItem_.SetTimeUnix
	if ttl2Subtract_ < 0 {
		cache.Remove(key)
		return nil, false
	}

	rsp, ok = cacheItem_.DohResolverResponse.SubtractTTL(uint32(ttl2Subtract_)), true
	cacheItem_.SetTimeUnix = time.Now().Unix()
	return
}
