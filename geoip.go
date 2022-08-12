package main

import (
	"github.com/IncSW/geoip2"
	"net"
)

var GeoipCountry = func(ip net.IP) string {
	return ""
}

func InitGeoipReader(maxmindDbPath string) {
	reader, err := geoip2.NewCityReaderFromFile(maxmindDbPath)
	if err != nil {
		log.Info(err)
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
}
