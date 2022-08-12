package main

import (
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"net/http"
)

func ResponseError(c *gin.Context, msgReq *dns.Msg) {
	msgRsp_ := new(dns.Msg)
	msgRsp_.SetReply(msgReq)
	msgRsp_.Rcode = dns.RcodeServerFailure
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
