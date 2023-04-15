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
		if cacheOptions.cacheType == CacheTypeInternal {
			rsv.cache = NewCacheInternal()
			rsv.cacheType = CacheTypeInternal
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
		log.Infof("new connection to: %+v", dParams_)
		return net.Dial(dParams_[0], dParams_[1])
	}
	pool, err := connpool.NewChannelPool(len(dialParams_)*8, len(dialParams_)*128, factory_)
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
	if rsv.cacheType == CacheTypeInternal {
		return cacheItem_.(*RspCacheItem).ResolverResponse, true
	} else {
		// TODO: redis cache type
		return nil, false
	}
}

func (rsv *Dns53DnsMsgResolver) SetCache(key string, value *RspCacheItem, ttl uint32) {
	rsv.cache.Set(key, value, ttl)
}

// Query Dns over dns53 endpoint.
func (rsv *Dns53DnsMsgResolver) Query(qName string, qType uint16, ecsIPs string) (
	rsp ResolverRsp, err error) {

	return CommonResolverQuery(rsv, qName, qType, ecsIPs)
}

func (rsv *Dns53DnsMsgResolver) Resolve(qName string, qType uint16, ecsIP *net.IP) (
	rsp ResolverRsp, err error) {

	msgReq_ := new(dns.Msg)
	defer func() { msgReq_ = nil }()
	msgReq_.SetQuestion(dns.Fqdn(qName), qType)
	msgReq_.RecursionDesired = true
	if ecsIP != nil {
		ChangeECSInDnsMsg(msgReq_, ecsIP)
	}
	msgRsp_, rtt_ := rsv.doQueryUpstream(msgReq_)
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

func (rsv *Dns53DnsMsgResolver) doQueryUpstream(reqMsg *dns.Msg) (rspMsg *dns.Msg, rtt time.Duration) {
	client_ := new(dns.Client)
	var (
		netCon_ net.Conn
		err     error
	)
	for {
		netCon_, err = rsv.netConnPool.Get(context.Background())
		if err != nil {
			log.Errorf("error: %+v, conn: %+v", err, netCon_)
			continue
		} else {
			rspMsg, rtt, err = client_.ExchangeWithConn(reqMsg, &dns.Conn{Conn: netCon_})
			if err != nil {
				if pc, ok := netCon_.(*connpool.PoolConn); ok {
					pc.MarkUnusable()
					if err = pc.Close(); err != nil {
						log.Error(err)
					}
				}
				continue
			} else if err = netCon_.Close(); err != nil {
				log.Error(err)
			}
			break
		}
	}
	return
}
