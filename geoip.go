package main

import (
	// import "github.com/IncSW/geoip2"
	"github.com/oschwald/geoip2-golang"
	"net"
)

type GeoIPCountryStateCityFun func(ip net.IP) (string, string, string)

var GeoIPCountryStateCity GeoIPCountryStateCityFun

//func InitGeoipReader(maxmindDbPath string) {
//	reader, err := geoip2.NewCityReaderFromFile(maxmindDbPath)
//	if err != nil {
//		log.Info(err)
//		GeoIPCountryStateCity = func(ip net.IP) (string, string, string) {
//			return "", "", ""
//		}
//		return
//	}
//
//	GeoIPCountryStateCity = func(ip net.IP) (countryCode, stateName, city string) {
//		record, err := reader.Lookup(ip)
//		if err != nil {
//			log.Warn(err)
//			return "", "", ""
//		}
//		countryCode = record.Country.ISOCode
//		if len(record.Subdivisions) != 0 {
//			stateName = record.Subdivisions[0].Names["en"]
//		}
//		city = record.City.Names["en"]
//		return
//	}
//}

func InitGeoipReader(maxmindDbPath string) {
	db_, err := geoip2.Open(maxmindDbPath)
	if err != nil {
		log.Info(err)
		GeoIPCountryStateCity = func(ip net.IP) (string, string, string) {
			return "", "", ""
		}
		return
	}

	GeoIPCountryStateCity = func(ip net.IP) (countryCode, stateName, city string) {
		record, err := db_.City(ip)
		if err != nil {
			log.Warn(err)
			return "", "", ""
		}
		countryCode = record.Country.IsoCode
		if len(record.Subdivisions) != 0 {
			stateName = record.Subdivisions[0].Names["en"]
		}
		city = record.City.Names["en"]
		return
	}
}
