package server

import (
	"github.com/lietu/better-dns/client"
	"github.com/lietu/better-dns/shared"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
)

type RequestHandler struct {
}

func getResult(req *dns.Msg) *dns.Msg {
	if cached := cache(req); cached != nil {
		return cached
	}

	var res *dns.Msg
	if filtered := filter(req); filtered != nil {
		go shared.ReportFiltered(req, filtered)
		res = newFilteredResponse(req)
	} else {
		res = client.Query(req)
		setCache(req, res)
	}

	return res
}

func (h *RequestHandler) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	res := getResult(req)

	if res != nil {
		err := w.WriteMsg(res)
		if err != nil {
			log.Errorf("Error while responding to request (cached): %s", err)
		}
	} else {
		res = new(dns.Msg)
		res.Rcode = dns.RcodeServerFailure

		err := w.WriteMsg(res)
		if err != nil {
			log.Errorf("Error while responding to request (cached): %s", err)
		}
	}
}

// Return a request handler for the DNS server
func NewHandler() *RequestHandler {
	h := &RequestHandler{}
	return h
}
