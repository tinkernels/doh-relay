package main

import (
	"net"
	"testing"
)

func TestInitGeoipReader(t *testing.T) {
	tests := []string{
		"23.78.102.25",
		"2600:141b:9000:0586:0000:0000:0000:24fe",
		"2600:141b:9000:05ae:0000:0000:0000:24fe",
	}
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			t.Logf("ip %v country: %v", tt, GeoipCountry(net.ParseIP(tt)))
		})
	}
	InitGeoipReader("")
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			t.Logf("ip %v country: %v", tt, GeoipCountry(net.ParseIP(tt)))
		})
	}
	InitGeoipReader("/usr/local/var/GeoIP/GeoLite2-City.mmdb")
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			t.Logf("ip %v country: %v", tt, GeoipCountry(net.ParseIP(tt)))
		})
	}
}
