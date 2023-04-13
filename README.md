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
        Enable DoH response cache. (default true)
  -cache-backend string
        Specify cache backend (default "internal")
  -dns53
        Enable dns53 service.
  -dns53-2nd-ecs-ip string
        Set dns53 secondary edns_client_subnet ip, eg: 12.34.56.78.
  -dns53-listen string
        Set dns53 service listen port. (default "udp://:53,tcp://:53")
  -dns53-upstream string
        Upstream DoH resolver for dns53 service, e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query
  -dns53-upstream-dns53
        If dns53 upstream endpoints using dns53 protocol.
  -dns53-upstream-json
        If dns53 upstream endpoints transfer with json format.
  -loglevel string
        Set log level. (default "info")
  -maxmind-citydb-file string
        Specify maxmind city db file path.
  -redis-uri string
        Specify redis uri for caching (default "redis://localhost:6379/0")
  -relay
        Enable DoH relay service.
  -relay-2nd-ecs-ip string
        Specify secondary edns-client-subnet ip, eg: 12.34.56.78
  -relay-listen string
        Set relay service listen port. (default "127.0.0.1:15353")
  -relay-path string
        DNS-over-HTTPS endpoint path. (default "/dns-query")
  -relay-tls
        Enable DoH relay service over TLS, default on clear http.
  -relay-tls-cert string
        Specify tls cert path.
  -relay-tls-key string
        Specify tls key path.
  -relay-upstream string
        Upstream DoH resolver for relay service, e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query
  -relay-upstream-dns53
        If relay upstream endpoints using dns53 protocol.
  -relay-upstream-json
        If relay upstream endpoints transfer with json format.
  -version
        Print version info.
```

### Example

- Setup a `DNS53` (listening on `tcp://0.0.0.0:53` and `udp://0.0.0.0:53`) service relaying DNS queries to `DNS-over-HTTPS` service (`https://9.9.9.11/dns-query`): 

  ```
  doh-relay -dns53 -dns53-listen tcp://:53,udp://:53 -dns53-upstream https://9.9.9.11/dns-query -maxmind-citydb-file /usr/local/var/GeoIP/GeoLite2-City.mmdb
  ```

- Setup a `DNS-over-HTTPS` (listening on `http://0.0.0.0:15353`) service relaying DNS queries to `DNS-over-HTTPS` service (`https://9.9.9.11/dns-query`), also with internal cache on: 

  ```
  doh-relay -cache -relay -relay-listen :15353 -relay-upstream https://9.9.9.11/dns-query -maxmind-citydb-file /var/lib/GeoIP/GeoLite2-City.mmdb
  ```
- Setup a `DNS-over-HTTPS` (listening on `http://0.0.0.0:15353`) service relaying DNS queries to `DNS53` service (`tcp://9.9.9.11:53`): 

  ```
  doh-relay -relay -relay-listen :15353 -relay-upstream tcp://9.9.9.11:53 -relay-upstream-dns53 -maxmind-citydb-file /var/lib/GeoIP/GeoLite2-City.mmdb
  ```

## Thanks

[github.com/miekg/dns](https://github.com/miekg/dns)

## License

[Apache-2.0](https://github.com/tinkernels/doh-relay/blob/master/LICENSE)
