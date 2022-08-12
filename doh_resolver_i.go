package main

import "time"

type DohResolver interface {
	Query(qName string, qType uint16, eDnsClientSubnet string) (rsp DohResolverRsp, err error)
	Resolve(qName string, qType uint16, eDnsClientSubnet string) (rsp DohResolverRsp, err error)
	IsUsingCache() bool
	GetCache(string) (rsp DohResolverRsp, ok bool)
	SetCache(string, *DohCacheItem, time.Duration)
}
