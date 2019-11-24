package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/lietu/better-dns/server"
	"github.com/lietu/better-dns/shared"
	"github.com/lietu/better-dns/stats"
	"github.com/mattn/go-colorable"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const PORT = 53
var defaultConfig = path.Join(shared.GetConfigDir() , "better-dns.yaml")
var configFileArg = flag.String("config", defaultConfig, "Path to YAML config")
var trayArg = flag.Bool("tray", false, "Use better-dns-tray communication protocol")

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
	// Check what config file we're supposed to be using
	flag.Parse()
	configFile := *configFileArg
	usingDefault := configFile == defaultConfig

	if *trayArg == false {
		formatter := &log.TextFormatter{ForceColors: true, DisableTimestamp: true}
		log.SetFormatter(formatter)
		log.SetOutput(colorable.NewColorableStdout())
	} else {
		// Log to file if running from tray
		logFile := path.Join(shared.GetConfigDir(), "better-dns.log")
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err == nil {
			log.SetOutput(file)
		} else {
			log.Errorf("Error opening log file: %s", err)
		}
	}

	// Read config (if it exists)
	config := shared.NewConfig(configFile, usingDefault)

	// Set logger level
	if *trayArg == false {
		level, err := log.ParseLevel(config.LogLevel)
		if err != nil {
			log.Fatalf("Invalid log level %s: %s", config.LogLevel, err)
		}
		log.SetLevel(level)
	}

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

	if *trayArg == false {
		go monitorStats()
	} else {
		go trayStats()
	}

	shared.UpdateDnsServers()

	exitCn := make(chan bool, 10)
	sig := make(chan os.Signal, 10)

	go func() {
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		s := <-sig
		log.Debugf("Signal (%v) received, stopping", s)
		exitCn <- true
	}()

	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			text, err := reader.ReadString('\n')
			if err != nil {
				log.Debugf("Caught error reading stdin: %s", err)
			} else {
				if strings.TrimSpace(text) == "exit" {
					log.Info("Got exit signal via stdin")
					exitCn <- true
				}
			}
		}
	}()

	<- exitCn

	shared.RestoreDnsServers()
	log.Info("Exiting...")
}

func monitorStats() {
	duration := time.Hour
	start := time.Now()
	previous := stats.Stats{}
	for {
		time.Sleep(duration)
		total := stats.GetStats()
		diffSuccesses := total.Successes - previous.Successes

		rtt := time.Duration(0)
		if total.Successes > 0 {
			rtt = (previous.Rtt + total.Rtt) / time.Duration(total.Successes)
		}

		diffRtt := time.Duration(0)
		if diffSuccesses > 0 {
			diffRtt = total.Rtt / time.Duration(diffSuccesses)
		}

		totalReqs := total.Successes + total.Blocked + total.Cached + total.Errors
		totalBlockPct := stats.RequestPct(total.Blocked, totalReqs)
		totalCachePct := stats.RequestPct(total.Cached, totalReqs)
		totalErrorPct := stats.RequestPct(total.Errors, totalReqs)

		diff := stats.Stats{
			Blocked:   total.Blocked - previous.Blocked,
			Cached:    total.Cached - previous.Cached,
			Errors:    total.Errors - previous.Errors,
			Successes: diffSuccesses,
			Rtt:       diffRtt,
		}

		diffReqs := diff.Successes + diff.Blocked + diff.Cached + diff.Errors
		diffBlockPct := stats.RequestPct(diff.Blocked, diffReqs)
		diffCachePct := stats.RequestPct(diff.Cached, diffReqs)
		diffErrorPct := stats.RequestPct(diff.Errors, diffReqs)

		diffSaved := (time.Duration(diff.Cached) * diff.Rtt).Truncate(time.Millisecond)
		totalSaved := (time.Duration(total.Cached) * rtt).Truncate(time.Millisecond)


		log.Infof("")
		log.Infof("------------------------------")
		log.Infof("Stats for last %s:", duration)
		log.Infof(" - Requests: %d", diffReqs)
		log.Infof(" - Successes: %d (%s avg)", diff.Successes, diff.Rtt.Truncate(time.Millisecond))
		log.Infof(" - Blocked: %d (%s)", diff.Blocked, diffBlockPct)
		log.Infof(" - Cache hits: %d (%s, ~%s saved)", diff.Cached, diffCachePct, diffSaved)
		log.Infof(" - Errors: %d (%s)", diff.Errors, diffErrorPct)

		log.Infof("")
		log.Infof("Stats since start (%s):", time.Since(start).Truncate(duration))
		log.Infof(" - Requests: %d", totalReqs)
		log.Infof(" - Successes: %d (%s avg)", total.Successes, rtt.Truncate(time.Millisecond))
		log.Infof(" - Blocked: %d (%s)", total.Blocked, totalBlockPct)
		log.Infof(" - Cache hits: %d (%s, ~%s saved)", total.Cached, totalCachePct, totalSaved)
		log.Infof(" - Errors: %d (%s)", total.Errors, totalErrorPct)
		log.Infof("------------------------------")

		previousRtt := previous.Rtt
		previous = total
		previous.Rtt += previousRtt
	}
}

func trayStats() {
	fmt.Println("S:0,B:0,C:0,E:0,R:0")

	duration := time.Second * 3
	previous := stats.Stats{}

	for {
		time.Sleep(duration)
		total := stats.GetStats()

		rtt := time.Duration(0)
		if total.Successes > 0 {
			rtt = (previous.Rtt + total.Rtt) / time.Duration(total.Successes)
		}

		fmt.Printf("S:%d,B:%d,C:%d,E:%d,R:%d\n", total.Successes, total.Blocked, total.Cached, total.Errors, rtt / time.Millisecond)

		previousRtt := previous.Rtt
		previous = total
		previous.Rtt += previousRtt
	}
}