package shared

import (
	"fmt"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"time"
)

func answerResult(a dns.RR) string {
	if a, ok := a.(*dns.A); ok {
		return fmt.Sprintf("%s", a.A)
	}

	if a, ok := a.(*dns.AAAA); ok {
		return fmt.Sprintf("%s", a.AAAA)
	}

	if a, ok := a.(*dns.MX); ok {
		return fmt.Sprintf("%s", a.Mx)
	}

	if a, ok := a.(*dns.NS); ok {
		return fmt.Sprintf("%s", a.Ns)
	}

	if a, ok := a.(*dns.PTR); ok {
		return fmt.Sprintf("%s", a.Ptr)
	}

	if a, ok := a.(*dns.CNAME); ok {
		return fmt.Sprintf("%s", a.Target)
	}

	return fmt.Sprintf("unknown (%T)", a)
}

func ReportError(req *dns.Msg, res *dns.Msg, rtt time.Duration) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Suppressing panic during ReportError: %s", err)
		}
	}()

	q := req.Question[0]
	c := dns.RcodeToString[res.Rcode]
	log.Debugf("❌ Failed to resolve %s %s-record (%s) %s", q.Name, dns.TypeToString[q.Qtype], rtt, c)
}

func ReportSuccess(req *dns.Msg, res *dns.Msg, rtt time.Duration) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Suppressing panic during ReportSuccess: %s", err)
		}
	}()

	if res.Rcode == dns.RcodeNameError {
		ReportError(req, res, rtt)
		return
	}

	answers := len(res.Answer)
	if answers > 0 {
		q := req.Question[0]
		t := dns.TypeToString[q.Qtype]
		name := q.Name
		ans := res.Answer[0]
		result := answerResult(ans)
		ttl := time.Second * time.Duration(ans.Header().Ttl)

		extra := ""
		if answers > 1 {
			extra = fmt.Sprintf(" (and %d more)", answers - 1)
		}

		log.Debugf("✔ %s %s-record resolved to %s%s (%s) TTL %s", name, t, result, extra, rtt, ttl)
	}
}

func ReportCached(req *dns.Msg, res *dns.Msg) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Suppressing panic during ReportCached: %s", err)
		}
	}()

	q := req.Question[0]
	name := q.Name
	t := dns.TypeToString[q.Qtype]

	result := "none"
	answers := len(res.Answer)
	if answers > 0 {
		result = answerResult(res.Answer[0])
	}

	extra := ""
	if answers > 1 {
		extra = fmt.Sprintf(" (and %d more)", answers - 1)
	}

	log.Debugf("✔ %s %s-record resolved to %s%s (cached)", name, t, result, extra)
}

func ReportFiltered(req *dns.Msg, be *BlockEntry) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Suppressing panic during ReportFiltered: %s", err)
		}
	}()

	name := req.Question[0].Name
	log.Debugf("⛔ %s blocked by %s list", name, be.Src)
}
