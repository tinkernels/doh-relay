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
			country, state, city := GeoIPCountryStateCity(net.ParseIP(tt))
			t.Logf("ip %v country: %v, state: %v, city: %v", tt, country, state, city)
		})
	}
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			country, state, city := GeoIPCountryStateCity(net.ParseIP(tt))
			t.Logf("ip %v country: %v, state: %v, city: %v", tt, country, state, city)
		})
	}
	InitGeoipReader("/usr/local/var/GeoIP/dbip-city.mmdb")
	for _, tt := range tests {
		t.Run("geoIP", func(t *testing.T) {
			country, state, city := GeoIPCountryStateCity(net.ParseIP(tt))
			t.Logf("ip %v country: %v, state: %v, city: %v", tt, country, state, city)
		})
	}
}
