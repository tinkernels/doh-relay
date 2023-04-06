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
		"119.96.90.251",
	}
	InitGeoipReader("")
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			t.Logf("ip %v country: %v", tt, GeoipCountry(net.ParseIP(tt)))
		})
	}
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			t.Logf("ip %v country: %v", tt, GeoipCountry(net.ParseIP(tt)))
		})
	}
	//InitGeoipReader("/usr/local/var/GeoIP/GeoLite2-City.mmdb")
	InitGeoipReader("/usr/local/var/GeoIP/dbip-city.mmdb")
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			t.Logf("ip %v country: %v", tt, GeoipCountry(net.ParseIP(tt)))
			t.Logf("ip %v location name: %v", tt, GeoIPLocName(net.ParseIP(tt)))
		})
	}
}
