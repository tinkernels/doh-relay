package main

import (
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"net/http"
	"strings"
)

func DohPostHandler(c *gin.Context) {
	data_, err := c.GetRawData()
	eDnsClientSubnet_ := ""
	if err != nil {
		log.Error(err)
		return
	}
	// Custom Header for specifying EDNS-Client-Subnet.
	if s_ := strings.TrimSpace(c.GetHeader("X-EDNS-Client-Subnet")); s_ != "" {
		eDnsClientSubnet_ = s_
	} else {
		eDnsClientSubnet_ = c.ClientIP()
	}
	log.Debugf("edns_client_subnet param is %v", eDnsClientSubnet_)
	msgReq_, msgRsp_ := new(dns.Msg), new(dns.Msg)
	err = msgReq_.Unpack(data_)
	if err != nil {
		log.Error(err)
		return
	}
	if *relay2ndECSFlag != "" {
		eDnsClientSubnet_ = strings.Join(append([]string{eDnsClientSubnet_}, *relay2ndECSFlag), ",")
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
