package main

import (
	"github.com/miekg/dns"
	"strings"
)

type Dns53Handler struct {
	ECSIPs []string
}

func NewDns53Handler() (h *Dns53Handler) {
	h = &Dns53Handler{
		ECSIPs: make([]string, 0),
	}
	exitIP_ := GetExitIPByResolver(Dns53Answerer.Resolver)
	log.Infof("Exit IP: %s", exitIP_)
	h.InsertECSIPStr(exitIP_)
	return
}

func (h *Dns53Handler) AppendECSIPStr(ipStr string) {
	if strings.TrimSpace(ipStr) == "" {
		return
	}
	h.ECSIPs = append(h.ECSIPs, ipStr)
}

func (h *Dns53Handler) InsertECSIPStr(ipStr string) {
	if strings.TrimSpace(ipStr) == "" {
		return
	}
	h.ECSIPs = append([]string{ipStr}, h.ECSIPs...)
}

func (h *Dns53Handler) ServeDNS(w dns.ResponseWriter, msgReq *dns.Msg) {
	msgRsp_, err := Dns53Answerer.Answer(msgReq, strings.Join(RemoveSliceDuplicate(h.ECSIPs), ","))
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
