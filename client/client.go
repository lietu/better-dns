package client

import (
	"crypto/tls"
	"github.com/lietu/better-dns/shared"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"io"
)

var client *dns.Client

// Do a DNS query for the given request
func Query(req *dns.Msg) *dns.Msg {
	// TODO: Parallelization of multiple queries
	// TODO: Configuration

	if client == nil {
		client = &dns.Client{}
		client.Net = "tcp-tls"
		client.TLSConfig = &tls.Config{ServerName: "cloudflare-dns.com"}
	}

	port := "853"
	server := "1.1.1.1:" + port

	retries := 3
	for retries > 0 {
		retries -= 1

		res, rtt, err := client.Exchange(req, server)

		if err != nil {
			go shared.ReportError(req, res, rtt)

			if err.Error() == "tls: DialWithDialer timed out" {
				log.Infof("TLS connection timed out.")
				continue
			}

			log.Errorf("Caught error while querying: %s", err)
			if err == io.EOF && client.Net == "tcp-tls" && port != "853" {
				log.Errorf("Maybe you're trying tcp-tls DNS to a non-TLS server? You might need to use port 853.")
			}
		} else {
			go shared.ReportSuccess(req, res, rtt)
			return res
		}
	}

	return nil
}
