package main

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
	"net/netip"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func PathExists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func ListenAddrPortAvailable(addrPort string) bool {
	if _, err := netip.ParseAddrPort(addrPort); err == nil {
		return true
	}
	// Match :port pattern.
	pattern_, _ := regexp.Compile(`:[1-9][0-9]+`)
	matched_ := pattern_.Match([]byte(addrPort))
	if matched_ {
		port_ := strings.TrimLeft(addrPort, ":")
		if portNum, err := strconv.Atoi(port_); err == nil && portNum > 0 && portNum < 65536 {
			return true
		}
	}
	return false
}

func ObtainIPFromString(ipStr string) net.IP {
	if ip, _, err := net.ParseCIDR(ipStr); err == nil {
		return ip
	} else if ip := net.ParseIP(ipStr); ip != nil {
		return ip
	} else {
		return nil
	}
}

func AdjustDnsMsgTtl(msg *dns.Msg, unixTSOfArrival int64) {
	subtrahend_ := time.Now().Unix() - unixTSOfArrival
	// If arrival recently, don't adjust ttl
	if subtrahend_ < 2 {
		return
	}
	subtrahendUInt32_ := uint32(subtrahend_)
	for i_ := range msg.Answer {
		if msg.Answer[i_].Header().Ttl < subtrahendUInt32_ {
			msg.Answer[i_].Header().Ttl = 0
			continue
		}
		targetTTL_ := msg.Answer[i_].Header().Ttl - subtrahendUInt32_
		msg.Answer[i_].Header().Ttl = targetTTL_
	}
	for i_ := range msg.Ns {
		if msg.Ns[i_].Header().Ttl < subtrahendUInt32_ {
			msg.Ns[i_].Header().Ttl = 0
			continue
		}
		targetTTL_ := msg.Ns[i_].Header().Ttl - subtrahendUInt32_
		msg.Ns[i_].Header().Ttl = targetTTL_
	}
	for i_ := range msg.Extra {
		if msg.Extra[i_].Header().Ttl < subtrahendUInt32_ {
			msg.Extra[i_].Header().Ttl = 0
			continue
		}
		targetTTL_ := msg.Extra[i_].Header().Ttl - subtrahendUInt32_
		msg.Extra[i_].Header().Ttl = targetTTL_
	}
}

func CommonResolverQuery(rsv Resolver, qName string, qType uint16, eDnsClientSubnet string) (
	rsp ResolverRsp, err error) {

	geoLocName_ := DefaultCountry
	if ip_ := ObtainIPFromString(eDnsClientSubnet); ip_ != nil {
		if loc_ := GeoIPLocName(ip_); loc_ != "" {
			geoLocName_ = loc_
		}
	}
	log.Debugf("eDnsClientSubnet geoip location: %s", geoLocName_)
	cacheKey_ := fmt.Sprintf("NAME[%s]TYPE[%d]LOC[%s]", qName, qType, geoLocName_)
	if rsv.IsUsingCache() {
		if c_, ok_ := rsv.GetCache(cacheKey_); ok_ {
			log.Infof("got cache for: %s %s", qName, dns.TypeToString[qType])
			rsp, err = c_, nil
			return
		}
	}
	rsp, err = rsv.Resolve(qName, qType, eDnsClientSubnet)
	if rsv.IsUsingCache() {
		if err != nil || rsp == nil {
			log.Errorf("err: %v, reply: %v", err, rsp)
		} else {
			ttl_ := rsp.ObtainMinimalTTL()
			if ttl_ > 1 {
				cacheTtl := ttl_
				// max cache ttl
				if cacheTtl > 3600 {
					cacheTtl = 3600
				}
				rsv.SetCache(cacheKey_,
					&RspCacheItem{
						ResolverResponse: rsp,
						TimeUnixWhenSet:  time.Now().Unix(),
						Ttl:              ttl_,
					},
					cacheTtl,
				)
			}
		}
	}
	return
}
