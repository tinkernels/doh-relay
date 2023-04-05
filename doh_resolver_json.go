package main

import (
    "encoding/json"
    "fmt"
    "github.com/ReneKroon/ttlcache"
    "github.com/gojek/heimdall/v7"
    "github.com/gojek/heimdall/v7/hystrix"
    "github.com/miekg/dns"
    "github.com/quic-go/quic-go/http3"
    "net"
    "net/http"
    "strings"
    "time"
)

var Quad9JsonEndpoints = []string{
    "https://149.112.112.11:5053/dns-query",
    "https://9.9.9.11:5053/dns-query",
}

type JsonResolver struct {
    httpClient   *hystrix.Client
    cache        *ttlcache.Cache
    useCache     bool
    endpoints    []string
    nextEndpoint func() string
}

func NewJsonResolver(endpoints []string, useCache bool, ifHttp3 bool) (rsv *JsonResolver) {
    httpClient_ := &http.Client{}
    if ifHttp3 {
        httpClient_.Transport = &http3.RoundTripper{}
    }
    rsv = &JsonResolver{
        httpClient: hystrix.NewClient(
            hystrix.WithHTTPClient(httpClient_),
            hystrix.WithHTTPTimeout(9*time.Second),
            hystrix.WithHystrixTimeout(15*time.Second),
            hystrix.WithMaxConcurrentRequests(HttpClientMaxConcurrency),
            hystrix.WithRequestVolumeThreshold(40),
            hystrix.WithErrorPercentThreshold(50),
            hystrix.WithSleepWindow(8),
            hystrix.WithRetryCount(5),
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

func (rsv *JsonResolver) IsUsingCache() bool {
    return rsv.useCache
}

func (rsv *JsonResolver) GetCache(key string) (rsp DohResolverRsp, ok bool) {
    return GetDohCache(rsv.cache, key)
}

func (rsv *JsonResolver) SetCache(key string, value *DohCacheItem, ttl time.Duration) {
    rsv.cache.SetWithTTL(key, value, ttl)
}

// Query Dns over HTTPS endpoint.
// If eDnsClientSubnet is empty, will use client ip as eDnsClientSubnet.
func (rsv *JsonResolver) Query(qName string, qType uint16, eDnsClientSubnet string) (
    rsp DohResolverRsp, err error) {

    return CommonResolverQuery(rsv, qName, qType, eDnsClientSubnet)
}

func (rsv *JsonResolver) Resolve(qName string, qType uint16, eDnsClientSubnet string) (
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

func (rsv *JsonResolver) queryUpstream(qName string, qType uint16, ecsIP net.IP) (rsp DohResolverRsp, err error) {

    httpRsp_, err := rsv.httpClient.Get(
        fmt.Sprintf("%s?name=%s&type=%d&do=1&edns_client_subnet=%s&random_padding=%d",
            rsv.nextEndpoint(), qName, qType, ecsIP.String(), time.Now().Nanosecond()),
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
    jsonRsp_ := new(JsonResolverRsp)
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
    return jsonRsp_, nil
}
