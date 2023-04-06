package main

import (
	"github.com/miekg/dns"
	"testing"
)

func TestDohJsonResolver_Resolve(t *testing.T) {
	type args struct {
		qName string
		qType uint16
	}
	resolver_ := NewDohJsonResolver(Quad9JsonEndpoints, true, &CacheOptions{cacheType: InternalCacheType})
	tests := []struct {
		name     string
		resolver *DohJsonResolver
		args     args
		wantErr  bool
	}{
		{
			name:     "test resolve t.tt",
			resolver: resolver_,
			args: args{
				qName: "t.tt",
				qType: dns.TypeA,
			},
			wantErr: false,
		},
		{
			name:     "test resolve g.alicdn.com",
			resolver: resolver_,
			args: args{
				qName: "g.alicdn.com",
				qType: dns.TypeA,
			},
			wantErr: false,
		},
		{
			name:     "test resolve google.com",
			resolver: resolver_,
			args: args{
				qName: "google.com",
				qType: dns.TypeA,
			},
			wantErr: false,
		},
		{
			name:     "test resolve will-reply-soa.google.com",
			resolver: resolver_,
			args: args{
				qName: "will-reply-soa.google.com",
				qType: dns.TypeA,
			},
			wantErr: false,
		},
		{
			name:     "test resolve ultradns.com",
			resolver: resolver_,
			args: args{
				qName: "ultradns.com",
				qType: dns.TypeNS,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsv := tt.resolver
			_, err := rsv.Resolve(tt.args.qName, tt.args.qType, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
