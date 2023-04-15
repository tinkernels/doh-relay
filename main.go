package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	logger "github.com/sirupsen/logrus"
	"net/url"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const CurrentVersion = "v1.0.0-beta.7"
const DefaultRelayListenAddr = "127.0.0.1:15353"

var (
	dns53Flag = flag.Bool(
		"dns53", false, "Enable dns53 relay service.",
	)
	dns53ListenFlag = flag.String(
		"dns53-listen",
		"udp://:53,tcp://:53", "Set dns53 service listen port.",
	)
	dns532ndECSIPsFlag = flag.String(
		"dns53-2nd-ecs-ip",
		"",
		"Set dns53 secondary EDNS-Client-Subnet ip, eg: 12.34.56.78.",
	)
	dns53UpstreamFlag = flag.String(
		"dns53-upstream",
		"",
		"Upstream resolver for dns53 service (default upstream type is standard DoH), "+
			"e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query",
	)
	dns53UpstreamJsonFlag = flag.Bool(
		"dns53-upstream-json",
		false,
		"If dns53 service relays DNS queries to upstream endpoints transfer with json format.",
	)
	dns53UpstreamDns53Flag = flag.Bool(
		"dns53-upstream-dns53",
		false,
		"If dns53 service relays DNS queries to upstream endpoints using dns53 protocol.",
	)
	dohFlag = flag.Bool(
		"doh",
		false,
		"Enable DoH relay service.",
	)
	dohListenFlag = flag.String(
		"doh-listen",
		DefaultRelayListenAddr, "Set doh relay service listen port.",
	)
	dohPathFlag = flag.String(
		"doh-path",
		"/dns-query",
		"DNS-over-HTTPS endpoint path.",
	)
	dohUpstreamFlag = flag.String(
		"doh-upstream",
		"",
		"Upstream resolver for doh service (default upstream type is standard DoH), "+
			"e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query",
	)
	dohUpstreamJsonFlag = flag.Bool(
		"doh-upstream-json",
		false,
		"If DoH service relays queries to upstream DoH endpoints transfer with json format.",
	)
	dohUpstreamDns53Flag = flag.Bool(
		"doh-upstream-dns53",
		false,
		"If DoH service relays queries to upstream endpoints using dns53 protocol.",
	)
	dohTlsFlag = flag.Bool(
		"doh-tls",
		false,
		"Enable DoH relay service over TLS, default on clear http.",
	)
	dohTlsCertFlag = flag.String(
		"doh-tls-cert",
		"",
		"Specify tls cert path.",
	)
	dohTlsKeyFlag = flag.String(
		"doh-tls-key",
		"",
		"Specify tls key path.",
	)
	doh2ndECSIPFlag = flag.String(
		"doh-2nd-ecs-ip",
		"",
		"Specify secondary EDNS-Client-Subnet ip, eg: 12.34.56.78",
	)
	maxmindCityDBFileFlag = flag.String(
		"maxmind-citydb-file",
		"",
		"Specify maxmind city db file path.",
	)
	cacheFlag = flag.Bool(
		"cache",
		true,
		"Enable cache for DNS answers.",
	)
	cacheBackendFLag = flag.String(
		"cache-backend",
		InternalCacheType,
		"Specify cache backend",
	)
	redisURIFLag = flag.String(
		"redis-uri",
		"redis://localhost:6379/0",
		"Specify redis uri for caching",
	)
	logLevelFlag = flag.String(
		"loglevel",
		"info",
		"Set log level.",
	)
	versionFlag = flag.Bool(
		"version",
		false,
		"Print version info.",
	)
)

var log = &logger.Logger{
	Out: os.Stdout,
	Formatter: &logger.TextFormatter{
		CallerPrettyfier: func(caller *runtime.Frame) (function string, file string) {
			function = ""
			_, filename_ := path.Split(caller.File)
			file = fmt.Sprintf("%s:%d", filename_, caller.Line)
			return
		},
		TimestampFormat: "2006-01-02T15:04:05",
	},
	Level:        logger.DebugLevel,
	ReportCaller: true,
}

var (
	RelayAnswerer *DnsMsgAnswerer
	Dns53Answerer *DnsMsgAnswerer
)

