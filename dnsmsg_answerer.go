package main

import (
	"fmt"
	"github.com/miekg/dns"
)

type DnsMsgAnswerer struct {
	Resolver DohResolver
}

func NewDnsMsgAnswerer(rsv DohResolver) (dma *DnsMsgAnswerer) {
	return &DnsMsgAnswerer{
		Resolver: rsv,
	}
}

func (dma *DnsMsgAnswerer) Answer(dnsReq *dns.Msg, eDnsClientSubnet string) (dnsRsp *dns.Msg, err error) {
	var question_ dns.Question
	if len(dnsReq.Question) > 0 {
		question_ = dnsReq.Question[0]
	} else {
		return nil, fmt.Errorf("no question in request")
	}

	rsvRsp_, err := dma.Resolver.Query(question_.Name, question_.Qtype, eDnsClientSubnet)
	if err != nil || rsvRsp_ == nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	dnsRsp = new(dns.Msg)
	dnsRsp.SetReply(dnsReq)
	dnsRsp.Rcode = rsvRsp_.StatusV()
	dnsRsp.Truncated = rsvRsp_.TruncatedV()
	dnsRsp.RecursionAvailable = rsvRsp_.RecursionAvailableV()
	dnsRsp.AuthenticatedData = rsvRsp_.AuthenticDataV()
	dnsRsp.Answer = rsvRsp_.AnswerV()
	dnsRsp.Ns = rsvRsp_.NsV()
	dnsRsp.Extra = rsvRsp_.ExtraV()
	return
}
