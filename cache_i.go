package main

var (
	InternalCacheType = "internal"
	RedisCacheType    = "redis"
)

type CacheOptions struct {
	cacheType string
	redisURI  string
}

type Cache interface {
	Get(key string) (val interface{}, ok bool)
	Set(key string, val interface{}, ttl uint32)
}
