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
	exitIP_ := GetExitIPByResolver(Dns53Answerer.Resolver)
	if ObtainIPFromString(exitIP_) == nil || SliceContains(h.DefaultECSIPs, exitIP_) {
		log.Errorf("Failed to get exit IP address")
		return
	}
	log.Infof("Exit IP: %s", exitIP_)
	h.AppendDefaultECSIPStr(exitIP_)
	return
}

func (h *Dns53Handler) AppendDefaultECSIPStr(ipStr string) {
	if ip := ObtainIPFromString(ipStr); ip != nil && !SliceContains(h.DefaultECSIPs, ip.String()) {
		h.DefaultECSIPs = append(h.DefaultECSIPs, ip.String())
	}
}

func (h *Dns53Handler) InsertDefaultECSIPStr(ipStr string) {
	if ip := ObtainIPFromString(ipStr); ip != nil && !SliceContains(h.DefaultECSIPs, ip.String()) {
		h.DefaultECSIPs = append([]string{ip.String()}, h.DefaultECSIPs...)
	}
}

func (h *Dns53Handler) ServeDNS(w dns.ResponseWriter, msgReq *dns.Msg) {
	var tryEcsIPs_ []string
	defer func() { tryEcsIPs_ = nil }()

	// ECS in request dns message.
	ecs_ := ObtainECS(msgReq)
	if ecs_ != nil && ecs_.Address != nil {
		tryEcsIPs_ = append(tryEcsIPs_, ecs_.Address.String())
	}
	tryEcsIPs_ = append(tryEcsIPs_, h.DefaultECSIPs...)

	msgRsp_, err := Dns53Answerer.Answer(msgReq, strings.Join(tryEcsIPs_, ","))
	defer func() { msgRsp_ = nil }()
	if err != nil {
		log.Error(err)
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
