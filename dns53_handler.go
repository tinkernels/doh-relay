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
	h.InsertDefaultECSIPStr(exitIP_)
	return
}

func (h *Dns53Handler) AppendDefaultECSIPStr(ipStr string) {
	if SliceContains(h.DefaultECSIPs, ipStr) || ObtainIPFromString(ipStr) == nil {
		return
	}
	h.DefaultECSIPs = append(h.DefaultECSIPs, ipStr)
}

func (h *Dns53Handler) InsertDefaultECSIPStr(ipStr string) {
	if SliceContains(h.DefaultECSIPs, ipStr) || ObtainIPFromString(ipStr) == nil {
		return
	}
	h.DefaultECSIPs = append([]string{ipStr}, h.DefaultECSIPs...)
}

func (h *Dns53Handler) ServeDNS(w dns.ResponseWriter, msgReq *dns.Msg) {
	var tryEcsIPs_ []string

	// ECS in request dns message.
	if ecs_ := ObtainECS(msgReq); ecs_ != nil && ecs_.Address != nil {
		tryEcsIPs_ = append(tryEcsIPs_, ecs_.Address.String())
	}
	tryEcsIPs_ = append(tryEcsIPs_, h.DefaultECSIPs...)

	msgRsp_, err := Dns53Answerer.Answer(msgReq, strings.Join(tryEcsIPs_, ","))
	defer func() { msgRsp_ = nil }()
	if err != nil {
		log.Error(err)
		return
	}
	err = w.WriteMsg(msgRsp_)
	if err != nil {
		log.Error(err)
		return
	}
}
