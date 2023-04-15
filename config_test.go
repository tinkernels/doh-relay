package main

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"os"
	"testing"
)

func TestConfigModelSerialization(t *testing.T) {
	cfg := ConfigModel{
		Dns53Config: Dns53ConfigModel{
			Enabled:       true,
			Listen:        "127.0.0.1:53",
			Upstream:      "8.8.8.8:53",
			UpstreamProto: RelayUpstreamProtoDns53,
			EcsIP2nd:      "192.0.2.1",
		},
		DohConfig: DohConfigModel{
			Enabled:       true,
			Listen:        "127.0.0.1:443",
			Upstream:      "https://dns.google/dns-query",
			UpstreamProto: RelayUpstreamProtoDoh,
			Path:          "/dns-query",
			EcsIP2nd:      "192.0.2.1",
			UseTls:        true,
			TLSCertFile:   "/path/to/cert.pem",
			TLSKeyFile:    "/path/to/key.pem",
		},
		CacheEnabled:    true,
		CacheBackend:    "redis",
		RedisURI:        "redis://localhost:6379",
		GeoIPCityDBPath: "/path/to/GeoIPCity.dat",
		LogLevel:        "info",
		NamesInJail: []NameInJailConfigModel{
			{
				NameRegex:    "badguy.*",
				CountryCodes: "RU",
			},
			{
				NameRegex:    "evilcorp.*",
				CountryCodes: "CN,RU",
			},
		},
	}

	// Serialize the struct to YAML
	data, err := yaml.Marshal(cfg)
	assert.NoError(t, err)

	// Deserialize the YAML to a new struct
	var cfg2 ConfigModel
	err = yaml.Unmarshal(data, &cfg2)
	assert.NoError(t, err)

	// Check that the deserialized struct is equal to the original
	assert.Equal(t, cfg, cfg2)
}

func TestReadConfigFromFile(t *testing.T) {
	// Create a temporary file with sample configuration data
	tmpFile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) // clean up

	sampleConfig := []byte(`
dns53:
  enabled: true
  listen: "127.0.0.1:53"
  upstream: "8.8.8.8:53"
  upstream_proto: "udp"
  ecs_ip_2nd: "192.168.1.1"
doh:
  enabled: true
  listen: "127.0.0.1:443"
  upstream: "https://dns.google/dns-query"
  upstream_proto: "doh"
  path: "/dns-query"
  ecs_ip_2nd: "192.168.1.1"
  use_tls: true
  tls_cert_file: "/path/to/cert.pem"
  tls_key_file: "/path/to/key.pem"
cache_enabled: true
cache_backend: "redis"
redis_uri: "redis://localhost:6379"
geoip_city_db_path: "/path/to/GeoIPCity.dat"
log_level: "info"
names_in_jail:
  - name_regex: ".*"
    country_codes: "US,CA"
  - name_regex: ".*"
    country_codes: "GB,IE"
`)

	if _, err := tmpFile.Write(sampleConfig); err != nil {
		t.Fatal(err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatal(err)
	}

	// Call ReadConfigFromFile with the temporary file path
	config := ReadConfigFromFile(tmpFile.Name())

	// Check if the returned ConfigModel is correct
	if config.Dns53Config.Listen != "127.0.0.1:53" {
		t.Errorf("Expected Dns53Config.Listen to be '127.0.0.1:53', but got '%s'", config.Dns53Config.Listen)
	}
	if config.DohConfig.Listen != "127.0.0.1:443" {
		t.Errorf("Expected DohConfig.Listen to be '127.0.0.1:443', but got '%s'", config.DohConfig.Listen)
	}
	if config.CacheEnabled != true {
		t.Errorf("Expected CacheEnabled to be true, but got false")
	}
	if config.CacheBackend != "redis" {
		t.Errorf("Expected CacheBackend to be 'redis', but got '%s'", config.CacheBackend)
	}
	if config.RedisURI != "redis://localhost:6379" {
		t.Errorf("Expected RedisURI to be 'redis://localhost:6379', but got '%s'", config.RedisURI)
	}
	if config.GeoIPCityDBPath != "/path/to/GeoIPCity.dat" {
		t.Errorf("Expected GeoIPCityDBPath to be '/path/to/GeoIPCity.dat', but got '%s'", config.GeoIPCityDBPath)
	}
	if config.LogLevel != "info" {
		t.Errorf("Expected LogLevel to be 'info', but got '%s'", config.LogLevel)
	}
	if len(config.NamesInJail) != 2 {
		t.Errorf("Expected NamesInJail to have 2 elements, but got %d", len(config.NamesInJail))
	}
	if config.NamesInJail[0].NameRegex != ".*" {
		t.Errorf("Expected NamesInJail[0].NameRegex to be '.*', but got '%s'", config.NamesInJail[0].NameRegex)
	}
	if config.NamesInJail[0].CountryCodes != "US,CA" {
		t.Errorf("Expected NamesInJail[0].CountryCodes to be 'US,CA', but got '%s'", config.NamesInJail[0].CountryCodes)
	}
	if config.NamesInJail[1].NameRegex != ".*" {
		t.Errorf("Expected NamesInJail[1].NameRegex to be '.*', but got '%s'", config.NamesInJail[1].NameRegex)
	}
	if config.NamesInJail[1].CountryCodes != "GB,IE" {
		t.Errorf("Expected NamesInJail[1].CountryCodes to be 'GB,IE', but got '%s'", config.NamesInJail[1].CountryCodes)
	}
}
