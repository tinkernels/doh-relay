package main

import (
	"github.com/miekg/dns"
	"strings"
)

type Dns53Handler struct {
	DefaultECSIPs []string
}

func NewDns53Handler() (h *Dns53Handler) {
	h = &Dns53Handler{
		DefaultECSIPs: make([]string, 0),
	}
	return
}

func (h *Dns53Handler) AppendDefaultECSIPStr(ipStr string) {
	if ip := ObtainIPFromString(ipStr); ip != nil &&
		!SliceContains(h.DefaultECSIPs, ip.String()) &&
		!IsPrivateIP(ip) {

		h.DefaultECSIPs = append(h.DefaultECSIPs, ip.String())
	}
}

func (h *Dns53Handler) InsertDefaultECSIPStr(ipStr string) {
	if ip := ObtainIPFromString(ipStr); ip != nil &&
		!SliceContains(h.DefaultECSIPs, ip.String()) &&
		!IsPrivateIP(ip) {

		h.DefaultECSIPs = append([]string{ip.String()}, h.DefaultECSIPs...)
	}
}

func (h *Dns53Handler) responseEmpty(w dns.ResponseWriter, msgReq *dns.Msg) {
	msgReq.Response = false
	msgReq.Rcode = dns.RcodeRefused
	err := w.WriteMsg(msgReq)
	if err != nil {
		return
	}
	return
}

func (h *Dns53Handler) ServeDNS(w dns.ResponseWriter, msgReq *dns.Msg) {
	// Ignore AAAA Question when configured to not answer
	if len(msgReq.Question) > 0 && msgReq.Question[0].Qtype == dns.TypeAAAA && !ExecConfig.IPv6Answer {
		h.responseEmpty(w, msgReq)
		return
	}

	var tryEcsIPs_ []string
	defer func() { tryEcsIPs_ = nil }()

	// ECS in request dns message.
	ecs_ := ObtainECS(msgReq)
	if ecs_ != nil && ecs_.Address != nil && !IsPrivateIP(ecs_.Address) {
		tryEcsIPs_ = append(tryEcsIPs_, ecs_.Address.String())
	}
	tryEcsIPs_ = append(tryEcsIPs_, h.DefaultECSIPs...)

	msgRsp_, err := Dns53Answerer.Answer(msgReq, strings.Join(tryEcsIPs_, ","))
	defer func() { msgRsp_ = nil }()
	if err != nil {
		log.Error(err)
		h.responseEmpty(w, msgReq)
		return
	}
	// Restore request ECS.
	if ecs_ == nil {
		RemoveECSInDnsMsg(msgRsp_)
	} else {
		ChangeECSInDnsMsg(msgRsp_, &ecs_.Address)
	}
	err = w.WriteMsg(msgRsp_)
	if err != nil {
		log.Error(err)
		return
	}
}
