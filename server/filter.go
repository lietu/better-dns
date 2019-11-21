package server

import (
	"bufio"
	"github.com/lietu/better-dns/shared"
	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

var blockedEntries = map[string]*shared.BlockEntry{}
var listEntries = map[string]int64{}
var blockListMutex = &sync.Mutex{}
var blackListEntry = &shared.BlockEntry{Src: "blacklist"}

func filter(req *dns.Msg, blacklist []string) *shared.BlockEntry {
	question := req.Question[0]
	if question.Qtype != dns.TypeA && question.Qtype != dns.TypeAAAA {
		return nil
	}

	if entry, ok := blockedEntries[question.Name]; ok {
		return entry
	}

	for _, pattern := range blacklist {
		if glob.Glob(pattern, question.Name) {
			return blackListEntry
		}
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

func BlockFromURL(listURL string) {
	start := time.Now()

	req, err := http.NewRequest("GET", listURL, nil)
	if err != nil {
		log.Errorf("Failed to create request to %s: %s", listURL, err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("Failed to request %s: %s", listURL, err)
		return
	}

	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Errorf("Error closing request body: %s", err)
		}
	}()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		entry := strings.TrimSpace(scanner.Text())

		// Skip clearly unnecessary lines
		if entry == "" || entry[0:1] == "#" {
			continue
		}

		// Strip away anything following # (comments)
		commentParts := strings.SplitN(entry, "#", 2)
		entry = strings.TrimSpace(commentParts[0])

		parts := strings.Split(entry, " ")
		if len(parts) == 2 {
			target := parts[0]
			name := parts[1]

			// Blackhole targets
			if target == "0.0.0.0" || target == "::1" || target == ":::1" || target == "255.255.255.255" || (len(target) >= 4 && target[0:4] == "127.") {
				AddBlockedEntry(name, listURL)
			} else {
				log.Debugf("Ignoring entry: %s", entry)
			}
		} else if len(parts) == 1 {
			AddBlockedEntry(parts[0], listURL)
		} else {
			log.Debugf("Unrecognized entry: %s", entry)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("Error while processing list %s: %s", listURL, err)
	}

	log.Infof("✔ Parsed %s list in %s", listURL, time.Since(start))
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
