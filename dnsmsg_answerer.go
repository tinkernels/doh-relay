package main

import (
	"fmt"
	"github.com/miekg/dns"
)

type DnsMsgAnswerer struct {
	Resolver         Resolver
	FallbackResolver Resolver
}

func NewDnsMsgAnswerer(rsv, fallback Resolver) (dma *DnsMsgAnswerer) {
	return &DnsMsgAnswerer{
		Resolver:         rsv,
		FallbackResolver: fallback,
	}
}

func (dma *DnsMsgAnswerer) Answer(dnsReq *dns.Msg, ecsIPs string) (dnsRsp *dns.Msg, err error) {
	var question_ dns.Question
	if len(dnsReq.Question) > 0 {
		question_ = dnsReq.Question[0]
	} else {
		return nil, fmt.Errorf("no question in request")
	}
	rsvRsp_, err := dma.Resolver.Query(question_.Name, question_.Qtype, ecsIPs)
	if err != nil || rsvRsp_ == nil {
		return nil, fmt.Errorf("query error: %v", err)
	}

	if len(rsvRsp_.AnswerV()) == 0 && dma.FallbackResolver != nil {
		log.Infof("using fallback resolver for %+v", question_)
		rsvRspFb_, errFb_ := dma.FallbackResolver.Query(question_.Name, question_.Qtype, ecsIPs)
		if errFb_ == nil && rsvRspFb_ != nil && len(rsvRspFb_.AnswerV()) != 0 {
			rsvRsp_, err = rsvRspFb_, errFb_
		}
	}

	tmpDnsRsp_ := new(dns.Msg)
	defer func() { tmpDnsRsp_ = nil }()
	tmpDnsRsp_.SetReply(dnsReq)
	tmpDnsRsp_.Truncated = rsvRsp_.TruncatedV()
	tmpDnsRsp_.RecursionAvailable = rsvRsp_.RecursionAvailableV()
	tmpDnsRsp_.AuthenticatedData = rsvRsp_.AuthenticDataV()
	tmpDnsRsp_.Answer = rsvRsp_.AnswerV()
	tmpDnsRsp_.Ns = rsvRsp_.NsV()
	tmpDnsRsp_.Extra = rsvRsp_.ExtraV()
	dnsRsp = tmpDnsRsp_.Copy()
	AdjustDnsMsgTtl(dnsRsp, rsvRsp_.UnixTSOfArrival())
	return
}
