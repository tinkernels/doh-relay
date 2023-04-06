package main

import (
	"github.com/miekg/dns"
)

type ResolverRsp interface {
	StatusV() int
	TruncatedV() bool
	RecursionAvailableV() bool
	AuthenticDataV() bool
	AnswerV() []dns.RR
	NsV() []dns.RR
	ExtraV() []dns.RR
	ObtainMinimalTTL() uint32
	UnixTSOfArrival() int64
}
