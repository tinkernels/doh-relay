package main

import (
	"github.com/miekg/dns"
	"math"
)

var DefaultEDnsSubnetIP = "113.105.171.123"
var DefaultCountry = "CN"

type DohJsonResolverQ struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
}

type DohJsonResolverRR struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
	TTL  uint32 `json:"TTL,omitempty"`
	Data string `json:"data,omitempty"`
}

type DohJsonResolverRsp struct {
	Status             int                 `json:"Status"`
	Truncated          bool                `json:"TC"`
	RecursionDesired   bool                `json:"RD"`
	RecursionAvailable bool                `json:"RA"`
	AuthenticData      bool                `json:"AD"`
	CheckingDisabled   bool                `json:"CD"`
	Question           []DohJsonResolverQ  `json:"Question,omitempty"`
	Answer             []DohJsonResolverRR `json:"Answer,omitempty"`
	Authority          []DohJsonResolverRR `json:"Authority,omitempty"`
	Additional         []DohJsonResolverRR `json:"Additional,omitempty"`
	EDNSClientSubnet   string              `json:"edns_client_subnet,omitempty"`
	Comment            string              `json:"Comment,omitempty"`
	UnixTSOfArrival_   int64
}

func (rsp *DohJsonResolverRsp) StatusV() int {
	return rsp.Status
}

func (rsp *DohJsonResolverRsp) TruncatedV() bool {
	return rsp.Truncated
}

func (rsp *DohJsonResolverRsp) RecursionAvailableV() bool {
	return rsp.RecursionAvailable
}

func (rsp *DohJsonResolverRsp) AuthenticDataV() bool {
	return rsp.AuthenticData
}

func (rsp *DohJsonResolverRsp) AnswerV() (answer []dns.RR) {
	for _, r_ := range rsp.Answer {
		rr, err := r_.RR()
		if err != nil {
			log.Warnf("Failed to parse RR: %s", err)
			return nil
		}
		answer = append(answer, rr)
	}
	return
}

func (rsp *DohJsonResolverRsp) NsV() (ns []dns.RR) {
	ns = make([]dns.RR, len(rsp.Authority))
	for i_, r_ := range rsp.Authority {
		rr, err := r_.RR()
		if err != nil {
			log.Warnf("Failed to parse RR: %s", err)
			return nil
		}
		ns[i_] = rr
	}
	return
}

func (rsp *DohJsonResolverRsp) ExtraV() (extra []dns.RR) {
	extra = make([]dns.RR, len(rsp.Additional))
	for i_, r_ := range rsp.Additional {
		rr, err := r_.RR()
		if err != nil {
			log.Warnf("Failed to parse RR: %s", err)
			return nil
		}
		extra[i_] = rr
	}
	return
}

func (rsp *DohJsonResolverRsp) UnixTSOfArrival() int64 {
	return rsp.UnixTSOfArrival_
}

func (rsp *DohJsonResolverRsp) ObtainMinimalTTL() (ttl uint32) {
	var minTTLInAnswer uint32 = math.MaxUint32
	for _, r_ := range rsp.Answer {
		if r_.TTL < minTTLInAnswer {
			minTTLInAnswer = r_.TTL
		}
	}
	for _, r_ := range rsp.Authority {
		if r_.TTL < minTTLInAnswer {
			minTTLInAnswer = r_.TTL
		}
	}
	if minTTLInAnswer == math.MaxUint32 {
		ttl = 0
	} else {
		ttl = minTTLInAnswer
	}
	return
}

// RR transforms a DohJsonResolverRR to a dns.RR
func (r DohJsonResolverRR) RR() (dns.RR, error) {
	hdr := dns.RR_Header{Name: r.Name, Rrtype: r.Type, Class: dns.ClassINET, Ttl: r.TTL}
	str := hdr.String() + r.Data
	return dns.NewRR(str)
}