func printVersion() {
	fmt.Println(CurrentVersion)
}

func main() {

	// Exit on some signals.
	termSig_ := make(chan os.Signal)
	signal.Notify(termSig_, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig_ := <-termSig_
		fmt.Printf("*** Terminating from signal [%+v] ***\n", sig_)
		os.Exit(0)
	}()

	flag.Usage = func() {
		_, execPath_ := filepath.Split(os.Args[0])
		_, _ = fmt.Fprint(os.Stderr, "DNS-over-HTTPS relay service.\n\n")
		_, _ = fmt.Fprint(os.Stderr, "Version: "+CurrentVersion+".\n\n")
		_, _ = fmt.Fprintf(os.Stderr, "Usage:\n\n  %s [options]\n\nOptions:\n\n", execPath_)
		flag.PrintDefaults()
	}
	flag.Parse()

	if *versionFlag {
		printVersion()
		return
	}

	fmt.Println("*** Starting ***")

	// Set the loglevel
	logLevel_, err := logger.ParseLevel(*logLevelFlag)
	if err != nil {
		log.Warnf("invalid log level: %v", err)
	}
	log.SetLevel(logLevel_)

	InitGeoipReader(*maxmindCityDBFileFlag)

	chRelaySvc_, chDns53Svc_ := make(chan error), make(chan error)

	if *dohFlag {
		initDohRsvAnswerer()
		go serveDohSvc(chRelaySvc_)
	}

	if *dns53Flag {
		initDns53RsvAnswerer()
		go serveDns53Svc(chDns53Svc_)
	}

	// Log services exit errors.
	if *dohFlag {
		serveRelayErr_ := <-chRelaySvc_
		log.Infof("relay service exit: %+v", serveRelayErr_)
	}
	if *dns53Flag {
		serveDns53Err_ := <-chDns53Svc_
		log.Infof("dns53 service exit: %+v", serveDns53Err_)
	}
	os.Exit(0)
}

// initDohRsvAnswerer initializes the DNS-over-HTTPS upstream query service.
func initDohRsvAnswerer() {
	var upstreamEndpoints_ []string
	if tmpEndpoints_ := strings.Split(*dohUpstreamFlag, ","); *dohUpstreamFlag != "" &&
		len(tmpEndpoints_) > 0 {
		upstreamEndpoints_ = make([]string, len(tmpEndpoints_))
		for i_ := range tmpEndpoints_ {
			upstreamEndpoints_[i_] = strings.TrimSpace(tmpEndpoints_[i_])
		}
	}
	var resolver Resolver
	cacheOptions_ := &CacheOptions{cacheType: *cacheBackendFLag, redisURI: *redisURIFLag}
	if *dohUpstreamJsonFlag {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9JsonEndpoints
		}
		resolver = NewDohJsonResolver(upstreamEndpoints_, *cacheFlag, cacheOptions_)
	} else if *dohUpstreamDns53Flag {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9Dns53Endpoints
		}
		resolver = NewDns53DnsMsgResolver(upstreamEndpoints_, *cacheFlag, cacheOptions_)
	} else {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9DnsMsgEndpoints
		}
		resolver = NewDohDnsMsgResolver(upstreamEndpoints_, *cacheFlag, cacheOptions_)
	}
	RelayAnswerer = NewDnsMsgAnswerer(resolver)
}

// initDns53RsvAnswerer initializes the DNS-over-HTTPS upstream query service.
func initDns53RsvAnswerer() {
	var upstreamEndpoints_ []string
	if tmpEndpoints_ := strings.Split(*dns53UpstreamFlag, ","); *dns53UpstreamFlag != "" &&
		len(tmpEndpoints_) > 0 {
		upstreamEndpoints_ = make([]string, len(tmpEndpoints_))
		for i_ := range tmpEndpoints_ {
			upstreamEndpoints_[i_] = strings.TrimSpace(tmpEndpoints_[i_])
		}
	}
	var resolver Resolver
	cacheOptions_ := &CacheOptions{cacheType: *cacheBackendFLag, redisURI: *redisURIFLag}
	if *dns53UpstreamJsonFlag {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9JsonEndpoints
		}
		resolver = NewDohJsonResolver(upstreamEndpoints_, *cacheFlag, cacheOptions_)
	} else if *dns53UpstreamDns53Flag {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9Dns53Endpoints
		}
		resolver = NewDns53DnsMsgResolver(upstreamEndpoints_, *cacheFlag, cacheOptions_)
	} else {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9DnsMsgEndpoints
		}
		resolver = NewDohDnsMsgResolver(upstreamEndpoints_, *cacheFlag, cacheOptions_)
	}
	Dns53Answerer = NewDnsMsgAnswerer(resolver)
}

