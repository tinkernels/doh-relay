package main

import (
	"github.com/miekg/dns"
	"math"
	"time"
)

var DefaultEDnsSubnetIP = "113.105.171.123"
var DefaultCountry = "CN"

type JsonResolverQ struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
}

type JsonResolverRR struct {
	Name string `json:"name,omitempty"`
	Type uint16 `json:"type,omitempty"`
	TTL  uint32 `json:"TTL,omitempty"`
	Data string `json:"data,omitempty"`
}

type JsonResolverRsp struct {
	Status             int              `json:"Status"`
	Truncated          bool             `json:"TC"`
	RecursionDesired   bool             `json:"RD"`
	RecursionAvailable bool             `json:"RA"`
	AuthenticData      bool             `json:"AD"`
	CheckingDisabled   bool             `json:"CD"`
	Question           []JsonResolverQ  `json:"Question,omitempty"`
	Answer             []JsonResolverRR `json:"Answer,omitempty"`
	Authority          []JsonResolverRR `json:"Authority,omitempty"`
	Additional         []JsonResolverRR `json:"Additional,omitempty"`
	EDNSClientSubnet   string           `json:"edns_client_subnet,omitempty"`
	Comment            string           `json:"Comment,omitempty"`
}

func (rsp *JsonResolverRsp) StatusV() int {
	return rsp.Status
}

func (rsp *JsonResolverRsp) TruncatedV() bool {
	return rsp.Truncated
}

func (rsp *JsonResolverRsp) RecursionAvailableV() bool {
	return rsp.RecursionAvailable
}

func (rsp *JsonResolverRsp) AuthenticDataV() bool {
	return rsp.AuthenticData
}

func (rsp *JsonResolverRsp) AnswerV() (answer []dns.RR) {
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

func (rsp *JsonResolverRsp) NsV() (ns []dns.RR) {
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

func (rsp *JsonResolverRsp) ExtraV() (extra []dns.RR) {
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

func (rsp *JsonResolverRsp) ObtainMinimalTTL() (ttl time.Duration) {
	var initTTL uint32 = 15 // Initial minimal TTL a reasonable value.
	var minTTLInAnswer uint32 = 0
	for _, r_ := range rsp.Answer {
		if minTTLInAnswer == 0 {
			minTTLInAnswer = r_.TTL
		} else if r_.TTL < minTTLInAnswer {
			minTTLInAnswer = r_.TTL
		}
	}
	ttl = time.Duration(math.Max(float64(initTTL), float64(minTTLInAnswer)))
	return ttl * time.Second
}

func (rsp *JsonResolverRsp) SubtractTTL(minus uint32) DohResolverRsp {
	for i_ := range rsp.Answer {
		targetTTL_ := rsp.Answer[i_].TTL - minus
		if targetTTL_ < 0 {
			targetTTL_ = 1 // At least 1 second.
		}
		rsp.Answer[i_].TTL = targetTTL_
	}
	for i_ := range rsp.Authority {
		targetTTL_ := rsp.Authority[i_].TTL - minus
		if targetTTL_ < 0 {
			targetTTL_ = 1 // At least 1 second.
		}
		rsp.Authority[i_].TTL = targetTTL_
	}
	for i_ := range rsp.Additional {
		targetTTL_ := rsp.Additional[i_].TTL - minus
		if targetTTL_ < 0 {
			targetTTL_ = 1 // At least 1 second.
		}
		rsp.Additional[i_].TTL = targetTTL_
	}
	return rsp
}

// RR transforms a JsonResolverRR to a dns.RR
func (r JsonResolverRR) RR() (dns.RR, error) {
	hdr := dns.RR_Header{Name: r.Name, Rrtype: r.Type, Class: dns.ClassINET, Ttl: r.TTL}
	str := hdr.String() + r.Data
	return dns.NewRR(str)
}
