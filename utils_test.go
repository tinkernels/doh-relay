package main

import "testing"

func TestGetExIPByResolver(t *testing.T) {
	type args struct {
		rsv Resolver
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"DNS53Resolver",
			args{
				rsv: NewDns53DnsMsgResolver([]string{"tcp://223.5.5.5:53"}, false, &CacheOptions{cacheType: InternalCacheType}),
			},
		},
		{
			"DohResolver",
			args{
				rsv: NewDohDnsMsgResolver([]string{"https://1.1.1.1/dns-query"}, false, &CacheOptions{cacheType: InternalCacheType}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIpStr := GetExIPByResolver(tt.args.rsv)
			t.Logf("GetExIPByResolver() = %v", gotIpStr)
		})
	}
}
