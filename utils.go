package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

func SliceContains[T string | int | uint | int8 | int16 | int32 | int64 | uint8 | uint16 | uint32 | uint64 |
	float32 | float64](s []T, e T) bool {

	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func ConcatSlices[T any](first []T, second []T) []T {
	n := len(first)
	return append(first[:n:n], second...)
}

func RemoveSliceDuplicate[T comparable](sliceList []T) (list []T) {
	allKeys_ := make(map[T]bool)
	defer func() { allKeys_ = nil }()
	list = make([]T, 0, len(sliceList))
	for _, item := range sliceList {
		if _, found := allKeys_[item]; !found {
			allKeys_[item] = true
			list = append(list, item)
		}
	}
	return
}

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
	if matched_ := pattern_.Match([]byte(addrPort)); matched_ {
		port_ := strings.TrimLeft(addrPort, ":")
		if portNum, err := strconv.Atoi(port_); err == nil && portNum > 0 && portNum < 65536 {
			return true
		}
	}
	return false
}

func ObtainIPFromString(ipStr string) net.IP {
	trimmedIPStr_ := strings.TrimSpace(ipStr)
	if ip, _, err := net.ParseCIDR(trimmedIPStr_); err == nil {
		return ip
	} else if ip := net.ParseIP(trimmedIPStr_); ip != nil {
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

func CommonResolverQuery(rsv Resolver, qName string, qType uint16, ecsIPsStr string) (
	rsp ResolverRsp, err error) {

	cacheKey_ := fmt.Sprintf("NAME[%s]TYPE[%d]", qName, qType)

	var (
		ips_             []net.IP
		countryCodes_    []string
		countryStateArr_ []string
	)
	ecsIPStrArr_ := strings.Split(ecsIPsStr, ",")
	for _, s := range ecsIPStrArr_ {
		if ip_ := ObtainIPFromString(s); ip_ != nil {
			country_, state_, _ := GeoIPCountryStateCity(ip_)
			if SliceContains(countryCodes_, country_) {
				continue
			}
			ips_ = append(ips_, ip_)
			countryCodes_ = append(countryCodes_, country_)
			countryStateArr_ = append(countryStateArr_, fmt.Sprintf("%s,%s", country_, state_))
		}
	}
	if len(countryStateArr_) != 0 {
		cacheKey_ = fmt.Sprintf("%sLOC[%s]", cacheKey_, strings.Join(countryStateArr_, "|"))
	}
	if rsv.IsUsingCache() {
		if rsp, ok := rsv.GetCache(cacheKey_); ok {
			log.Infof("got cache for: %s %s, cache-key: %s", qName, dns.TypeToString[qType], cacheKey_)
			return rsp, nil
		}
	}
	rsp, err = resolveWithECSIPs(rsv, qName, qType, ips_, countryCodes_)
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

func resolveWithECSIPs(rsv Resolver, qName string, qType uint16, ecsIPs []net.IP, ecsCountryCodes []string) (
	rsp ResolverRsp, err error) {

	if len(ecsIPs) == 0 || (qType != dns.TypeA && qType != dns.TypeAAAA) {
		return rsv.Resolve(qName, qType, nil)
	}

	type Result struct {
		Rsp ResolverRsp
		Ok  bool
		Err error
	}

	// Create a channel to receive the results of each goroutine.
	resultChanArr_, resultChanArrClosed_ := make([]chan *Result, len(ecsIPs)), false
	for i := range resultChanArr_ {
		resultChanArr_[i] = make(chan *Result)
	}

	// Launch a goroutine for each IP address for A, AAAA query.
	for i, ip := range ecsIPs {
		go func(ip net.IP, countryCode string, resultChan chan *Result) {
			r, err := rsv.Resolve(qName, qType, &ip)
			if err == nil {
				// Check if the response matches the expected country code.
				switch qType {
				case dns.TypeA:
					for _, rr_ := range r.AnswerV() {
						if rrA_, ok := rr_.(*dns.A); ok {
							if c, _, _ := GeoIPCountryStateCity(rrA_.A); c == countryCode {
								if !resultChanArrClosed_ {
									resultChan <- &Result{Rsp: r, Ok: true}
								}
								return
							}
						}
					}
					if !resultChanArrClosed_ {
						resultChan <- &Result{Rsp: r, Ok: false}
					}
				case dns.TypeAAAA:
					for _, rr_ := range r.AnswerV() {
						if rrAAAA_, ok := rr_.(*dns.AAAA); ok {
							if c, _, _ := GeoIPCountryStateCity(rrAAAA_.AAAA); c == countryCode {
								if !resultChanArrClosed_ {
									resultChan <- &Result{Rsp: r, Ok: true}
								}
								return
							}
						}
					}
					if !resultChanArrClosed_ {
						resultChan <- &Result{Rsp: r, Ok: false}
					}
				}
			} else {
				if !resultChanArrClosed_ {
					resultChan <- &Result{Ok: false, Err: err}
				}
			}
		}(ip, ecsCountryCodes[i], resultChanArr_[i])
	}

	// Wait for all the results to come in.
	var lastResult_ ResolverRsp
	for i := 0; i < len(ecsIPs); i++ {
		r := <-resultChanArr_[i]
		rsp, ok, err := r.Rsp, r.Ok, r.Err
		lastResult_ = rsp
		if err != nil {
			log.Error(err)
			continue
		} else if !ok {
			continue
		} else {
			// close the channel to indicate that the goroutine is done.
			go func() {
				resultChanArrClosed_ = true
				for _, c := range resultChanArr_ {
					select {
					case <-c:
					default:
						close(c)
					}
				}
			}()
			return lastResult_, nil
		}
	}

	if lastResult_ != nil {
		return lastResult_, nil
	} else {
		// If all resolves are failed, return nil
		return nil, fmt.Errorf("no result for %s %s", qName, dns.TypeToString[qType])
	}
}

func ObtainECS(msg *dns.Msg) (ecs *dns.EDNS0_SUBNET) {
	var eDns0 = msg.IsEdns0()
	if eDns0 != nil {
		for _, o := range eDns0.Option {
			switch o.(type) {
			case *dns.EDNS0_SUBNET:
				ecs = o.(*dns.EDNS0_SUBNET)
				return
			}
		}
	}
	return
}

func RemoveECSInDnsMsg(msg *dns.Msg) {
	recEdns0_ := msg.IsEdns0()
	// Replace existing.
	if recEdns0_ != nil {
		ecsIdx_ := -1
		for i, o := range recEdns0_.Option {
			switch o.(type) {
			case *dns.EDNS0_SUBNET:
				ecsIdx_ = i
				break
			}
		}
		if ecsIdx_ >= 0 {
			recEdns0_.Option = append(recEdns0_.Option[:ecsIdx_], recEdns0_.Option[ecsIdx_+1:]...)
		}
	}
}

func ChangeECSInDnsMsg(msg *dns.Msg, ip *net.IP) {
	eDnsSubnetRec_ := new(dns.EDNS0_SUBNET)
	eDnsSubnetRec_.Code = dns.EDNS0SUBNET
	eDnsSubnetRec_.SourceScope = 0

	if ip4_ := ip.To4(); ip4_ != nil {
		eDnsSubnetRec_.Family = 1
		eDnsSubnetRec_.Address = ip4_
		eDnsSubnetRec_.SourceNetmask = 24 // ipv4 mask
	} else {
		eDnsSubnetRec_.Family = 2
		eDnsSubnetRec_.Address = ip.To16()
		eDnsSubnetRec_.SourceNetmask = 56 // ipv6 mask
	}

	recEdns0_ := msg.IsEdns0()
	// Replace existing.
	if recEdns0_ != nil {
		for i, o := range recEdns0_.Option {
			switch o.(type) {
			case *dns.EDNS0_SUBNET:
				recEdns0_.Option[i] = eDnsSubnetRec_
				return
			}
		}
		recEdns0_.Option = append([]dns.EDNS0{eDnsSubnetRec_}, recEdns0_.Option...)
	} else {
		// Add new EDNS0 record
		opt_ := &dns.OPT{Hdr: dns.RR_Header{
			Name: ".", Rrtype: dns.TypeOPT}, Option: []dns.EDNS0{eDnsSubnetRec_},
		}
		msg.Extra = []dns.RR{opt_}
	}
}

type CheckIPApiRsp struct {
	IP      string `json:"ip"`
	Address string `json:"address"`
}

var (
	checkIPEndpoints = map[string]string{
		"https://wq.apnic.net/ip":                                   "json",
		"https://accountws.arin.net/public/seam/resource/rest/myip": "json",
		"https://www.ripe.net/@@ipaddress":                          "plain_text",
		"https://rdap.lacnic.net/rdap/info/myip":                    "json",
		"https://checkip.amazonaws.com":                             "plain_text",
	}
)

func GetIPAnswerFromResolverRsp(rsvRsp ResolverRsp) (ipStr string) {
	for _, r := range rsvRsp.AnswerV() {
		switch r.(type) {
		case *dns.A:
			{
				if ipA := r.(*dns.A).A; ipA != nil {
					return ipA.String()
				}
			}
		case *dns.AAAA:
			{
				if ipAAAA := r.(*dns.AAAA).AAAA; ipAAAA != nil {
					return ipAAAA.String()
				}
			}
		}
	}
	return
}

func GetExitIPByResolver(rsv Resolver) (ipStr string) {
	for edp, typ := range checkIPEndpoints {
		url_, _ := url.Parse(edp)
		hostname_ := url_.Hostname()
		rsvRspA_, errA_ := rsv.Resolve(hostname_, dns.TypeA, nil)
		rsvRspAAAA_, errAAAA_ := rsv.Resolve(hostname_, dns.TypeAAAA, nil)
		if errA_ != nil || errAAAA_ != nil {
			continue
		}
		ips_ := []string{GetIPAnswerFromResolverRsp(rsvRspA_), GetIPAnswerFromResolverRsp(rsvRspAAAA_)}
	getIPApiLoop:
		for _, ip := range ips_ {
			if ip == "" {
				continue getIPApiLoop
			}
			tlsConfig_ := &tls.Config{
				ServerName: hostname_,
			}
			timeout_ := time.Second * 9
			dialer_ := net.Dialer{Timeout: timeout_}
			httpClientTr_ := &http.Transport{
				Proxy:           nil,
				TLSClientConfig: tlsConfig_,
				DialContext: func(ctx context.Context, network string, addr string) (net.Conn, error) {
					_, port_, _ := net.SplitHostPort(addr)
					addrReplaced := net.JoinHostPort(ip, port_)
					log.Infof("check ip api dailing: network[%s], addr[%s], previous addr[%s]",
						network, addrReplaced, addr)
					return dialer_.DialContext(ctx, network, addrReplaced)
				},
			}
			client_ := &http.Client{Transport: httpClientTr_, Timeout: timeout_}
		getIPApiTypeSwitch:
			switch typ {
			case "json":
				{
					if rsp_, err := client_.Get(edp); err == nil {
						if rspBytes_, err := io.ReadAll(rsp_.Body); err == nil {
							var apiRspJson CheckIPApiRsp
							err := json.Unmarshal(rspBytes_, &apiRspJson)
							if err != nil {
								continue getIPApiLoop
							}
							ipRet_, ipRetAlt_ := apiRspJson.IP, apiRspJson.Address
							if ipRet_ != "" {
								return ipRet_
							} else if ipRetAlt_ != "" {
								return ipRetAlt_
							} else {
								continue getIPApiLoop
							}
						}
					}
					break getIPApiTypeSwitch
				}
			default:
				{
					if rsp_, err := client_.Get(edp); err == nil {
						if rspBytes_, err := io.ReadAll(rsp_.Body); err == nil {
							plainText_ := string(rspBytes_[:])
							if ipTest_ := net.ParseIP(strings.TrimSpace(plainText_)); ipTest_ != nil {
								return ipTest_.String()
							}
						}
					}
					break getIPApiTypeSwitch
				}
			}
		}
	}
	return
}

func HTTPGetString(urlStr string) (string, error) {
	resp_, err_ := http.Get(urlStr)
	if err_ != nil {
		return "", err_
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp_.Body)
	if resp_.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http get error: %s", resp_.Status)
	}
	bodyBytes_, err_ := io.ReadAll(resp_.Body)
	if err_ != nil {
		return "", err_
	}
	return string(bodyBytes_[:]), nil
}
