package main

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
	"net/http"
	"strings"
)

type DohHandler struct {
	DefaultECSIPs []string
}

func NewDohHandler() (h *DohHandler) {
	h = &DohHandler{
		DefaultECSIPs: make([]string, 0),
	}
	return
}

func (h *DohHandler) AppendDefaultECSIPStr(ipStr string) {
	if ip := ObtainIPFromString(ipStr); ip != nil && !SliceContains(h.DefaultECSIPs, ip.String()) {
		h.DefaultECSIPs = append(h.DefaultECSIPs, ip.String())
	}
}

func (h *DohHandler) InsertDefaultECSIPStr(ipStr string) {
	if ip := ObtainIPFromString(ipStr); ip != nil && !SliceContains(h.DefaultECSIPs, ip.String()) {
		h.DefaultECSIPs = append([]string{ip.String()}, h.DefaultECSIPs...)
	}
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
	var tryEcsIPs_ []string
	defer func() { tryEcsIPs_ = nil }()

	// ECS in request dns message.
	ecs_ := ObtainECS(msgReq)
	if ecs_ != nil && ecs_.Address != nil {
		tryEcsIPs_ = append(tryEcsIPs_, ecs_.Address.String())
	}

	// Custom Header for specifying EDNS-Client-Subnet.
	if s_ := strings.TrimSpace(c.GetHeader("X-EDNS-Client-Subnet")); s_ != "" {
		for _, s := range strings.Split(s_, ",") {
			if ip := ObtainIPFromString(s); ip != nil && !SliceContains(tryEcsIPs_, ip.String()) {
				tryEcsIPs_ = append(tryEcsIPs_, ip.String())
			}
		}
	}
	// Client IP
	if !SliceContains(tryEcsIPs_, c.ClientIP()) {
		tryEcsIPs_ = append(tryEcsIPs_, c.ClientIP())
	}
	tryEcsIPs_ = append(tryEcsIPs_, h.DefaultECSIPs...)

	log.Debugf("edns_client_subnet param is %+v", tryEcsIPs_)
	msgRsp_ := new(dns.Msg)
	defer func() { msgRsp_ = nil }()
	msgRsp_, err := RelayAnswerer.Answer(msgReq, strings.Join(tryEcsIPs_, ","))
	defer func() { msgRsp_ = nil }()
	if err != nil || msgRsp_ == nil {
		log.Error(err)
		return
	}
	// Restore request ECS.
	if ecs_ == nil {
		RemoveECSInDnsMsg(msgRsp_)
	} else {
		ChangeECSInDnsMsg(msgRsp_, &ecs_.Address)
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
