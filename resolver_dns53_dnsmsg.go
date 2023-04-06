package main

import (
	"context"
	"github.com/buraksezer/connpool"
	"github.com/miekg/dns"
	"net"
	"net/url"
	"strings"
	"time"
)

var Quad9Dns53Endpoints = []string{
	"tcp://149.112.112.11:53",
	"tcp://9.9.9.11:53",
}

type Dns53DnsMsgResolver struct {
	cache       Cache
	cacheType   string
	useCache    bool
	endpoints   []string
	netConnPool connpool.Pool
}

func NewDns53DnsMsgResolver(endpoints []string, useCache bool, cacheOptions *CacheOptions) (rsv *Dns53DnsMsgResolver) {
	rsv = &Dns53DnsMsgResolver{
		useCache:  useCache,
		endpoints: endpoints,
	}
	rsv.netConnPool = newConnPool4Resolver(endpoints)
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

func newConnPool4Resolver(endpoints []string) (pool connpool.Pool) {
	dialParams_ := make([][]string, 0, len(endpoints))
	for _, edp := range endpoints {
		url_, err := url.Parse(strings.TrimSpace(edp))
		if err != nil {
			panic(err)
		}
		if strings.ToLower(url_.Scheme) != "tcp" || !ListenAddrPortAvailable(url_.Host) {
			log.Errorf("endpoint not usable, should be like tcp://8.8.8.8:53,tcp://8.8.4.4:53")
			continue
		}
		dialParams_ = append(dialParams_, []string{url_.Scheme, url_.Host})
	}
	if len(dialParams_) == 0 {
		panic("endpoint not usable, should be like tcp://8.8.8.8:53,tcp://8.8.4.4:53")
	}
	nextDailParams_ := func() func() []string {
		initV_ := 0
		if len(dialParams_) == 1 {
			return func() []string {
				return dialParams_[0]
			}
		} else {
			return func() []string {
				ret_ := dialParams_[initV_]
				initV_ = (initV_ + 1) % len(dialParams_)
				return ret_
			}
		}
	}()
	factory_ := func() (net.Conn, error) {
		dParams_ := nextDailParams_()
		return net.Dial(dParams_[0], dParams_[1])
	}
	pool, err := connpool.NewChannelPool(len(dialParams_), len(dialParams_)*64, factory_)
	if err != nil {
		panic(err)
	}
	return
}

func (rsv *Dns53DnsMsgResolver) IsUsingCache() bool {
	return rsv.useCache
}

func (rsv *Dns53DnsMsgResolver) GetCache(key string) (rsp ResolverRsp, ok bool) {
	cacheItem_, ok := rsv.cache.Get(key)
	if !ok {
		return nil, false
	}
	if rsv.cacheType == InternalCacheType {
		return cacheItem_.(RspCacheItem).ResolverResponse, true
	} else {
		// TODO: redis cache type
		return nil, false
	}
}

func (rsv *Dns53DnsMsgResolver) SetCache(key string, value RspCacheItem, ttl uint32) {
	rsv.cache.Set(key, value, ttl)
}

// Query Dns over dns53 endpoint.
// If eDnsClientSubnet is empty, will use client ip as eDnsClientSubnet.
func (rsv *Dns53DnsMsgResolver) Query(qName string, qType uint16, eDnsClientSubnet string) (
	rsp ResolverRsp, err error) {

	return CommonResolverQuery(rsv, qName, qType, eDnsClientSubnet)
}

func (rsv *Dns53DnsMsgResolver) Resolve(qName string, qType uint16, eDnsClientSubnet string) (
	rsp ResolverRsp, err error) {

	ecsIP_ := []net.IP{net.ParseIP(DefaultEDnsSubnetIP)}
	ecsGEOCountryCodes_ := []string{DefaultCountry}

	var tmpIPs_ []net.IP
	var tmpGeoCountries_ []string
	ecsIPStrs_ := strings.Split(eDnsClientSubnet, ",")
	for _, s := range ecsIPStrs_ {
		if strings.TrimSpace(s) == "" {
			continue
		}
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

func (rsv *Dns53DnsMsgResolver) queryUpstream(qName string, qType uint16, ecsIP net.IP) (rsp ResolverRsp, err error) {
	msgReq_ := new(dns.Msg)
	msgReq_.SetQuestion(dns.Fqdn(qName), qType)
	msgReq_.RecursionDesired = true
	eDnsSubnetRec_ := new(dns.EDNS0_SUBNET)
	eDnsSubnetRec_.Code = dns.EDNS0SUBNET
	eDnsSubnetRec_.SourceScope = 0
	if ip4_ := ecsIP.To4(); ip4_ != nil {
		eDnsSubnetRec_.Family = 1
		eDnsSubnetRec_.Address = ip4_
		eDnsSubnetRec_.SourceNetmask = 24 // ipv4 mask
	} else {
		eDnsSubnetRec_.Family = 2
		eDnsSubnetRec_.Address = ecsIP.To16()
		eDnsSubnetRec_.SourceNetmask = 56 // ipv6 mask
	}
	opt_ := &dns.OPT{Hdr: dns.RR_Header{
		Name: ".", Rrtype: dns.TypeOPT}, Option: []dns.EDNS0{eDnsSubnetRec_},
	}
	msgReq_.Extra = []dns.RR{opt_}
	client_ := new(dns.Client)
	netCon_, err := rsv.netConnPool.Get(context.Background())
	if err != nil {
		log.Error(err)
		return
	}
	defer func() {
		if errCon_ := netCon_.Close(); errCon_ != nil {
			log.Error(errCon_)
		}
	}()
	msgRsp_, rtt_, err := client_.ExchangeWithConn(msgReq_, &dns.Conn{Conn: netCon_})
	if err != nil {
		log.Error(err)
		if pc, ok := netCon_.(*connpool.PoolConn); ok {
			pc.MarkUnusable()
			if errPCon_ := pc.Close(); errPCon_ != nil {
				log.Error(errPCon_)
			}
		}
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
		log.Infof("got reply to question: %s, %s, %+v", msgRsp_.Question[0].Name,
			dns.TypeToString[msgRsp_.Question[0].Qtype], rtt_)
	}
	log.Tracef("got reply from upstream: %v", msgRsp_.String())
	rsvRsp_.UnixTSOfArrival_ = time.Now().Unix()
	return rsvRsp_, nil
}
