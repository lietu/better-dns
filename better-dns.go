package main

import (
	"flag"
	"github.com/lietu/better-dns/server"
	"github.com/lietu/better-dns/shared"
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

	// Process "home directory" in cross-platform manner
	if strings.HasPrefix(configFile, "~/") {
		usr, err := user.Current()
		if err != nil {
			log.Fatalf("Could not resolve user: %s", err)
		}

		configFile = path.Join(usr.HomeDir, strings.TrimPrefix(configFile, "~/"))
	}

	// Read config (if it exists)
	usingDefault := configFile == DEFAULT_CONFIG
	config := shared.NewConfig(configFile, usingDefault)

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

	shared.UpdateDnsServers()

	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig

	shared.RestoreDnsServers()

	log.Fatalf("Signal (%v) received, stopping", s)
}
