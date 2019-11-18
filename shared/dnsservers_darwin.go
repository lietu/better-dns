package shared

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
)

var interfaces = []string{}

func RememberDnsServers() {
	cmd := exec.Command("networksetup", "-listallhardwareports")

	stdout, err := cmd.Output()
	if err != nil {
		log.Panicf("Could not check current interfaces: %s", err)
	}

	lines := strings.Split(string(stdout[:]), "\n")

	for _, line := range lines {
		line := strings.TrimSpace(line)

		// Configuration for interface "Local Area Connection* 1"
		if strings.HasPrefix(line, "Hardware Port: ") {
			parts := strings.SplitN(line, ": ", 2)
			interfaces = append(interfaces, parts[1])
		}
	}

	log.Debugf("Interfaces found: %s", strings.Join(interfaces, ", "))
}

func UpdateDnsServers() {
	for _, iface := range interfaces {
		cmd := exec.Command("networksetup", "-setdnsservers", iface, "127.0.0.1")
		output, err := cmd.CombinedOutput()
		log.Error(strings.TrimSpace(string(output[:])))
		if err != nil {
			log.Errorf("Error setting %s DNS servers: %s", iface, err)
		} else {
			log.Debugf("%s now using 127.0.0.1 for DNS", iface)
		}
	}
}

func RestoreDnsServers() {
	for _, iface := range interfaces {
		cmd := exec.Command("networksetup", "-setdnsservers", iface)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Errorf("Error restoring %s DNS servers: %s", iface, err)
			log.Errorf("Try manually with: networksetup -setdnsservers %s", iface)
			log.Error(strings.TrimSpace(string(output[:])))
		} else {
			log.Debugf("%s now using DNS servers set by DHCP", iface)
		}
	}
}
