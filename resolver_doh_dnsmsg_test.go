package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/miekg/dns"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func Test_DnsMsgBase64(t *testing.T) {
	type args struct {
		qName string
		qType uint16
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "t.tt",
			args: args{
				qType: dns.TypeA,
				qName: "t.tt",
			},
		},
		{
			name: "baidu.com",
			args: args{
				qType: dns.TypeAAAA,
				qName: "baidu.com",
			},
		},
		{
			name: "mi.com",
			args: args{
				qType: dns.TypeSOA,
				qName: "mi.com",
			},
		},
		{
			name: "google.com",
			args: args{
				qType: dns.TypeNS,
				qName: "google.com",
			},
		},
		{
			name: "facebook.com",
			args: args{
				qType: dns.TypeMX,
				qName: "facebook.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgReq_ := new(dns.Msg)
			msgReq_.SetQuestion(dns.Fqdn(tt.args.qName), tt.args.qType)
			msgReq_.RecursionDesired = true
			msgBytes_, err := msgReq_.Pack()
			if err != nil {
				t.Error(err)
				return
			}
			msgBase64_ := base64.RawURLEncoding.EncodeToString(msgBytes_)
			t.Logf("make get param for %v %v: %s", tt.args.qName, tt.args.qType, msgBase64_)
		})
	}
}

func TestDnsMsgResolver_Resolve(t *testing.T) {
	resolver_ := NewDohDnsMsgResolver(Quad9DnsMsgEndpoints, true, &CacheOptions{cacheType: CacheTypeInternal})
	type args struct {
		qName            string
		qType            uint16
		eDnsClientSubnet string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "t.tt",
			args: args{
				qType:            dns.TypeA,
				qName:            "t.tt",
				eDnsClientSubnet: "223.5.5.5",
			},
			wantErr: false,
		},
		{
			name: "baidu.com",
			args: args{
				qType:            dns.TypeAAAA,
				qName:            "baidu.com",
				eDnsClientSubnet: "114.114.114.114",
			},
			wantErr: false,
		},
		{
			name: "mi.com",
			args: args{
				qType:            dns.TypeSOA,
				qName:            "mi.com",
				eDnsClientSubnet: "208.67.222.2222",
			},
			wantErr: false,
		},
		{
			name: "google.com",
			args: args{
				qType:            dns.TypeNS,
				qName:            "google.com",
				eDnsClientSubnet: "138.183.55.112",
			},
			wantErr: false,
		},
		{
			name: "facebook.com",
			args: args{
				qType:            dns.TypeMX,
				qName:            "facebook.com",
				eDnsClientSubnet: "1.582.12.185",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsv := resolver_
			ecsIP_ := net.ParseIP(tt.args.eDnsClientSubnet)
			_, err := rsv.Resolve(tt.args.qName, tt.args.qType, &ecsIP_)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestCustomResolver4HttpClient(t *testing.T) {
	resolver := "tcp://223.5.5.5:53"
	url_, err := url.Parse(resolver)
	if err != nil {
		return
	}
	if !ListenAddrPortAvailable(url_.Host) {
		return
	}
	var (
		dnsResolverIP    = url_.Host   // Google DNS resolver.
		dnsResolverProto = url_.Scheme // Protocol to use for the DNS resolver
	)

	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: time.Duration(5000) * time.Millisecond,
					}
					return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
				},
			},
		}
		return dialer.DialContext(ctx, network, addr)
	}

	httpTransport_ := &http.Transport{
		Proxy:                 nil,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext:           dialContext, // Use the custom resolver
	}

	httpClient := &http.Client{
		Transport: httpTransport_,
	}
	for {
		resp, err := httpClient.Get("https://www.baidu.com")
		if err != nil {
			return
		}
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		fmt.Println(bodyStr)
		err = resp.Body.Close()
		if err != nil {
			return
		}
	}
}
