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
	if cached := getCache(req); cached != nil {
		go shared.ReportCached(req, cached)
		return cached
	}

	var res *dns.Msg
	if filtered := filter(req); filtered != nil {
		go shared.ReportFiltered(req, filtered)
		res = newFilteredResponse(req)
	} else {
		res = client.Query(req)
		if res != nil {
			// TODO: Check for blocked results in reply in case of CNAME entries
			setCache(req, res)
		}
	}

	return res
}

func writeResponse(w dns.ResponseWriter, res *dns.Msg) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Caught panic while responding to request: %s", err)
		}
	}()

	err := w.WriteMsg(res)
	if err != nil {
		log.Errorf("Error while responding to request: %s", err)
	}
}

func (h *RequestHandler) ServeDNS(w dns.ResponseWriter, req *dns.Msg) {
	defer func() {
		if err := recover(); err != nil {
			// Unknown panic, should probably recover DNS settings and crash
			log.Errorf("Caught panic %s - exiting...", err)
			shared.RestoreDnsServers()
			panic(err)
		}
	}()

	res := getResult(req)

	if res != nil {
		writeResponse(w, res)
	} else {
		res = new(dns.Msg)
		res.SetReply(req)
		res.Rcode = dns.RcodeServerFailure

		writeResponse(w, res)
	}
}

// Return a request handler for the DNS server
func NewHandler() *RequestHandler {
	h := &RequestHandler{}
	return h
}
