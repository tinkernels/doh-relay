package main

import "github.com/miekg/dns"

type Dns53Handler struct {
	EDNSSubnet string
}

func NewDns53Handler(eDnsSubnet string) (h *Dns53Handler) {
	h = &Dns53Handler{
		EDNSSubnet: eDnsSubnet,
	}
	return
}

func (h *Dns53Handler) ServeDNS(w dns.ResponseWriter, msgReq *dns.Msg) {
	msgRsp_, err := Dns53Answerer.Answer(msgReq, h.EDNSSubnet)
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
