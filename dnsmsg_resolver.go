package main

import (
	"encoding/base64"
	"fmt"
	"github.com/ReneKroon/ttlcache"
	"github.com/gojek/heimdall/v7"
	"github.com/gojek/heimdall/v7/hystrix"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/miekg/dns"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var Quad9DnsMsgEndpoints = []string{
	"https://149.112.112.11/dns-query",
	"https://9.9.9.11/dns-query",
}

type DnsMsgResolver struct {
	httpClient   *hystrix.Client
	cache        *ttlcache.Cache
	useCache     bool
	endpoints    []string
	nextEndpoint func() string
}

func NewDnsMsgResolver(endpoints []string, useCache bool, ifHttp3 bool) (rsv *DnsMsgResolver) {
	httpClient_ := &http.Client{}
	if ifHttp3 {
		httpClient_.Transport = &http3.RoundTripper{}
	}
	rsv = &DnsMsgResolver{
		httpClient: hystrix.NewClient(
			hystrix.WithHTTPClient(httpClient_),
			hystrix.WithHTTPTimeout(9*time.Second),
			hystrix.WithHystrixTimeout(15*time.Second),
			hystrix.WithMaxConcurrentRequests(HttpClientMaxConcurrency),
			hystrix.WithRequestVolumeThreshold(40),
			hystrix.WithErrorPercentThreshold(50),
			hystrix.WithSleepWindow(8),
			hystrix.WithRetryCount(4),
			hystrix.WithRetrier(heimdall.NewRetrier(heimdall.NewExponentialBackoff(
				time.Millisecond*50, time.Second*1, 1.8, time.Millisecond*20,
			))),
		),
		useCache:  useCache,
		endpoints: endpoints,
	}
	rsv.nextEndpoint = func() func() string {
		initV_ := 0
		if len(rsv.endpoints) == 1 {
			return func() string {
				return rsv.endpoints[0]
			}
		} else {
			return func() string {
				ret_ := rsv.endpoints[initV_]
				initV_ = (initV_ + 1) % len(rsv.endpoints)
				return ret_
			}
		}
	}()
	// If using specified upstream endpoints.
	if rsv.useCache {
		rsv.cache = ttlcache.NewCache()
		rsv.cache.SkipTtlExtensionOnHit(true)
	}
	return
}

func (rsv *DnsMsgResolver) IsUsingCache() bool {
	return rsv.useCache
}

func (rsv *DnsMsgResolver) GetCache(key string) (rsp DohResolverRsp, ok bool) {
	return GetDohCache(rsv.cache, key)
}

func (rsv *DnsMsgResolver) SetCache(key string, value *DohCacheItem, ttl time.Duration) {
	rsv.cache.SetWithTTL(key, value, ttl)
}

// Query Dns over HTTPS endpoint.
// If eDnsClientSubnet is empty, will use client ip as eDnsClientSubnet.
func (rsv *DnsMsgResolver) Query(qName string, qType uint16, eDnsClientSubnet string) (
	rsp DohResolverRsp, err error) {

	return CommonResolverQuery(rsv, qName, qType, eDnsClientSubnet)
}

func (rsv *DnsMsgResolver) Resolve(qName string, qType uint16, eDnsClientSubnet string) (
	rsp DohResolverRsp, err error) {

	ecsIP_ := []net.IP{net.ParseIP(DefaultEDnsSubnetIP)}
	ecsGEOCountryCodes_ := []string{DefaultCountry}

	var tmpIPs_ []net.IP
	var tmpGeoCountries_ []string
	ecsIPStrs_ := strings.Split(eDnsClientSubnet, ",")
	for _, s := range ecsIPStrs_ {
		if ip_ := ObtainIPFromString(s); ip_ != nil && GeoipCountry(ip_) != "" {
			tmpIPs_ = append(tmpIPs_, ip_)
			tmpGeoCountries_ = append(tmpGeoCountries_, GeoipCountry(ip_))
		}
	}
	if len(tmpIPs_) > 0 && len(tmpIPs_) == len(tmpGeoCountries_) {
		ecsIP_ = tmpIPs_
		ecsGEOCountryCodes_ = tmpGeoCountries_
	}

ipGEOLoop:
	for i, ip := range ecsIP_ {
		rsp, err = rsv.queryUpstream(qName, qType, ip)
		if err != nil {
			continue
		}
		switch qType {
		case dns.TypeA:
			{
				for _, r := range rsp.AnswerV() {
					switch r.(type) {
					case *dns.A:
						{
							if ipA := r.(*dns.A).A; ipA != nil &&
								GeoipCountry(ipA) == ecsGEOCountryCodes_[i] {
								break ipGEOLoop
							}
						}
					}
				}
				break
			}
		case dns.TypeAAAA:
			{
				for _, r := range rsp.AnswerV() {
					switch r.(type) {
					case *dns.AAAA:
						if ipAAAA := r.(*dns.AAAA).AAAA; ipAAAA != nil &&
							GeoipCountry(ipAAAA) == ecsGEOCountryCodes_[i] {
							break ipGEOLoop
						}
					}
				}
				break
			}
		default:
			break ipGEOLoop
		}
	}
	return
}

