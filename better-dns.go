package main

import (
	"bufio"
	"github.com/lietu/better-dns/server"
	"github.com/lietu/better-dns/shared"
	"github.com/mattn/go-colorable"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// TODO: Use config
const PORT = 53

func blockFromURL(listURL string) {
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
				server.AddBlockedEntry(name, listURL)
			} else {
				log.Debugf("Ignoring entry: %s", entry)
			}
		} else if len(parts) == 1 {
			server.AddBlockedEntry(parts[0], listURL)
		} else {
			log.Debugf("Unrecognized entry: %s", entry)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Errorf("Error while processing list %s: %s", listURL, err)
	}

	log.Infof("✔ Parsed %s list in %s", listURL, time.Since(start))
}

func loadLists() {
	urls := []string{
		"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
		"https://mirror1.malwaredomains.com/files/justdomains",
		"http://sysctl.org/cameleon/hosts",
		"https://s3.amazonaws.com/lists.disconnect.me/simple_tracking.txt",
		"https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
		"https://hosts-file.net/ad_servers.txt",
		"https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts&showintro=0&mimetype=plaintext",
	}

	for i := range urls {
		blockFromURL(urls[i])
	}

	server.LogLists()
}

func main() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.TextFormatter{ForceColors: true})
	log.SetOutput(colorable.NewColorableStdout())

	shared.RememberDnsServers()
	loadLists()

	handler := server.NewHandler()
	port := strconv.Itoa(PORT)

	go func() {
		log.Infof("Listening to UDP 127.0.0.1:%s", port)
		err := dns.ListenAndServe("127.0.0.1:"+port, "udp", handler)
		if err != nil {
			log.Panicf("Could not listen to UDP port: %s", err)
		}
	}()

	go func() {
		log.Infof("Listening to TCP 127.0.0.1:%s", port)
		err := dns.ListenAndServe("127.0.0.1:"+port, "tcp", handler)
		if err != nil {
			log.Panicf("Could not listen to TCP port: %s", err)
		}
	}()

	shared.UpdateDnsServers()

	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig

	shared.RestoreDnsServers()

	log.Fatalf("Signal (%v) received, stopping", s)
}
