package main

type RspCacheItem struct {
	TimeUnixWhenSet  int64
	Ttl              uint32
	ResolverResponse ResolverRsp
}
