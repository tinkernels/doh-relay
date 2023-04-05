package main

import (
	"github.com/miekg/dns"
	"math"
	"time"
)

type DnsMsgResolverQ struct {
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
	Question           []DnsMsgResolverQ
	Answer             []dns.RR
	Authority          []dns.RR
	Additional         []dns.RR
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

func (rsp *DnsMsgResolverRsp) ObtainMinimalTTL() (ttl time.Duration) {
	var initTTL uint32 = 15 // Initial minimal TTL a reasonable value.
	var minTTLInAnswer uint32 = 0
	for _, r_ := range rsp.Answer {
		if minTTLInAnswer == 0 {
			minTTLInAnswer = r_.Header().Ttl
		} else if r_.Header().Ttl < minTTLInAnswer {
			minTTLInAnswer = r_.Header().Ttl
		}
	}
	ttl = time.Duration(math.Max(float64(initTTL), float64(minTTLInAnswer)))
	return ttl * time.Second
}

func (rsp *DnsMsgResolverRsp) SubtractTTL(minus uint32) DohResolverRsp {
	for i_ := range rsp.Answer {
		targetTTL_ := rsp.Answer[i_].Header().Ttl - minus
		if targetTTL_ < 0 {
			targetTTL_ = 1 // At least 1 second.
		}
		rsp.Answer[i_].Header().Ttl = targetTTL_
	}
	for i_ := range rsp.Authority {
		targetTTL_ := rsp.Authority[i_].Header().Ttl - minus
		if targetTTL_ < 0 {
			targetTTL_ = 1 // At least 1 second.
		}
		rsp.Authority[i_].Header().Ttl = targetTTL_
	}
	for i_ := range rsp.Additional {
		targetTTL_ := rsp.Additional[i_].Header().Ttl - minus
		if targetTTL_ < 0 {
			targetTTL_ = 1 // At least 1 second.
		}
		rsp.Additional[i_].Header().Ttl = targetTTL_
	}
	return rsp
}
