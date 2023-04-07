package main

import (
	"encoding/json"
	"fmt"
	"github.com/gojek/heimdall/v7/hystrix"
	"github.com/miekg/dns"
	"net"
	"net/http"
	"time"
)

var Quad9JsonEndpoints = []string{
	"https://149.112.112.11:5053/dns-query",
	"https://9.9.9.11:5053/dns-query",
}

type DohJsonResolver struct {
	httpClient   *hystrix.Client
	cache        Cache
	cacheType    string
	useCache     bool
	endpoints    []string
	nextEndpoint func() string
}

func NewDohJsonResolver(endpoints []string, useCache bool, cacheOptions *CacheOptions) (rsv *DohJsonResolver) {
	httpClient_ := &http.Client{}
	rsv = &DohJsonResolver{
		httpClient: hystrix.NewClient(
			hystrix.WithHTTPClient(httpClient_),
			hystrix.WithHTTPTimeout(9*time.Second),
			hystrix.WithHystrixTimeout(15*time.Second),
			hystrix.WithMaxConcurrentRequests(HttpClientMaxConcurrency),
			hystrix.WithRequestVolumeThreshold(40),
			hystrix.WithErrorPercentThreshold(50),
			hystrix.WithRetryCount(0),
			//hystrix.WithRetrier(heimdall.NewRetrier(heimdall.NewExponentialBackoff(
			//    time.Millisecond*50, time.Second*1, 1.8, time.Millisecond*20,
			//))),
			//hystrix.WithSleepWindow(8),
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

func (rsv *DohJsonResolver) IsUsingCache() bool {
	return rsv.useCache
}

func (rsv *DohJsonResolver) GetCache(key string) (rsp ResolverRsp, ok bool) {
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

func (rsv *DohJsonResolver) SetCache(key string, value *RspCacheItem, ttl uint32) {
	rsv.cache.Set(key, value, ttl)
}

// Query Dns over HTTPS endpoint.
func (rsv *DohJsonResolver) Query(qName string, qType uint16, ecsIPs string) (
	rsp ResolverRsp, err error) {

	return CommonResolverQuery(rsv, qName, qType, ecsIPs)
}

func (rsv *DohJsonResolver) Resolve(qName string, qType uint16, ecsIP *net.IP) (
	rsp ResolverRsp, err error) {

	ecsP_ := fmt.Sprintf("")
	if ecsIP != nil {
		ecsP_ = fmt.Sprintf("&edns_client_subnet=%s", ecsIP.String())
	}
	qStr_ := fmt.Sprintf("%s?name=%s&type=%d&do=1%s&random_padding=%d",
		rsv.nextEndpoint(), qName, qType, ecsP_, time.Now().Nanosecond())
	httpRsp_, err := rsv.httpClient.Get(
		qStr_,
		http.Header{"Accept": []string{"application/x-javascript,application/json"}},
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
	jsonRsp_ := new(DohJsonResolverRsp)
	decoder_ := json.NewDecoder(httpRsp_.Body)
	err = decoder_.Decode(&jsonRsp_)
	if err != nil {
		log.Error(err)
		return
	}
	if jsonRsp_.Status != 0 {
		log.Warnf("response status is not 0: %+v", rsp)
	} else {
		log.Tracef("json response: %+v", jsonRsp_)
	}
	if len(jsonRsp_.Question) == 1 {
		log.Infof("got reply to question: %s %s", jsonRsp_.Question[0].Name,
			dns.TypeToString[jsonRsp_.Question[0].Type])
	}
	for _, r_ := range jsonRsp_.Answer {
		if rr_, err := r_.RR(); err != nil {
			log.Error(err)
			return nil, err
		} else {
			log.Debugf("answer record: %s", rr_)
		}
	}
	for _, r_ := range jsonRsp_.Authority {
		if rr_, err := r_.RR(); err != nil {
			log.Error(err)
		} else {
			log.Debugf("authority record: %s", rr_)
		}
	}
	for _, r_ := range jsonRsp_.Additional {
		if rr_, err := r_.RR(); err != nil {
			log.Error(err)
		} else {
			log.Debugf("extra record: %s", rr_)
		}
	}
	jsonRsp_.UnixTSOfArrival_ = time.Now().Unix()
	return jsonRsp_, nil
}