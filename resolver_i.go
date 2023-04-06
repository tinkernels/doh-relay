package main

import "net"

type Resolver interface {
	Query(qName string, qType uint16, eDnsClientSubnets string) (rsp ResolverRsp, err error)
	Resolve(qName string, qType uint16, ip *net.IP) (rsp ResolverRsp, err error)
	IsUsingCache() bool
	GetCache(string) (rsp ResolverRsp, ok bool)
	SetCache(string, *RspCacheItem, uint32)
}