func (rsv *DnsMsgResolver) queryUpstream(qName string, qType uint16, ecsIP net.IP) (rsp DohResolverRsp, err error) {
	msgReq_ := new(dns.Msg)
	msgReq_.SetQuestion(dns.Fqdn(qName), qType)
	msgReq_.RecursionDesired = true
	eDnsSubnetRec_ := new(dns.EDNS0_SUBNET)
	eDnsSubnetRec_.Code = dns.EDNS0SUBNET
	eDnsSubnetRec_.SourceScope = 0
	if ip4_ := ecsIP.To4(); ip4_ != nil {
		eDnsSubnetRec_.Family = 1
		eDnsSubnetRec_.Address = ip4_
		eDnsSubnetRec_.SourceNetmask = net.IPv4len * 8
	} else {
		eDnsSubnetRec_.Family = 2
		eDnsSubnetRec_.Address = ecsIP.To16()
		eDnsSubnetRec_.SourceNetmask = net.IPv6len * 8
	}
	opt_ := &dns.OPT{Hdr: dns.RR_Header{
		Name: ".", Rrtype: dns.TypeOPT}, Option: []dns.EDNS0{eDnsSubnetRec_},
	}
	msgReq_.Extra = []dns.RR{opt_}
	msgBytes_, err := msgReq_.Pack()
	if err != nil {
		log.Error(err)
		return
	}
	msgBase64_ := base64.RawURLEncoding.EncodeToString(msgBytes_)
	httpRsp_, err := rsv.httpClient.Get(
		fmt.Sprintf("%s?dns=%s", rsv.nextEndpoint(), msgBase64_),
		http.Header{"Accept": []string{"application/dns-message"}},
	)
	defer func() {
		if httpRsp_ != nil && httpRsp_.Body != nil {
			_ = httpRsp_.Body.Close()
		}
	}()
	if err != nil {
		log.Error(err)
		return
	}
	buf_, err := io.ReadAll(httpRsp_.Body)
	if err != nil {
		log.Error(err)
		return
	}
	msgRsp_ := new(dns.Msg)
	err = msgRsp_.Unpack(buf_)
	if err != nil {
		log.Error(err)
		return
	}
	rsvRsp_ := &DnsMsgResolverRsp{
		Status:             msgRsp_.Rcode,
		Truncated:          msgRsp_.Truncated,
		RecursionDesired:   msgRsp_.RecursionDesired,
		RecursionAvailable: msgRsp_.RecursionAvailable,
		AuthenticData:      msgRsp_.AuthenticatedData,
		CheckingDisabled:   msgRsp_.CheckingDisabled,
	}
	rsvRsp_.Question = make([]DnsMsgResolverQ, len(msgRsp_.Question))
	for i, q := range msgRsp_.Question {
		rsvRsp_.Question[i] = DnsMsgResolverQ{
			Name: q.Name,
			Type: q.Qtype,
		}
	}
	rsvRsp_.Answer = msgRsp_.Answer
	rsvRsp_.Authority = msgRsp_.Ns
	rsvRsp_.Additional = msgRsp_.Extra
	if len(msgRsp_.Question) > 0 {
		log.Infof("got reply to question: %s, %s [%s]", msgRsp_.Question[0].Name,
			dns.TypeToString[msgRsp_.Question[0].Qtype], msgBase64_)
	}
	log.Tracef("got reply from upstream: %v", msgRsp_.String())
	return rsvRsp_, nil
}
