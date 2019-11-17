package shared

import (
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"time"
)

func ReportError(req *dns.Msg, res *dns.Msg, rtt time.Duration) {
	q := req.Question[0]
	log.Debugf("❌ Failed to resolve %s %s-record (%s)", q.Name, dns.TypeToString[q.Qtype], rtt)
}

func ReportSuccess(req *dns.Msg, res *dns.Msg, rtt time.Duration) {
	if res.Rcode == dns.RcodeNameError {
		ReportError(req, res, rtt)
		return
	}

	if len(res.Answer) > 0 {
		q := req.Question[0]
		name := q.Name
		if a, ok := res.Answer[0].(*dns.A); ok {
			log.Debugf("✔ %s %s-record resolved to %s (%s)", name, dns.TypeToString[q.Qtype], a.A, rtt)
		}

		if a, ok := res.Answer[0].(*dns.AAAA); ok {
			log.Debugf("✔ %s %s-record resolved to %s (%s)", name, dns.TypeToString[q.Qtype], a.AAAA, rtt)
		}

		if a, ok := res.Answer[0].(*dns.MX); ok {
			log.Debugf("✔ %s %s-record resolved to %s (%s)", name, dns.TypeToString[q.Qtype], a.Mx, rtt)
		}

		if a, ok := res.Answer[0].(*dns.NS); ok {
			log.Debugf("✔ %s %s-record resolved to %s (%s)", name, dns.TypeToString[q.Qtype], a.Ns, rtt)
		}

		if a, ok := res.Answer[0].(*dns.PTR); ok {
			log.Debugf("✔ %s %s-record resolved to %s (%s)", name, dns.TypeToString[q.Qtype], a.Ptr, rtt)
		}
	}
}

func ReportFiltered(req *dns.Msg, be *BlockEntry) {
	name := req.Question[0].Name
	log.Debugf("⛔ %s blocked by %s list", name, be.Src)
}
