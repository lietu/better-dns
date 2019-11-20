package server

import (
	"fmt"
	lru "github.com/hashicorp/golang-lru"
	"github.com/miekg/dns"
	"time"
)

const CACHE_SIZE = 2048
const MIN_TTL = 30

var cache, _ = lru.New2Q(CACHE_SIZE) // This is thread-safe

type CachedItem struct {
	res     *dns.Msg
	expires time.Time
}

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

	// Totally breaking DNS standards and caching for a bit longer than necessary because it seems nice
	if ttl < MIN_TTL {
		ttl = MIN_TTL
	}

	key := getEntryName(req)
	if key != "" {
		// log.Debugf("Could cache response for %ds", ttl)
		item := &CachedItem{}
		item.res = res
		item.expires = time.Now().Add(time.Second * time.Duration(ttl))
		cache.Add(key, item)
	}
}

func getEntryName(r *dns.Msg) string {
	if len(r.Question) > 0 {
		q := r.Question[0]
		t := dns.TypeToString[q.Qtype]
		n := q.Name

		return fmt.Sprintf("%s:%s", n, t)
	}

	return ""
}

func getCache(req *dns.Msg) *dns.Msg {
	key := getEntryName(req)
	if cached, ok := cache.Get(key); ok {
		cached := cached.(*CachedItem)

		if cached.expires.After(time.Now()) {
			res := cached.res.Copy()
			res.SetReply(req)
			return res
		} else {
			cache.Remove(key)
		}
	}

	return nil
}
