package server

import (
	"github.com/miekg/dns"
)

// TODO: All caching logic

func setCache(req *dns.Msg, res *dns.Msg) {
	var ttl uint32 = 0

	for i := range res.Answer {
		answerTTL := res.Answer[i].Header().Ttl
		if ttl == 0 {
			ttl = answerTTL
		} else if ttl > answerTTL {
			ttl = answerTTL
		}
	}

	// log.Debugf("Could cache response for %ds", ttl)
}

func cache(r *dns.Msg) *dns.Msg {
	return nil
}
