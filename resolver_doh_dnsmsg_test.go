package main

import (
	"encoding/base64"
	"github.com/miekg/dns"
	"net"
	"testing"
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