func serveDohSvc(c chan error) {
	// Set Gin mode referred to loglevel.
	var err error
	if logLevel_, err := logger.ParseLevel(*logLevelFlag); err == nil && logLevel_ >= logger.DebugLevel {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router_ := gin.Default()
	err = router_.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"})
	if err != nil {
		c <- err
		return
	}
	router_.RemoteIPHeaders = []string{"X-Real-IP"}

	dohHandler := NewDohHandler()
	if *doh2ndECSIPFlag != "" {
		dohHandler.AppendDefaultECSIPStr(*doh2ndECSIPFlag)
	}

	// Routes.
	router_.GET(*dohPathFlag, dohHandler.DohGetHandler)
	router_.POST(*dohPathFlag, dohHandler.DohPostHandler)

	listenAddr_ := DefaultRelayListenAddr
	if ListenAddrPortAvailable(*dohListenFlag) {
		listenAddr_ = *dohListenFlag
	}
	if *dohTlsFlag {
		if !PathExists(*dohTlsCertFlag) || !PathExists(*dohTlsKeyFlag) {
			c <- fmt.Errorf("missing tls cert or key")
			return
		}
		err = router_.RunTLS(listenAddr_,
			*dohTlsCertFlag,
			*dohTlsKeyFlag,
		)
		c <- err
		return
	}
	err = router_.Run(listenAddr_)
	c <- err
}

func serveDns53Svc(c chan error) {
	dns53Handler := NewDns53Handler()
	if *dns532ndECSIPsFlag != "" {
		dns53Handler.AppendDefaultECSIPStr(*dns532ndECSIPsFlag)
	}
	dns.HandleFunc(".", dns53Handler.ServeDNS)
	dns53ListenAddrs_ := strings.Split(*dns53ListenFlag, ",")
	var dns53CHs_ []chan error
	for i := range dns53ListenAddrs_ {
		url_, err := url.Parse(strings.TrimSpace(dns53ListenAddrs_[i]))
		if err != nil {
			c <- err
			return
		}
		if !ListenAddrPortAvailable(url_.Host) {
			continue
		}
		if strings.ToLower(url_.Scheme) == "udp" {
			c_ := make(chan error)
			dns53CHs_ = append(dns53CHs_, c_)
			go serveDns53UDP(url_.Host, c_)
			log.Infof("dns53 listening on %s", url_.String())
		} else if strings.ToLower(url_.Scheme) == "tcp" {
			c_ := make(chan error)
			dns53CHs_ = append(dns53CHs_, c_)
			go serveDns53TCP(url_.Host, c_)
			log.Infof("dns53 listening on %s", url_.String())
		}
	}
	// Collect dns53 services errors.
	var errs_ []error
	for _, c := range dns53CHs_ {
		err_ := <-c
		if err_ != nil {
			errs_ = append(errs_, <-c)
		}
	}
	if len(errs_) > 0 {
		c <- fmt.Errorf("%+v", errs_)
		return
	}
	c <- nil
}

func serveDns53TCP(addr string, c chan error) {
	server := &dns.Server{Addr: addr, Net: "tcp", Handler: nil, TsigSecret: nil}
	if err := server.ListenAndServe(); err != nil {
		log.Errorf("Failed to setup the %s dns53 server on %s: %v", "tcp", addr, err)
	}
	c <- nil
}
func serveDns53UDP(addr string, c chan error) {
	server := &dns.Server{Addr: addr, Net: "udp", Handler: nil, TsigSecret: nil}
	if err := server.ListenAndServe(); err != nil {
		log.Errorf("Failed to setup the %s dns53 server on %s: %v", "tcp", addr, err)
	}
	c <- nil
}
