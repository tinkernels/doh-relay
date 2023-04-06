package main

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"net/http"
	"strings"
)

type DohHandler struct {
	ECSIPs []string
}

func NewDohHandler() (h *DohHandler) {
	h = &DohHandler{
		ECSIPs: make([]string, 0),
	}
	return
}

func (h *DohHandler) AppendECSIPStr(ipStr string) {
	if strings.TrimSpace(ipStr) == "" {
		return
	}
	h.ECSIPs = append(h.ECSIPs, ipStr)
}

func (h *DohHandler) InsertECSIPStr(ipStr string) {
	if strings.TrimSpace(ipStr) == "" {
		return
	}
	h.ECSIPs = append([]string{ipStr}, h.ECSIPs...)
}

func (h *DohHandler) DohGetHandler(c *gin.Context) {
	dnsQParam_ := c.Query("dns")
	if s_ := strings.TrimSpace(dnsQParam_); s_ == "" {
		log.Error("dns param is empty")
		return
	}

	msgReqBytes_, err := base64.RawURLEncoding.DecodeString(dnsQParam_)
	defer func() { msgReqBytes_ = nil }()
	if err != nil {
		log.Error(err)
		return
	}
	msgReq_ := new(dns.Msg)
	defer func() { msgReq_ = nil }()
	err = msgReq_.Unpack(msgReqBytes_)
	if err != nil {
		log.Error(err)
		return
	}
	h.doDohResponse(c, msgReq_)
}

func (h *DohHandler) DohPostHandler(c *gin.Context) {
	data_, err := c.GetRawData()
	if err != nil {
		log.Error(err)
		return
	}
	msgReq_ := new(dns.Msg)
	defer func() { msgReq_ = nil }()
	err = msgReq_.Unpack(data_)
	if err != nil {
		log.Error(err)
		return
	}
	h.doDohResponse(c, msgReq_)
}

func (h *DohHandler) doDohResponse(c *gin.Context, msgReq *dns.Msg) {

	h.InsertECSIPStr(c.ClientIP())
	// Custom Header for specifying EDNS-Client-Subnet.
	if s_ := strings.TrimSpace(c.GetHeader("X-EDNS-Client-Subnet")); s_ != "" {
		for _, s := range strings.Split(s_, ",") {
			trimmedS_ := strings.TrimSpace(s)
			if trimmedS_ != "" {
				h.InsertECSIPStr(trimmedS_)
			}
		}
	}

	log.Debugf("edns_client_subnet param is %+v", h.ECSIPs)
	msgRsp_ := new(dns.Msg)
	defer func() { msgRsp_ = nil }()
	msgRsp_, err := RelayAnswerer.Answer(msgReq, strings.Join(RemoveSliceDuplicate(h.ECSIPs), ","))
	defer func() { msgRsp_ = nil }()
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
