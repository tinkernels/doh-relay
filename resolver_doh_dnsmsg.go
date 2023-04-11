package main

import (
	"encoding/base64"
	"fmt"
	"github.com/gojek/heimdall/v7/hystrix"
	"github.com/miekg/dns"
	"io"
	"net"
	"net/http"
	"time"
)

var Quad9DnsMsgEndpoints = []string{
	"https://149.112.112.11/dns-query",
	"https://9.9.9.11/dns-query",
}

type DohDnsMsgResolver struct {
	httpClient   *hystrix.Client
	cache        Cache
	cacheType    string
	useCache     bool
	endpoints    []string
	nextEndpoint func() string
}

func NewDohDnsMsgResolver(endpoints []string, useCache bool, cacheOptions *CacheOptions) (rsv *DohDnsMsgResolver) {
	httpClient_ := &http.Client{
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	rsv = &DohDnsMsgResolver{
		httpClient: hystrix.NewClient(
			hystrix.WithHTTPClient(httpClient_),
			hystrix.WithHTTPTimeout(15*time.Second),
			hystrix.WithHystrixTimeout(20*time.Second),
			hystrix.WithMaxConcurrentRequests(HttpClientMaxConcurrency),
			hystrix.WithRequestVolumeThreshold(HttpClientMaxConcurrency),
			hystrix.WithErrorPercentThreshold(50),
			hystrix.WithSleepWindow(1),
			hystrix.WithRetryCount(0),
			//hystrix.WithRetrier(heimdall.NewRetrier(heimdall.NewExponentialBackoff(
			//    time.Millisecond*50, time.Second*1, 1.8, time.Millisecond*20,
			//))),
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
	// If using cache
	if rsv.useCache {
		if cacheOptions.cacheType == InternalCacheType {
			rsv.cache = NewCacheInternal()
			rsv.cacheType = InternalCacheType
		}
		// TODO: redis cache type
	}
	return
}

func (rsv *DohDnsMsgResolver) IsUsingCache() bool {
	return rsv.useCache
}

func (rsv *DohDnsMsgResolver) GetCache(key string) (rsp ResolverRsp, ok bool) {
	cacheItem_, ok := rsv.cache.Get(key)
	if !ok {
		return nil, false
	}
	if rsv.cacheType == InternalCacheType {
		return cacheItem_.(*RspCacheItem).ResolverResponse, true
	} else {
		// TODO: redis cache type
		return nil, false
	}
}

func (rsv *DohDnsMsgResolver) SetCache(key string, value *RspCacheItem, ttl uint32) {
	rsv.cache.Set(key, value, ttl)
}

// Query Dns over HTTPS endpoint.
func (rsv *DohDnsMsgResolver) Query(qName string, qType uint16, ecsIPs string) (
	rsp ResolverRsp, err error) {

	return CommonResolverQuery(rsv, qName, qType, ecsIPs)
}

func (rsv *DohDnsMsgResolver) Resolve(qName string, qType uint16, ecsIP *net.IP) (
	rsp ResolverRsp, err error) {

	msgReq_ := new(dns.Msg)
	defer func() { msgReq_ = nil }()
	msgReq_.SetQuestion(dns.Fqdn(qName), qType)
	msgReq_.RecursionDesired = true
	if ecsIP != nil {
		ChangeECSInDnsMsg(msgReq_, ecsIP)
	}
	msgBytes_, err := msgReq_.Pack()
	defer func() { msgBytes_ = nil }()
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
	rsvRsp_.Question = make([]DohDnsMsgResolverQ, len(msgRsp_.Question))
	for i, q := range msgRsp_.Question {
		rsvRsp_.Question[i] = DohDnsMsgResolverQ{
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
	rsvRsp_.UnixTSOfArrival_ = time.Now().Unix()
	return rsvRsp_, nil
}
