package main

import (
	"fmt"
	"github.com/IncSW/geoip2"
	"net"
)

type GeoIPCountryGetter func(ip net.IP) string
type GeoIPLocNameGetter func(ip net.IP) string

var (
	GeoipCountry GeoIPCountryGetter
	GeoIPLocName GeoIPLocNameGetter
)

func InitGeoipReader(maxmindDbPath string) {
	reader, err := geoip2.NewCityReaderFromFile(maxmindDbPath)
	if err != nil {
		log.Info(err)
		GeoipCountry = func(ip net.IP) string {
			return ""
		}
		GeoIPLocName = func(ip net.IP) string {
			return ""
		}
		return
	}

	GeoipCountry = func(ip net.IP) (countryISOCode string) {
		record, err := reader.Lookup(ip)
		if err != nil {
			log.Warn(err)
			return ""
		}
		countryISOCode = record.Country.ISOCode
		return
	}

	GeoIPLocName = func(ip net.IP) (locationName string) {
		record, err := reader.Lookup(ip)
		if err != nil {
			log.Warn(err)
			return ""
		}
		countryISOCode := record.Country.ISOCode
		if len(record.Subdivisions) != 0 {
			locationName = fmt.Sprintf("%s-%s", countryISOCode, record.Subdivisions[0].Names["en"])
		} else {
			locationName = countryISOCode
		}
		return
	}
}
