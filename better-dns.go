package main

import (
	"flag"
	"github.com/lietu/better-dns/server"
	"github.com/lietu/better-dns/shared"
	"github.com/lietu/better-dns/stats"
	"github.com/mattn/go-colorable"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const PORT = 53
const DEFAULT_CONFIG = "~/.better-dns.yaml"

var configFileArg = flag.String("config", DEFAULT_CONFIG, "Path to YAML config")

func loadLists(urls []string) {
	wg := &sync.WaitGroup{}
	for i := range urls {
		wg.Add(1)
		go func(url string) {
			server.BlockFromURL(url)
			wg.Done()
		}(urls[i])
	}

	wg.Wait()
	server.LogLists()
}

func main() {
	formatter := &log.TextFormatter{ForceColors: true, DisableTimestamp: true}
	log.SetFormatter(formatter)
	log.SetOutput(colorable.NewColorableStdout())

	// Check what config file we're supposed to be using
	flag.Parse()
	configFile := *configFileArg
	usingDefault := configFile == DEFAULT_CONFIG

	// Process "home directory" in cross-platform manner
	if strings.HasPrefix(configFile, "~/") {
		usr, err := user.Current()
		if err != nil {
			log.Fatalf("Could not resolve user: %s", err)
		}

		configFile = path.Join(usr.HomeDir, strings.TrimPrefix(configFile, "~/"))
	}

	// Read config (if it exists)
	config := shared.NewConfig(configFile, usingDefault)

	// Set log level
	level, err := log.ParseLevel(config.LogLevel)
	if err != nil {
		log.Fatalf("Invalid log level %s: %s", config.LogLevel, err)
	}
	log.SetLevel(level)

	shared.RememberDnsServers()
	loadLists(config.BlockLists)

	handler := server.NewHandler(config)
	port := strconv.Itoa(PORT)

	go func() {
		log.Infof("Listening to UDP %s:%s", config.ListenHost, port)
		err := dns.ListenAndServe(config.ListenHost+":"+port, "udp", handler)
		if err != nil {
			log.Panicf("Could not listen to UDP port: %s", err)
		}
	}()

	go func() {
		log.Infof("Listening to TCP %s:%s", config.ListenHost, port)
		err := dns.ListenAndServe(config.ListenHost+":"+port, "tcp", handler)
		if err != nil {
			log.Panicf("Could not listen to TCP port: %s", err)
		}
	}()

	go func() {
		duration := time.Hour
		start := time.Now()
		previous := stats.Stats{}
		for {
			time.Sleep(duration)
			total := stats.GetStats()
			successes := total.Successes - previous.Successes

			rtt := time.Duration(0)
			if successes > 0 {
				rtt = total.Rtt / time.Duration(successes)
			}

			diff := stats.Stats{
				Blocked:   total.Blocked - previous.Blocked,
				Cached:    total.Cached - previous.Cached,
				Errors:    total.Errors - previous.Errors,
				Successes: successes,
				Rtt:       rtt,
			}

			saved := (time.Duration(diff.Cached) * diff.Rtt).Truncate(time.Millisecond)
			log.Infof("")
			log.Infof("------------------------------")
			log.Infof("Stats for last %s:", duration)
			log.Infof(" - Blocked: %d", diff.Blocked)
			log.Infof(" - Successes: %d (%s avg)", diff.Successes, diff.Rtt.Truncate(time.Millisecond))
			log.Infof(" - Cache hits: %d (~%s saved)", diff.Cached, saved)
			log.Infof(" - Errors: %d", diff.Errors)

			totalSaved := (time.Duration(total.Cached) * diff.Rtt).Truncate(time.Millisecond)
			log.Infof("")
			log.Infof("Stats since start (%s):", time.Since(start).Truncate(duration))
			log.Infof(" - Blocked: %d", total.Blocked)
			log.Infof(" - Successes: %d (%s avg)", total.Successes, diff.Rtt.Truncate(time.Millisecond))
			log.Infof(" - Cache hits: %d (~%s saved)", total.Cached, totalSaved)
			log.Infof(" - Errors: %d", total.Errors)
			log.Infof("------------------------------")
			previous = total
		}
	}()

	shared.UpdateDnsServers()

	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig

	shared.RestoreDnsServers()
	log.Debugf("Signal (%v) received, stopping", s)
	log.Info("Exiting...")
}
