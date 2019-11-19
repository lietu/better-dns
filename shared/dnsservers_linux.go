package shared

import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
)

const RESOLV_CONF = "/etc/resolv.conf"
const BETTER_DNS_RESOLV_CONF = `# File generated by better-dns
nameserver 127.0.0.1
`

var oldResolvConf string = ""

func RememberDnsServers() {
	data, err := ioutil.ReadFile(RESOLV_CONF)
	if err != nil {
		log.Panicf("Could not check current /etc/resolv.conf: %s", err)
	}

	oldResolvConf = string(data[:])
	log.Debugf("Previous /etc/resolv.conf: %s", oldResolvConf)
}

func UpdateDnsServers() {
	log.Debugf("Updating /etc/resolv.conf to: %s", BETTER_DNS_RESOLV_CONF)
	err := ioutil.WriteFile(RESOLV_CONF, []byte(BETTER_DNS_RESOLV_CONF), os.FileMode(0644))
	if err != nil {
		log.Errorf("Could not update /etc/resolv.conf: %s", err)
	}
}

func RestoreDnsServers() {
	log.Debugf("Restoring /etc/resolv.conf to: %s", oldResolvConf)
	err := ioutil.WriteFile(RESOLV_CONF, []byte(oldResolvConf), os.FileMode(0644))
	if err != nil {
		log.Errorf("Could not update /etc/resolv.conf: %s", err)
	}
}
