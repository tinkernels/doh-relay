package main

import (
	"fmt"
	"github.com/miekg/dns"
)

type DnsMsgAnswerer struct {
	Resolver Resolver
}

func NewDnsMsgAnswerer(rsv Resolver) (dma *DnsMsgAnswerer) {
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
	var (
		waitCh_ = make(chan bool)
		rsvRsp_ ResolverRsp
	)
	go func() {
		rsvRsp_, err = dma.Resolver.Query(question_.Name, question_.Qtype, eDnsClientSubnet)
		waitCh_ <- true
	}()
	<-waitCh_

	if err != nil || rsvRsp_ == nil {
		return nil, fmt.Errorf("query error: %v", err)
	}
	dnsRsp = new(dns.Msg)
	dnsRsp.SetReply(dnsReq)
	dnsRsp.Truncated = rsvRsp_.TruncatedV()
	dnsRsp.RecursionAvailable = rsvRsp_.RecursionAvailableV()
	dnsRsp.AuthenticatedData = rsvRsp_.AuthenticDataV()
	dnsRsp.Answer = rsvRsp_.AnswerV()
	dnsRsp.Ns = rsvRsp_.NsV()
	dnsRsp.Extra = rsvRsp_.ExtraV()
	// Repack msg for adjust ttl.
	rspPack_, err := dnsRsp.Pack()
	if err != nil {
		return nil, err
	}
	err = dnsRsp.Unpack(rspPack_)
	if err != nil {
		return nil, err
	}
	AdjustDnsMsgTtl(dnsRsp, rsvRsp_.UnixTSOfArrival())
	return
}
