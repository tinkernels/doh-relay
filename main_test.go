package main

import (
	"net/netip"
	"testing"
)

func Test_ParseListen(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{
			name: "test_parse_listen",
			arg:  ":15353",
		},
		{
			name: "test_parse_listen",
			arg:  "127.0.0.1",
		},
		{
			name: "test_parse_listen",
			arg:  "127.0.0.1:15353",
		},
		{
			name: "test_parse_listen",
			arg:  "[::1]:15353",
		},
		{
			name: "test_parse_listen",
			arg:  "::1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			IpPort, err := netip.ParseAddrPort(tt.arg)
			if err != nil {
				t.Logf("ParseAddrPort %v error = %v", tt.arg, err)
			}
			t.Logf("Parse %v to IP: %v, Port: %v", tt.arg, IpPort.Addr(), IpPort.Port())
		})
	}
}
