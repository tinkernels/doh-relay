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

const CurrentVersion = "v0.6.0"
const DefaultRelayListenAddr = "127.0.0.1:15353"

var (
	dns53Flag = flag.Bool(
		"dns53", false, "Enable dns53 service.",
	)
	dns53ListenFlag = flag.String(
		"dns53-listen",
		"udp://:53,tcp://:53", "Set dns53 service listen port.",
	)
	dns53EDnsClientSubnetFlag = flag.String(
		"dns53-edns_client_subnet",
		"",
		"Set dns53 edns_client_subnet field.",
	)
	dns53UpstreamFlag = flag.String(
		"dns53-upstream",
		"",
		"Upstream DoH resolver for dns53 service, "+
			"e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query",
	)
	dns53UpstreamJsonFlag = flag.Bool(
		"dns53-upstream-json",
		false,
		"If dns53 upstream endpoints transfer with json format.",
	)
	relayFlag = flag.Bool(
		"relay",
		false,
		"Enable DoH relay service.",
	)
	relayListenFlag = flag.String(
		"relay-listen",
		DefaultRelayListenAddr, "Set relay service listen port.",
	)
	relayPathFlag = flag.String(
		"relay-path",
		"/dns-query",
		"DNS-over-HTTPS endpoint path.",
	)
	relayUpstreamFlag = flag.String(
		"relay-upstream",
		"",
		"Upstream DoH resolver for relay service, "+
			"e.g. https://149.112.112.11/dns-query,https://9.9.9.11/dns-query",
	)
	relayUpstreamJsonFlag = flag.Bool(
		"relay-upstream-json",
		false,
		"If relay upstream endpoints transfer with json format.",
	)
	relayTlsFlag = flag.Bool(
		"relay-tls",
		false,
		"Enable DoH relay service over TLS, default on clear http.",
	)
	relayTlsCertFlag = flag.String(
		"relay-tls-cert",
		"",
		"Specify tls cert path.",
	)
	relayTlsKeyFlag = flag.String(
		"relay-tls-key",
		"",
		"Specify tls key path.",
	)
	maxmindCityDBFileFlag = flag.String(
		"maxmind-citydb-file",
		"",
		"Specify maxmind city db file path.",
	)
	cacheFlag = flag.Bool(
		"cache",
		true,
		"Enable DoH response cache.",
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

	// Set the loglevel
	logLevel_, err := logger.ParseLevel(*logLevelFlag)
	if err != nil {
		log.Warnf("invalid log level: %v", err)
	}
	log.SetLevel(logLevel_)

	InitGeoipReader(*maxmindCityDBFileFlag)

	var (
		serveRelayErr_ error
		serveDns53Err_ error
	)
	chRelaySvc_, chDns53Svc_ := make(chan error), make(chan error)

	if *relayFlag {
		initRelayRsvAnswerer()
		go serveRelaySvc(chRelaySvc_)
	}

	if *dns53Flag {
		initDns53RsvAnswerer()
		go serveDns53Svc(chDns53Svc_)
	}

	// Exit on some signals.
	termSig_ := make(chan os.Signal)
	signal.Notify(termSig_, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-termSig_
		os.Exit(0)
	}()

	// Log services exit errors.
	if *relayFlag {
		serveRelayErr_ = <-chRelaySvc_
	}
	if *dns53Flag {
		serveDns53Err_ = <-chDns53Svc_
	}

	if serveRelayErr_ != nil {
		log.Errorf("relay service exit: %v", serveRelayErr_)
		os.Exit(1)
	}
	if serveDns53Err_ != nil {
		log.Errorf("dns53 service exit: %v", serveDns53Err_)
		os.Exit(1)
	}
	log.Infof("relay service exit: %v", serveRelayErr_)
	log.Infof("dns53 service exit: %v", serveDns53Err_)
	os.Exit(0)
}

// initRelayRsvAnswerer initializes the DNS-over-HTTPS upstream query service.
func initRelayRsvAnswerer() {
	upstreamEndpoints_ := Quad9JsonEndpoints
	if tmpEndpoints_ := strings.Split(*relayUpstreamFlag, ","); *relayUpstreamFlag != "" &&
		len(tmpEndpoints_) > 0 {
		upstreamEndpoints_ = make([]string, len(tmpEndpoints_))
		for i_ := range tmpEndpoints_ {
			upstreamEndpoints_[i_] = strings.TrimSpace(tmpEndpoints_[i_])
		}
	}
	var resolver DohResolver
	if *relayUpstreamJsonFlag {
		resolver = NewJsonResolver(upstreamEndpoints_, *cacheFlag)
	} else {
		resolver = NewDnsMsgResolver(upstreamEndpoints_, *cacheFlag)
	}
	RelayAnswerer = NewDnsMsgAnswerer(resolver)
}

// initDns53RsvAnswerer initializes the DNS-over-HTTPS upstream query service.
func initDns53RsvAnswerer() {
	upstreamEndpoints_ := Quad9DnsMsgEndpoints
	if tmpEndpoints_ := strings.Split(*dns53UpstreamFlag, ","); *dns53UpstreamFlag != "" &&
		len(tmpEndpoints_) > 0 {
		upstreamEndpoints_ = make([]string, len(tmpEndpoints_))
		for i_ := range tmpEndpoints_ {
			upstreamEndpoints_[i_] = strings.TrimSpace(tmpEndpoints_[i_])
		}
	}
	var resolver DohResolver
	if *dns53UpstreamJsonFlag {
		resolver = NewJsonResolver(upstreamEndpoints_, *cacheFlag)
	} else {
		resolver = NewDnsMsgResolver(upstreamEndpoints_, *cacheFlag)
	}
	Dns53Answerer = NewDnsMsgAnswerer(resolver)
}

func serveRelaySvc(c chan error) {
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

	// Routes.
	router_.GET(*relayPathFlag, DohGetHandler)
	router_.POST(*relayPathFlag, DohPostHandler)

	listenAddr_ := DefaultRelayListenAddr
	if ListenAddrPortAvailable(*relayListenFlag) {
		listenAddr_ = *relayListenFlag
	}
	if *relayTlsFlag {
		if !PathExists(*relayTlsCertFlag) || !PathExists(*relayTlsKeyFlag) {
			c <- fmt.Errorf("missing tls cert or key")
			return
		}
		err = router_.RunTLS(listenAddr_,
			*relayTlsCertFlag,
			*relayTlsKeyFlag,
		)
		c <- err
		return
	}
	err = router_.Run(listenAddr_)
	c <- err
}

func serveDns53Svc(c chan error) {
	dns.HandleFunc(".", NewDns53Handler(*dns53EDnsClientSubnetFlag).ServeDNS)
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
		if strings.ToLower(url_.Scheme) != "udp" {
			c_ := make(chan error)
			dns53CHs_ = append(dns53CHs_, c_)
			go serveDns53UDP(url_.Host, c_)
			log.Infof("dns53 listening on %s", url_.String())
		} else if strings.ToLower(url_.Scheme) != "tcp" {
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
