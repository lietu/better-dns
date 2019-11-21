package client

import (
	"bytes"
	"crypto/tls"
	"errors"
	"github.com/lietu/better-dns/shared"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

var DnsOverHttpsRequestError = errors.New("DNS-over-HTTPS server did not respond with 200 OK")

type queryResult struct {
	res    *dns.Msg
	rtt    time.Duration
	server string
}

var nullResult = queryResult{
	res:    nil,
	rtt:    0,
	server: "",
}

var httpClients = map[string]*http.Client{}

func queryDnsOverTls(req *dns.Msg, host string, serverName string) queryResult {
	server := host + ":853"

	tlsClient := &dns.Client{}
	tlsClient.Net = "tcp-tls"
	tlsClient.TLSConfig = &tls.Config{ServerName: serverName}

	retries := 3
	for retries > 0 {
		retries -= 1

		res, rtt, err := tlsClient.Exchange(req, server)

		if err != nil {
			go shared.ReportError(req, res, rtt, err)

			if err.Error() == "tls: DialWithDialer timed out" {
				log.Debug("TLS connection timed out.")
				continue
			}

			log.Debugf("Caught error while querying: %s", err)
		} else {
			return queryResult{
				res: res,
				rtt: rtt,
			}
		}
	}

	return queryResult{
		res: nil,
		rtt: 0,
	}
}

func queryDns(req *dns.Msg, server string) queryResult {
	client := &dns.Client{}
	client.Net = "udp"

	server = server + ":53"

	retries := 3
	for retries > 0 {
		retries -= 1

		res, rtt, err := client.Exchange(req, server)

		if err != nil {
			go shared.ReportError(req, res, rtt, err)
			log.Debugf("Caught error while querying: %s", err)
		} else {
			return queryResult{
				res: res,
				rtt: rtt,
			}
		}
	}

	return nullResult
}

func queryDnsOverHttps(req *dns.Msg, serverUrl string) queryResult {
	var client *http.Client
	if c, ok := httpClients[serverUrl]; ok {
		client = c
	} else {
		client = &http.Client{}
		httpClients[serverUrl] = client
	}

	if !strings.HasPrefix(serverUrl, "https://") {
		log.Errorf("%s is not a https URL", serverUrl)
		return nullResult
	}

	// Convert DNS msg to UDP wire protocol
	buf, err := req.Pack()
	if err != nil {
		log.Errorf("Could not generate UDP wire protocol payload for query: %s", err)
		return nullResult
	}

	// Create request with body
	httpReq, err := http.NewRequest("POST", serverUrl, bytes.NewReader(buf))
	if err != nil {
		log.Errorf("Could not generate UDP wire protocol payload for query: %s", err)
		return nullResult
	}

	// Ensure that SSL verification, routing, etc. works with the Host header
	httpReq.Header.Add("Accept", "application/dns-message")
	httpReq.Header.Add("Content-Type", "application/dns-message")

	var res *dns.Msg = nil
	retries := 3
	for retries > 0 {
		retries -= 1

		// Do the HTTPS request
		start := time.Now()
		httpRes, err := client.Do(httpReq)
		rtt := time.Since(start)

		if err != nil {
			go shared.ReportError(req, res, rtt, err)
			log.Errorf("Caught error while querying: %s", err)
			if httpRes != nil {
				if err = httpRes.Body.Close(); err != nil {
					log.Errorf("Error closing HTTP client body: %s", err)
				}
			}
		} else {
			if httpRes.StatusCode == 200 {
				udpPayload, err := ioutil.ReadAll(httpRes.Body)
				if err != nil {
					go shared.ReportError(req, res, rtt, err)
					if err = httpRes.Body.Close(); err != nil {
						log.Errorf("Error closing HTTP client body: %s", err)
					}
					continue
				}

				if err = httpRes.Body.Close(); err != nil {
					log.Errorf("Error closing HTTP client body: %s", err)
				}

				res := &dns.Msg{}
				err = res.Unpack(udpPayload)
				if err != nil {
					go shared.ReportError(req, res, rtt, err)
					continue
				}

				return queryResult{
					res: res,
					rtt: rtt,
				}
			} else {
				msg, _ := ioutil.ReadAll(httpRes.Body)
				log.Errorf("HTTP response %d %s: %s", httpRes.StatusCode, httpRes.Status, string(msg[:]))
				err = DnsOverHttpsRequestError
				go shared.ReportError(req, res, rtt, err)
				continue
			}
		}
	}

	return nullResult
}

// Do a DNS query for the given request
func Query(req *dns.Msg, dnsServers []string) *dns.Msg {
	// TODO: Parallelization of multiple queries
	// TODO: Configuration
	qrChan := make(chan queryResult)
	count := len(dnsServers)
	received := 0

	defer func() {
		go func() {
			// Throw away extra answers before closing
			for received < count {
				<-qrChan
				received++
			}
			close(qrChan)
		}()
	}()

	if count == 0 {
		log.Errorf("No DNS servers to query!")
		return nil
	}

	/*
		Supported formats:
		- dns+tls://1.1.1.1
		- dns+tls://1.1.1.1/cloudflare-dns.com
		- https://cloudflare-dns.com/dns-query#1.0.0.1
		- dns://1.0.0.1
	*/

	for _, dnsServer := range dnsServers {
		go func(dnsServer string) {
			var queryResult queryResult
			if strings.HasPrefix(dnsServer, "dns+tls://") {
				parts := strings.SplitN(strings.TrimPrefix(dnsServer, "dns+tls://"), "/", 2)
				ip := parts[0]
				name := ""
				if len(parts) == 2 {
					name = parts[1]
				}

				queryResult = queryDnsOverTls(req, ip, name)
			} else if strings.HasPrefix(dnsServer, "https://") {
				queryResult = queryDnsOverHttps(req, dnsServer)
			} else if strings.HasPrefix(dnsServer, "dns://") {
				queryResult = queryDns(req, strings.TrimPrefix(dnsServer, "dns://"))
			}

			queryResult.server = dnsServer

			select {
			case qrChan <- queryResult:
				return
			default:
				log.Debugf("Got result from %s but result was already closed", dnsServer)
			}
		}(dnsServer)
	}

	for qr := range qrChan {
		received++

		if qr.res != nil {
			// Just return first success
			go shared.ReportSuccess(req, qr.res, qr.rtt, qr.server)
			return qr.res
		} else if received == count {
			// All servers failed
			return nil
		}
	}

	return nil
}
