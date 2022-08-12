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

func CommonResolverQuery(rsv DohResolver, qName string, qType uint16, eDnsClientSubnet string) (
	rsp DohResolverRsp, err error) {

	geoIPCountry_ := DefaultCountry
	if ip_ := ObtainIPFromString(eDnsClientSubnet); ip_ != nil {
		if country_ := GeoipCountry(ip_); country_ != "" {
			geoIPCountry_ = country_
		}
	}
	log.Debugf("eDnsClientSubnet geoip country: %s", geoIPCountry_)
	cacheKey_ := fmt.Sprintf("NAME[%s]TYPE[%d]LOC[%s]", qName, qType, geoIPCountry_)
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

			// error response, set cache for 9 seconds.
			rsv.SetCache(cacheKey_, &DohCacheItem{
				DohResolverResponse: rsp,
				SetTimeUnix:         time.Now().Unix()},
				9*time.Second)
		} else {
			rsv.SetCache(cacheKey_, &DohCacheItem{
				DohResolverResponse: rsp,
				SetTimeUnix:         time.Now().Unix()},
				rsp.ObtainMinimalTTL())
		}
	}
	return
}
