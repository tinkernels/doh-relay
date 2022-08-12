package main

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"net/http"
	"strings"
)

func DohGetHandler(c *gin.Context) {
	dnsQParam_ := c.Query("dns")
	eDnsClientSubnet_ := ""
	if s_ := strings.TrimSpace(dnsQParam_); s_ == "" {
		log.Error("dns param is empty")
		return
	}
	// Custom Header for specifying EDNS-Client-Subnet.
	if s_ := strings.TrimSpace(c.GetHeader("X-EDNS-Client-Subnet")); s_ != "" {
		eDnsClientSubnet_ = s_
	} else {
		eDnsClientSubnet_ = c.ClientIP()
	}
	log.Debugf("edns_client_subnet param is %v", eDnsClientSubnet_)
	msgReqBytes_, err := base64.RawURLEncoding.DecodeString(dnsQParam_)
	if err != nil {
		log.Error(err)
		return
	}
	msgReq_, msgRsp_ := new(dns.Msg), new(dns.Msg)
	err = msgReq_.Unpack(msgReqBytes_)
	if err != nil {
		log.Error(err)
		return
	}
	msgRsp_, err = RelayAnswerer.Answer(msgReq_, eDnsClientSubnet_)
	if err != nil || msgRsp_ == nil {
		log.Error(err)
		return
	}
	msgRspBytes_, err := msgRsp_.Pack()
	if err != nil {
		log.Error(err)
		return
	}
	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/dns-message")
	_, err = c.Writer.Write(msgRspBytes_)
	if err != nil {
		log.Error(err)
		return
	}
}
