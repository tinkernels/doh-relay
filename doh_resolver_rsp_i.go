package main

import (
	"github.com/miekg/dns"
	"time"
)

type DohResolverRsp interface {
	StatusV() int
	TruncatedV() bool
	RecursionAvailableV() bool
	AuthenticDataV() bool
	AnswerV() []dns.RR
	NsV() []dns.RR
	ExtraV() []dns.RR
	ObtainMinimalTTL() time.Duration
	SubtractTTL(minus uint32) DohResolverRsp
}
