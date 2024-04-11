package main

import (
	"fmt"
	"github.com/miekg/dns"
	"regexp"
)

type DnsMsgAnswerer struct {
	Resolver         Resolver
	FallbackResolver Resolver
	FixedResolvers   map[*regexp.Regexp]Resolver
}

func NewDnsMsgAnswerer(rsv, fallback Resolver, fixedResolvers map[*regexp.Regexp]Resolver) (dma *DnsMsgAnswerer) {
	return &DnsMsgAnswerer{
		Resolver:         rsv,
		FallbackResolver: fallback,
		FixedResolvers:   fixedResolvers,
	}
}

func (dma *DnsMsgAnswerer) Answer(dnsReq *dns.Msg, ecsIPs string) (dnsRsp *dns.Msg, err error) {
	var question_ dns.Question
	if len(dnsReq.Question) > 0 {
		question_ = dnsReq.Question[0]
	} else {
		return nil, fmt.Errorf("no question in request")
	}

	usingFixedResolver := false
	var rsvRsp_ ResolverRsp
	for n, r := range dma.FixedResolvers {
		if n.Match([]byte(question_.Name)) {
			rsvRsp_, err = r.Query(question_.Name, question_.Qtype, "")
			if err != nil || rsvRsp_ == nil {
				return nil, fmt.Errorf("query error: %v", err)
			}
			usingFixedResolver = true
		}
	}
	if !usingFixedResolver {
		rsvRsp_, err = dma.Resolver.Query(question_.Name, question_.Qtype, ecsIPs)
		if err != nil || rsvRsp_ == nil {
			if dma.FallbackResolver != nil {
				log.Infof("using fallback resolver for %+v", question_)
				rsvRspFb_, errFb_ := dma.FallbackResolver.Query(question_.Name, question_.Qtype, ecsIPs)
				if errFb_ == nil && rsvRspFb_ != nil {
					rsvRsp_, err = rsvRspFb_, errFb_
				} else {
					rsvRsp_, err = nil, fmt.Errorf("query error: %v", rsvRspFb_)
				}
			}
		}
	}

	if err != nil {
		log.Errorf("error in query: %+v", err)
		return
	}

	if rsvRsp_ == nil {
		log.Errorf("nil response from resolver")
		return
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
