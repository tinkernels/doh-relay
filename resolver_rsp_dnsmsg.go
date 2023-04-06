package main

import (
	"github.com/miekg/dns"
	"math"
)

type DohDnsMsgResolverQ struct {
	Name string
	Type uint16
}

type DnsMsgResolverRsp struct {
	Status             int
	Truncated          bool
	RecursionDesired   bool
	RecursionAvailable bool
	AuthenticData      bool
	CheckingDisabled   bool
	Question           []DohDnsMsgResolverQ
	Answer             []dns.RR
	Authority          []dns.RR
	Additional         []dns.RR
	UnixTSOfArrival_   int64
}

func (rsp *DnsMsgResolverRsp) StatusV() int {
	return rsp.Status
}

func (rsp *DnsMsgResolverRsp) TruncatedV() bool {
	return rsp.Truncated
}

func (rsp *DnsMsgResolverRsp) RecursionAvailableV() bool {
	return rsp.RecursionAvailable
}

func (rsp *DnsMsgResolverRsp) AuthenticDataV() bool {
	return rsp.AuthenticData
}

func (rsp *DnsMsgResolverRsp) AnswerV() (answer []dns.RR) {
	return rsp.Answer
}

func (rsp *DnsMsgResolverRsp) NsV() (ns []dns.RR) {
	return rsp.Authority
}

func (rsp *DnsMsgResolverRsp) ExtraV() (extra []dns.RR) {
	return rsp.Additional
}

func (rsp *DnsMsgResolverRsp) UnixTSOfArrival() int64 {
	return rsp.UnixTSOfArrival_
}

func (rsp *DnsMsgResolverRsp) ObtainMinimalTTL() (ttl uint32) {
	var minTTLInAnswer uint32 = math.MaxUint32
	for _, r_ := range ConcatSlices(rsp.Answer, rsp.Authority) {
		if r_.Header().Ttl < minTTLInAnswer {
			minTTLInAnswer = r_.Header().Ttl
		}
	}
	if minTTLInAnswer == math.MaxUint32 {
		ttl = 0
	} else {
		ttl = minTTLInAnswer
	}
	return
}
