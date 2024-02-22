package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"regexp"
	"strings"
)

var (
	ExecConfig = ConfigModel{
		Dns53Config: Dns53ConfigModel{
			Enabled:       false,
			Listen:        "tcp://:53,udp://53",
			Upstream:      "https://dns.google/dns-query",
			UpstreamProto: "doh",
			EcsIP2nd:      "",
		},
		DohConfig: DohConfigModel{
			Enabled:       false,
			Listen:        DefaultDohListen,
			Upstream:      "tcp://8.8.8.8:53,tcp://9.9.9.9:53",
			UpstreamProto: "dns53",
			Path:          "/dns-query",
			EcsIP2nd:      "",
			UseTls:        false,
			TLSCertFile:   "",
			TLSKeyFile:    "",
		},
		CacheEnabled:    false,
		CacheBackend:    CacheTypeInternal,
		RedisURI:        "",
		GeoIPCityDBPath: "",
		LogLevel:        "info",
		IPv6Answer:      false,
		NamesInJail:     []NameInJailConfigModel{},
	}

	NamesInJailConfig = map[string][]*regexp.Regexp{}
)

const (
	RelayUpstreamProtoDoh   = "doh"
	RelayUpstreamProtoJson  = "doh_json"
	RelayUpstreamProtoDns53 = "dns53"
)

type NameInJailConfigModel struct {
	NameRegex    string `yaml:"name_regex"`
	CountryCodes string `yaml:"country_codes"`
}

type Dns53ConfigModel struct {
	Enabled          bool   `yaml:"enabled"`
	Listen           string `yaml:"listen"`
	Upstream         string `yaml:"upstream"`
	UpstreamFallback string `yaml:"upstream_fallback"`
	UpstreamProto    string `yaml:"upstream_proto"`
	EcsIP1st         string `yaml:"1st_ecs_ip"`
	EcsIP2nd         string `yaml:"2nd_ecs_ip"`
	UseClientIP      bool   `yaml:"use_client_ip"`
}

type DohConfigModel struct {
	Enabled          bool   `yaml:"enabled"`
	Listen           string `yaml:"listen"`
	Upstream         string `yaml:"upstream"`
	UpstreamFallback string `yaml:"upstream_fallback"`
	UpstreamProto    string `yaml:"upstream_proto"`
	Path             string `yaml:"path"`
	EcsIP1st         string `yaml:"1st_ecs_ip"`
	EcsIP2nd         string `yaml:"2nd_ecs_ip"`
	UseTls           bool   `yaml:"use_tls"`
	TLSCertFile      string `yaml:"tls_cert_file"`
	TLSKeyFile       string `yaml:"tls_key_file"`
	UseClientIP      bool   `yaml:"use_client_ip"`
}

type ConfigModel struct {
	Dns53Config          Dns53ConfigModel        `yaml:"dns53"`
	DohConfig            DohConfigModel          `yaml:"doh"`
	CacheEnabled         bool                    `yaml:"cache_enabled"`
	CacheBackend         string                  `yaml:"cache_backend"`
	RedisURI             string                  `yaml:"redis_uri"`
	GeoIPCityDBPath      string                  `yaml:"geoip_city_db_path"`
	LogLevel             string                  `yaml:"log_level"`
	IPv6Answer           bool                    `yaml:"ipv6_answer"`
	NamesInJail          []NameInJailConfigModel `yaml:"names_in_jail"`
	UpstreamHostResolver string                  `yaml:"upstream_host_resolver"`
}

func ReadConfigFromFile(path string) (config ConfigModel) {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Println("Read file error:", err)
		panic(err)
	}
	err = yaml.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Unmarshal config file error:", err)
		panic(err)
	}
	ExecConfig = config
	for _, nameInJail := range ExecConfig.NamesInJail {
		regexp_, err := regexp.Compile(nameInJail.NameRegex)
		if err != nil {
			fmt.Println("Compile regex error:", err)
			continue
		}
		countryCodes_ := strings.Split(nameInJail.CountryCodes, ",")
		for _, countryCode := range countryCodes_ {
			countryCode = strings.TrimSpace(countryCode)
			if countryCode == "" {
				continue
			}
			if _, ok := NamesInJailConfig[countryCode]; !ok {
				NamesInJailConfig[countryCode] = []*regexp.Regexp{}
			}
			NamesInJailConfig[countryCode] = append(NamesInJailConfig[countryCode], regexp_)
		}
	}
	return
}

func IsNameInJailOfCountry(name, countryCode string) bool {
	regexps_, ok := NamesInJailConfig[countryCode]
	if !ok {
		return false
	}
	chRet_, chFinal_, ret_ := make(chan bool, len(regexps_)), make(chan bool), false
	go func() {
		for i := 0; i < len(regexps_); i++ {
			if <-chRet_ {
				ret_ = true
			drainChannelLoop:
				for {
					select {
					case _ = <-chRet_:
					default:
						break drainChannelLoop
					}
				}
				break
			}
		}
		chFinal_ <- true
	}()
	for _, regexp_ := range regexps_ {
		go func(re *regexp.Regexp) {
			matched_ := re.MatchString(name)
			if ret_ {
				return
			}
			if matched_ {
				chRet_ <- true
			} else {
				chRet_ <- false
			}
		}(regexp_)
	}
	<-chFinal_
	return ret_
}
