package shared

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

type Config struct {
	BlockLists []string `yaml:"block_lists"`
	Blacklist  []string `yaml:"blacklist"`
	DnsServers []string `yaml:"dns_servers"`
	ListenHost string   `yaml:"listen_host"`
	mutex      sync.Mutex
}

// Some sensible lists that seem to cause little to no problems
var defaultBlockLists = []string{
	"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
	"https://mirror1.malwaredomains.com/files/justdomains",
	"http://sysctl.org/cameleon/hosts",
	"https://s3.amazonaws.com/lists.disconnect.me/simple_tracking.txt",
	"https://s3.amazonaws.com/lists.disconnect.me/simple_ad.txt",
	"https://hosts-file.net/ad_servers.txt",
	"https://pgl.yoyo.org/adservers/serverlist.php?hostformat=hosts&showintro=0&mimetype=plaintext",
}

// Cloudflare's DNS-over-HTTPS and DNS-over-TLS servers seem like good defaults, one likely works
var defaultDnsServers = []string{
	"https://1.1.1.1/dns-query",
	"dns+tls://1.0.0.1",
}

// Blacklist Web Proxy Auto-Discovery protocol by default for minor speedup
var defaultBlacklist = []string{
	"wpad.*",
}

func (c *Config) SetDnsServers(dnsServers []string) {
	// String arrays are thread safe, right?
	c.DnsServers = dnsServers
}

func (c *Config) GetDnsServers() []string {
	// String arrays are thread safe, right?
	return c.DnsServers
}

func (c *Config) SetBlacklist(blacklist []string) {
	// String arrays are thread safe, right?
	c.Blacklist = blacklist
}

func (c *Config) GetBlacklist() []string {
	// String arrays are thread safe, right?
	return c.Blacklist
}

func validate(c Config) {
	haveErrors := false
	for _, uri := range c.DnsServers {
		if strings.HasPrefix(uri, "dns://") {
			log.Infof("Using insecure DNS server: %s", uri)
		} else if strings.HasPrefix(uri, "https://") {
			log.Infof("Using DNS over HTTPS server: %s", uri)
		} else if strings.HasPrefix(uri, "dns+tls://") {
			log.Infof("Using DNS over TLS server: %s", uri)
		} else {
			haveErrors = true
			log.Errorf("Unsupported DNS URI: %s.", uri)
			log.Errorf("Should look like: https://1.1.1.1/dns-query dns+tls://1.1.1.1 or dns://1.1.1.1")
		}
	}

	if haveErrors {
		panic("Cannot continue with invalid configuration.")
	}
}

func NewConfig(src string, usingDefault bool) *Config {
	c := Config{
		BlockLists: defaultBlockLists,
		Blacklist:  defaultBlacklist,
		DnsServers: defaultDnsServers,
		ListenHost: "127.0.0.1",
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		if usingDefault {
			// Default config path not overridden, file does not exist, just use defaults
			log.Info("Using built-in default configuration.")
			return &c
		}

		log.Panicf("Could not find config file %s", src)
	}

	data, err := ioutil.ReadFile(src)
	if err != nil {
		log.Panicf("Error reading config file %s: %s", src, err)
	}

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		log.Panicf("Error parsing config file %s: %s", src, err)
	}

	log.Infof("Using configuration from %s", src)

	validate(c)

	return &c
}
