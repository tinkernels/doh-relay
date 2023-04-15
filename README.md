# doh-relay &middot; [![License](https://img.shields.io/hexpm/l/plug?logo=Github&style=flat)](https://github.com/tinkernels/doh-relay/blob/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/tinkernels/doh-relay)](https://goreportcard.com/report/github.com/tinkernels/doh-relay)
doh-relay is a tool for relaying DNS queries

- Ability to provide `DNS53` and `DNS-over-HTTPS` services simultaneously. 

- Relay DNS queries to upsteram service (can be `DNS53` or `DNS-over-HTTPS`). 

- Support `EDNS-Client-Subnet`.  

## Build

```
make release
```


## Usage 

```
Usage:

  doh-relay [options]

Options:

  -cache
        Enable cache for DNS answers. (default true)
  -cache-backend string
        Specify cache backend (default "internal")
  -config string
        use config file (yaml format)
  -dns53
        Enable dns53 relay service.
  -dns53-2nd-ecs-ip string
        Set dns53 secondary EDNS-Client-Subnet ip, eg: 12.34.56.78.
  -dns53-listen string
        Set dns53 service listen port. (default "udp://:53,tcp://:53")
  -dns53-upstream string
        Upstream resolver for dns53 service (default upstream type is standard DoH), e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query
  -dns53-upstream-dns53
        If dns53 service relays DNS queries to upstream endpoints using dns53 protocol.
  -dns53-upstream-json
        If dns53 service relays DNS queries to upstream endpoints transfer with json format.
  -doh
        Enable DoH relay service.
  -doh-2nd-ecs-ip string
        Specify secondary EDNS-Client-Subnet ip, eg: 12.34.56.78
  -doh-listen string
        Set doh relay service listen port. (default "127.0.0.1:15353")
  -doh-path string
        DNS-over-HTTPS endpoint path. (default "/dns-query")
  -doh-tls
        Enable DoH relay service over TLS, default on clear http.
  -doh-tls-cert string
        Specify tls cert path.
  -doh-tls-key string
        Specify tls key path.
  -doh-upstream string
        Upstream resolver for doh service (default upstream type is standard DoH), e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query
  -doh-upstream-dns53
        If DoH service relays queries to upstream endpoints using dns53 protocol.
  -doh-upstream-json
        If DoH service relays queries to upstream DoH endpoints transfer with json format.
  -loglevel string
        Set log level. (default "info")
  -maxmind-citydb-file string
        Specify maxmind city db file path.
  -redis-uri string
        Specify redis uri for caching
  -version
        Print version info.
```
### Config file
  There's a example config file with comments [here](config-example.yml).

### Usage example

- Set up a `DNS53` (listening on `tcp://0.0.0.0:53` and `udp://0.0.0.0:53`) service relaying DNS queries to `DNS-over-HTTPS` service (`https://9.9.9.11/dns-query`): 

  ```
  doh-relay -dns53 -dns53-listen tcp://:53,udp://:53 -dns53-upstream https://9.9.9.11/dns-query -maxmind-citydb-file /usr/local/var/GeoIP/GeoLite2-City.mmdb
  ```

- Set up a `DNS-over-HTTPS` (listening on `http://0.0.0.0:15353`) service relaying DNS queries to `DNS-over-HTTPS` service (`https://9.9.9.11/dns-query`), also with internal cache on: 

  ```
  doh-relay -cache -doh -doh-listen :15353 -doh-upstream https://9.9.9.11/dns-query -maxmind-citydb-file /var/lib/GeoIP/GeoLite2-City.mmdb
  ```

- Set up a `DNS-over-HTTPS` (listening on `http://0.0.0.0:15353`) service relaying DNS queries to `DNS53` service (`tcp://9.9.9.11:53`): 

  ```
  doh-relay -doh -doh-listen :15353 -doh-upstream tcp://9.9.9.11:53 -doh-upstream-dns53 -maxmind-citydb-file /var/lib/GeoIP/GeoLite2-City.mmdb
  ```

## TODOs

  - [ ] Implement redis cache backend.

## Thanks

[github.com/miekg/dns](https://github.com/miekg/dns)

## License

[Apache-2.0](https://github.com/tinkernels/doh-relay/blob/master/LICENSE)
