package shared

import (
	"fmt"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"sort"
	"time"
)

func answerResult(a dns.RR) string {
	if a, ok := a.(*dns.A); ok {
		return a.A.String()
	}

	if a, ok := a.(*dns.AAAA); ok {
		return a.AAAA.String()
	}

	if a, ok := a.(*dns.MX); ok {
		return a.Mx
	}

	if a, ok := a.(*dns.NS); ok {
		return a.Ns
	}

	if a, ok := a.(*dns.PTR); ok {
		return a.Ptr
	}

	if a, ok := a.(*dns.CNAME); ok {
		return a.Target
	}

	return fmt.Sprintf("unknown (%T)", a)
}

func ReportError(req *dns.Msg, res *dns.Msg, rtt time.Duration, err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Suppressing panic during ReportError: %s", err)
		}
	}()

	q := req.Question[0]
	c := "nil"
	if res != nil {
		c = dns.RcodeToString[res.Rcode]
	} else if err != nil {
		c = fmt.Sprintf("error: %s", err)
	}
	log.Debugf("❌ %s (%s) not resolved (%s) %s", CleanName(q.Name), dns.TypeToString[q.Qtype], CleanDuration(rtt), c)
}

func ReportSuccess(req *dns.Msg, res *dns.Msg, rtt time.Duration, server string) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Suppressing panic during ReportSuccess: %s", err)
		}
	}()

	if res.Rcode == dns.RcodeNameError {
		ReportError(req, res, rtt, nil)
		return
	}

	answers := len(res.Answer)
	if answers == 0 {
		return
	}

	// TODO: Track per-server stats

	var ttl time.Duration = 0
	q := req.Question[0]
	t := dns.TypeToString[q.Qtype]
	name := q.Name

	answerList := []dns.RR{}
	for _, a := range res.Answer {
		answerList = append(answerList, a)
	}

	// Sort A records first, then AAAA, then rest
	sort.Slice(answerList[:], func(i, j int) bool {
		first := answerList[i]
		second := answerList[j]

		_, firstA := first.(*dns.A)
		_, firstAAAA := first.(*dns.AAAA)
		_, secondA := second.(*dns.A)
		_, secondAAAA := second.(*dns.AAAA)

		if firstA {
			return true
		} else if secondA {
			return false
		} else if firstAAAA {
			return true
		} else if secondAAAA {
			return false
		}

		return false // Doesn't really matter
	})

	extra := ""
	ans := answerList[0]
	ttl = time.Second * time.Duration(ans.Header().Ttl)
	extra = fmt.Sprintf(" (+%d more)", answers-1)
	result := answerResult(ans)

	log.Debugf("✔ %s (%s) is %s%s for %s (%s)", CleanName(name), t, result, extra, CleanDuration(ttl), CleanDuration(rtt))
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
		extra = fmt.Sprintf(" (+%d more)", answers-1)
	}

	log.Debugf("✔ %s (%s) is %s%s (cached)", CleanName(name), t, result, extra)
}

func ReportFiltered(req *dns.Msg, be *BlockEntry) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("Suppressing panic during ReportFiltered: %s", err)
		}
	}()

	name := req.Question[0].Name
	log.Debugf("⛔ %s blocked by %s", CleanName(name), be.Src)
}

func CleanDuration(d time.Duration) time.Duration {
	return d.Truncate(time.Millisecond)
}

func CleanName(name string) string {
	return name[0 : len(name)-1]
}
