package main

import (
	"github.com/miekg/dns"
	"testing"
)

func TestDnsMsgResolver_Resolve(t *testing.T) {
	resolver_ := NewDnsMsgResolver(Quad9DnsMsgEndpoints, true)
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
			_, err := rsv.Resolve(tt.args.qName, tt.args.qType, tt.args.eDnsClientSubnet)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
