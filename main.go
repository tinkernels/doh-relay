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
	"regexp"
	"runtime"
	"strings"
	"syscall"
)

const CurrentVersion = "v1.0.0"
const DefaultDohListen = "127.0.0.1:15353"

var (
	configFileFlag = flag.String(
		"config",
		"",
		"use config file (yaml format)",
	)
	dns53Flag = flag.Bool(
		"dns53", false, "Enable dns53 relay service.",
	)
	dns53ListenFlag = flag.String(
		"dns53-listen",
		"udp://:53,tcp://:53", "Set dns53 service listen port.",
	)
	dns53UseClientIPFlag = flag.Bool(
		"dns53-use-client-ip",
		false,
		"If dns53 service use client ip as ECS.",
	)
	dns531stECSIPsFlag = flag.String(
		"dns53-1st-ecs-ip",
		"",
		"Set dns53 primary EDNS-Client-Subnet ip, eg: 12.34.56.78.",
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
	dohUseClientIPFlag = flag.Bool(
		"doh-use-client-ip",
		false,
		"If doh service use client ip as ECS.",
	)
	dohListenFlag = flag.String(
		"doh-listen",
		DefaultDohListen, "Set doh relay service listen port.",
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
	doh1stECSIPFlag = flag.String(
		"doh-1st-ecs-ip",
		"",
		"Specify primary EDNS-Client-Subnet ip, eg: 12.34.56.78",
	)
	doh2ndECSIPFlag = flag.String(
		"doh-2nd-ecs-ip",
		"",
		"Specify secondary EDNS-Client-Subnet ip, eg: 12.34.56.78",
	)
	ipv6AnswerFlag = flag.Bool(
		"ipv6-answer",
		false,
		"Enable ipv6 answer.",
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
		CacheTypeInternal,
		"Specify cache backend",
	)
	redisURIFLag = flag.String(
		"redis-uri",
		"",
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

func fillExecConfigFromFlags() {

	ExecConfig.Dns53Config.Enabled = *dns53Flag
	ExecConfig.Dns53Config.Listen = *dns53ListenFlag
	ExecConfig.Dns53Config.Upstream = *dns53UpstreamFlag
	if *dns53UpstreamJsonFlag {
		ExecConfig.Dns53Config.UpstreamProto = RelayUpstreamProtoJson
	} else if *dns53UpstreamDns53Flag {
		ExecConfig.Dns53Config.UpstreamProto = RelayUpstreamProtoDns53
	} else {
		ExecConfig.Dns53Config.UpstreamProto = RelayUpstreamProtoDoh
	}
	ExecConfig.Dns53Config.EcsIP2nd = *dns532ndECSIPsFlag
	ExecConfig.Dns53Config.EcsIP1st = *dns531stECSIPsFlag
	ExecConfig.Dns53Config.UseClientIP = *dns53UseClientIPFlag

	ExecConfig.DohConfig.Enabled = *dohFlag
	ExecConfig.DohConfig.Listen = *dohListenFlag
	ExecConfig.DohConfig.Upstream = *dohUpstreamFlag
	if *dohUpstreamJsonFlag {
		ExecConfig.DohConfig.UpstreamProto = RelayUpstreamProtoJson
	} else if *dohUpstreamDns53Flag {
		ExecConfig.DohConfig.UpstreamProto = RelayUpstreamProtoDns53
	} else {
		ExecConfig.DohConfig.UpstreamProto = RelayUpstreamProtoDoh
	}
	ExecConfig.DohConfig.Path = *dohPathFlag
	ExecConfig.DohConfig.EcsIP2nd = *doh2ndECSIPFlag
	ExecConfig.DohConfig.UseTls = *dohTlsFlag
	ExecConfig.DohConfig.TLSCertFile = *dohTlsCertFlag
	ExecConfig.DohConfig.TLSKeyFile = *dohTlsKeyFlag
	ExecConfig.DohConfig.UseClientIP = *dohUseClientIPFlag
	ExecConfig.DohConfig.EcsIP1st = *doh1stECSIPFlag

	ExecConfig.CacheEnabled = *cacheFlag
	ExecConfig.CacheBackend = *cacheBackendFLag
	ExecConfig.RedisURI = *redisURIFLag
	ExecConfig.GeoIPCityDBPath = *maxmindCityDBFileFlag
	ExecConfig.IPv6Answer = *ipv6AnswerFlag
	ExecConfig.LogLevel = *logLevelFlag
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
	if *configFileFlag != "" && PathExists(*configFileFlag) {
		ReadConfigFromFile(*configFileFlag)
	} else {
		fillExecConfigFromFlags()
	}

	if *versionFlag {
		printVersion()
		return
	}

	fmt.Println("*** Starting ***")

	// Set the loglevel
	logLevel_, err := logger.ParseLevel(ExecConfig.LogLevel)
	if err != nil {
		log.Warnf("invalid log level: %v", err)
	}
	log.SetLevel(logLevel_)

	InitGeoipReader(ExecConfig.GeoIPCityDBPath)

	chRelaySvc_, chDns53Svc_ := make(chan error), make(chan error)

	if ExecConfig.DohConfig.Enabled {
		initDohRsvAnswerer()
		go serveDohSvc(chRelaySvc_)
	}

	if ExecConfig.Dns53Config.Enabled {
		initDns53RsvAnswerer()
		go serveDns53Svc(chDns53Svc_)
	}

	// Log services exit errors.
	if ExecConfig.DohConfig.Enabled {
		serveRelayErr_ := <-chRelaySvc_
		log.Infof("relay service exit: %+v", serveRelayErr_)
	}
	if ExecConfig.Dns53Config.Enabled {
		serveDns53Err_ := <-chDns53Svc_
		log.Infof("dns53 service exit: %+v", serveDns53Err_)
	}
	os.Exit(0)
}

func initFixedResolvers(t UpstreamType, conf []FixedResolvingConfigModel) (resolvers map[*regexp.Regexp]Resolver) {
	cacheOptions_ := &CacheOptions{cacheType: ExecConfig.CacheBackend, redisURI: ExecConfig.RedisURI}
	resolvers = make(map[*regexp.Regexp]Resolver)
	for _, f := range conf {
		c_ := cacheOptions_
		pattern_, err := regexp.Compile(f.NameRegex)
		if err != nil {
			log.Warnf("domain name regex invalid: %+v", f.NameRegex)
		}
		if t == RelayUpstreamProtoDoh {
			resolvers[pattern_] = NewDohDnsMsgResolver([]string{f.Server}, true, c_)
		} else if t == RelayUpstreamProtoJson {
			resolvers[pattern_] = NewDohJsonResolver([]string{f.Server}, true, c_)
		} else if t == RelayUpstreamProtoDns53 {
			resolvers[pattern_] = NewDns53DnsMsgResolver([]string{f.Server}, true, c_)
		} else {
			continue
		}
	}
	return
}

// initDohRsvAnswerer initializes the DNS-over-HTTPS upstream query service.
func initDohRsvAnswerer() {
	var upstreamEndpoints_, fallbackUpstreamEndpoints_, tmpEndpoints_ []string

	tmpEndpoints_ = strings.Split(ExecConfig.DohConfig.Upstream, ",")
	for _, edp := range tmpEndpoints_ {
		if trimmedEdp_ := strings.TrimSpace(edp); trimmedEdp_ != "" {
			upstreamEndpoints_ = append(upstreamEndpoints_, trimmedEdp_)
		}
	}

	tmpEndpoints_ = strings.Split(ExecConfig.DohConfig.UpstreamFallback, ",")
	for _, edp := range tmpEndpoints_ {
		if trimmedEdp_ := strings.TrimSpace(edp); trimmedEdp_ != "" {
			fallbackUpstreamEndpoints_ = append(fallbackUpstreamEndpoints_, trimmedEdp_)
		}
	}

	var resolver, fallbackResolver Resolver
	fixedResolvers := make(map[*regexp.Regexp]Resolver)
	cacheOptions_ := &CacheOptions{cacheType: ExecConfig.CacheBackend, redisURI: ExecConfig.RedisURI}
	if ExecConfig.DohConfig.UpstreamProto == RelayUpstreamProtoJson {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9JsonEndpoints
		}
		resolver = NewDohJsonResolver(upstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		if len(fallbackUpstreamEndpoints_) != 0 {
			fallbackResolver = NewDohJsonResolver(fallbackUpstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		}
		if len(ExecConfig.DohConfig.FixedResolving) != 0 {
			fixedResolvers = initFixedResolvers(RelayUpstreamProtoJson, ExecConfig.DohConfig.FixedResolving)
		}
	} else if ExecConfig.DohConfig.UpstreamProto == RelayUpstreamProtoDns53 {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9Dns53Endpoints
		}
		resolver = NewDns53DnsMsgResolver(upstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		if len(fallbackUpstreamEndpoints_) != 0 {
			fallbackResolver = NewDns53DnsMsgResolver(fallbackUpstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		}
		if len(ExecConfig.DohConfig.FixedResolving) != 0 {
			fixedResolvers = initFixedResolvers(RelayUpstreamProtoDns53, ExecConfig.DohConfig.FixedResolving)
		}
	} else {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9DnsMsgEndpoints
		}
		resolver = NewDohDnsMsgResolver(upstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		if len(fallbackUpstreamEndpoints_) != 0 {
			fallbackResolver = NewDohDnsMsgResolver(fallbackUpstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		}
		if len(ExecConfig.DohConfig.FixedResolving) != 0 {
			fixedResolvers = initFixedResolvers(RelayUpstreamProtoDoh, ExecConfig.DohConfig.FixedResolving)
		}
	}
	log.Infof("resolver: %+v, fallback: %+v", upstreamEndpoints_, fallbackUpstreamEndpoints_)
	RelayAnswerer = NewDnsMsgAnswerer(resolver, fallbackResolver, fixedResolvers)
}

// initDns53RsvAnswerer initializes the DNS-over-HTTPS upstream query service.
func initDns53RsvAnswerer() {
	var upstreamEndpoints_, fallbackUpstreamEndpoints_, tmpEndpoints_ []string

	tmpEndpoints_ = strings.Split(ExecConfig.Dns53Config.Upstream, ",")
	for _, edp := range tmpEndpoints_ {
		if trimmedEdp_ := strings.TrimSpace(edp); trimmedEdp_ != "" {
			upstreamEndpoints_ = append(upstreamEndpoints_, trimmedEdp_)
		}
	}

	tmpEndpoints_ = strings.Split(ExecConfig.Dns53Config.UpstreamFallback, ",")
	for _, edp := range tmpEndpoints_ {
		if trimmedEdp_ := strings.TrimSpace(edp); trimmedEdp_ != "" {
			fallbackUpstreamEndpoints_ = append(fallbackUpstreamEndpoints_, trimmedEdp_)
		}
	}

	var resolver, fallbackResolver Resolver
	fixedResolvers := make(map[*regexp.Regexp]Resolver)
	cacheOptions_ := &CacheOptions{cacheType: ExecConfig.CacheBackend, redisURI: ExecConfig.RedisURI}
	if ExecConfig.Dns53Config.UpstreamProto == RelayUpstreamProtoJson {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9JsonEndpoints
		}
		resolver = NewDohJsonResolver(upstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		if len(fallbackUpstreamEndpoints_) != 0 {
			fallbackResolver = NewDohJsonResolver(fallbackUpstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		}
		if len(ExecConfig.Dns53Config.FixedResolving) != 0 {
			fixedResolvers = initFixedResolvers(RelayUpstreamProtoJson, ExecConfig.Dns53Config.FixedResolving)
		}
	} else if ExecConfig.Dns53Config.UpstreamProto == RelayUpstreamProtoDns53 {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9Dns53Endpoints
		}
		resolver = NewDns53DnsMsgResolver(upstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		if len(fallbackUpstreamEndpoints_) != 0 {
			fallbackResolver = NewDns53DnsMsgResolver(fallbackUpstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		}
		if len(ExecConfig.Dns53Config.FixedResolving) != 0 {
			fixedResolvers = initFixedResolvers(RelayUpstreamProtoDns53, ExecConfig.Dns53Config.FixedResolving)
		}
	} else {
		if len(upstreamEndpoints_) == 0 {
			upstreamEndpoints_ = Quad9DnsMsgEndpoints
		}
		resolver = NewDohDnsMsgResolver(upstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		if len(fallbackUpstreamEndpoints_) != 0 {
			fallbackResolver = NewDohDnsMsgResolver(fallbackUpstreamEndpoints_, ExecConfig.CacheEnabled, cacheOptions_)
		}
		if len(ExecConfig.Dns53Config.FixedResolving) != 0 {
			fixedResolvers = initFixedResolvers(RelayUpstreamProtoDoh, ExecConfig.Dns53Config.FixedResolving)
		}
	}
	log.Infof("dns53 upstream resolver: %+v, fallback: %+v", upstreamEndpoints_, fallbackUpstreamEndpoints_)
	Dns53Answerer = NewDnsMsgAnswerer(resolver, fallbackResolver, fixedResolvers)
}

func serveDohSvc(c chan error) {
	// Set Gin mode referred to loglevel.
	var err error
	if logLevel_, err := logger.ParseLevel(ExecConfig.LogLevel); err == nil && logLevel_ >= logger.DebugLevel {
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
	if ExecConfig.DohConfig.EcsIP2nd != "" {
		for _, ip_ := range strings.Split(ExecConfig.DohConfig.EcsIP2nd, ",") {
			dohHandler.AppendDefaultECSIPStr(ip_)
		}
	}

	if ExecConfig.Dns53Config.EcsIP1st != "" {
		for _, ip_ := range strings.Split(ExecConfig.DohConfig.EcsIP1st, ",") {
			dohHandler.InsertDefaultECSIPStr(ip_)
		}
	}

	// Routes.
	router_.GET(ExecConfig.DohConfig.Path, dohHandler.DohGetHandler)
	router_.GET("/checkip", func(context *gin.Context) {
		_, err = context.Writer.WriteString(context.ClientIP())
	})
	router_.POST(ExecConfig.DohConfig.Path, dohHandler.DohPostHandler)

	listenAddr_ := DefaultDohListen
	if ExecConfig.DohConfig.Listen != "" && !ListenAddrPortAvailable(ExecConfig.DohConfig.Listen) {
		c <- fmt.Errorf("doh listen config invalid: %s", ExecConfig.DohConfig.Listen)
		return
	} else {
		listenAddr_ = ExecConfig.DohConfig.Listen
	}
	if ExecConfig.DohConfig.UseTls {
		if !PathExists(ExecConfig.DohConfig.TLSCertFile) || !PathExists(ExecConfig.DohConfig.TLSKeyFile) {
			c <- fmt.Errorf("missing tls cert or key")
			return
		}
		err = router_.RunTLS(listenAddr_,
			ExecConfig.DohConfig.TLSCertFile,
			ExecConfig.DohConfig.TLSKeyFile,
		)
		c <- err
		return
	}
	err = router_.Run(listenAddr_)
	c <- err
}

func serveDns53Svc(c chan error) {
	dns53Handler := NewDns53Handler()
	if ExecConfig.Dns53Config.EcsIP2nd != "" {
		for _, ip_ := range strings.Split(ExecConfig.DohConfig.EcsIP2nd, ",") {
			dns53Handler.AppendDefaultECSIPStr(ip_)
		}
	}
	if ExecConfig.Dns53Config.UseClientIP {
		var exitIP_ string
		// Use doh relay service to add high priority exit ip.
		if ExecConfig.Dns53Config.UpstreamProto != RelayUpstreamProtoDns53 {
			upstreamURL_, err := url.Parse(ExecConfig.Dns53Config.Upstream)
			if err != nil {
				c <- err
			}
			exitIP_, err = HTTPGetString(fmt.Sprintf("%s://%s/checkip", upstreamURL_.Scheme, upstreamURL_.Host))
			if err == nil {
				log.Infof("Exit IP from checkip service of upstream doh: %s", exitIP_)
				dns53Handler.InsertDefaultECSIPStr(exitIP_)
			}
		}
		if exitIP_ == "" {
			exitIP_ = GetExitIPByResolver(Dns53Answerer.Resolver)
			if ObtainIPFromString(exitIP_) != nil {
				log.Infof("Exit IP from checkip service of thrid parties: %s", exitIP_)
				dns53Handler.InsertDefaultECSIPStr(exitIP_)
			}
		}
	} else if ExecConfig.Dns53Config.EcsIP1st != "" {
		for _, ip_ := range strings.Split(ExecConfig.Dns53Config.EcsIP1st, ",") {
			dns53Handler.InsertDefaultECSIPStr(ip_)
		}
	}

	dns.HandleFunc(".", dns53Handler.ServeDNS)
	dns53ListenAddrs_ := strings.Split(ExecConfig.Dns53Config.Listen, ",")
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
