package server

import (
	"github.com/lietu/better-dns/shared"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
)

var blockedEntries = map[string]*shared.BlockEntry{}
var listEntries = map[string]int64{}
var blockListMutex = &sync.Mutex{}

func filter(req *dns.Msg) *shared.BlockEntry {
	question := req.Question[0]
	if question.Qtype != dns.TypeA && question.Qtype != dns.TypeAAAA {
		return nil
	}

	if entry, ok := blockedEntries[question.Name]; ok {
		return entry
	}

	return nil
}

func newFilteredResponse(req *dns.Msg) *dns.Msg {
	res := new(dns.Msg)
	res.SetReply(req)

	res.MsgHdr = dns.MsgHdr{
		Id:                 req.Id,
		Response:           true,
		Opcode:             dns.OpcodeQuery,
		Authoritative:      true,
		Truncated:          false,
		RecursionDesired:   true,
		RecursionAvailable: false,
		Zero:               false,
		AuthenticatedData:  false,
		CheckingDisabled:   false,
		Rcode:              dns.RcodeSuccess,
	}

	res.Compress = false

	question := req.Question[0]

	if question.Qtype == dns.TypeAAAA {
		hdr := dns.RR_Header{Name: question.Name, Rrtype: question.Qtype, Class: question.Qclass, Ttl: 2, Rdlength: 16}
		res.Answer = []dns.RR{&dns.AAAA{Hdr: hdr, AAAA: net.IPv6zero}}
	} else {
		hdr := dns.RR_Header{Name: question.Name, Rrtype: question.Qtype, Class: question.Qclass, Ttl: 2, Rdlength: 4}
		res.Answer = []dns.RR{&dns.A{Hdr: hdr, A: net.IPv4zero}}
	}

	return res
}

// Add an entry to block list
func AddBlockedEntry(name string, list string) {
	// DNS queries actually come in format "domain.name."
	if name[len(name)-1:] != "." {
		name = name + "."
	}

	// Called from multiple goroutines so making the map and list processing safe
	blockListMutex.Lock()
	defer blockListMutex.Unlock()

	blockedEntries[name] = &shared.BlockEntry{Src: list}

	old, ok := listEntries[list]
	if !ok {
		old = 0
	}
	listEntries[list] = old + 1
}

// Show current log lists
func LogLists() {
	log.Info("Blocked entries based on given lists:")
	var total int64 = 0
	for key, count := range listEntries {
		log.Infof(" - %s: %d ⛔ entries", key, count)
		total += count
	}
	log.Infof("Total %d ⛔ entries", total)
}
