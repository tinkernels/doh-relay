package main

import (
	"encoding/base64"
	"testing"
)

func Test_Base64Encoding(t *testing.T) {
	tests := []struct {
		name string
		arg  string
	}{
		{
			name: "test_base64_encoding",
			arg: "AAABAAABAAAAAAABCmNsb3VkZmxhcmUDY29tAAABAAEAACkQAAAAAAAAVQAMAFEAAAAAAAAAAAAAAAAAAAAAAAAA" +
				"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decodeString, err := base64.RawURLEncoding.DecodeString(tt.arg)
			if err != nil {
				return
			}
			t.Logf("Base64Decode %v to %v", tt.arg, decodeString)
		})
	}
}
