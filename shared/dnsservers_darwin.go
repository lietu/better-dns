package shared

import (
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
	"sync"
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

		// Hardware Port: Ethernet
		if strings.HasPrefix(line, "Hardware Port: ") {
			parts := strings.SplitN(line, ": ", 2)
			interfaces = append(interfaces, parts[1])
		}
	}

	log.Debugf("Interfaces found: %s", strings.Join(interfaces, ", "))
}

func UpdateDnsServers() {
	wg := &sync.WaitGroup{}
	for _, iface := range interfaces {
		wg.Add(1)

		go func(iface string) {
			cmd := exec.Command("networksetup", "-setdnsservers", iface, "127.0.0.1")
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Errorf("Error setting %s DNS servers: %s", iface, err)
				log.Errorf("Output was: %s", string(output[:]))
			} else {
				log.Debugf("%s now using 127.0.0.1 for DNS", iface)
			}
			wg.Done()
		}(iface)
	}

	wg.Wait()
}

func RestoreDnsServers() {
	wg := &sync.WaitGroup{}
	for _, iface := range interfaces {
		wg.Add(1)

		go func(iface string) {
			cmd := exec.Command("networksetup", "-setdnsservers", iface, "empty")
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Errorf("Error restoring %s DNS servers: %s", iface, err)
				log.Errorf("Try manually with: networksetup -setdnsservers %s empty", iface)
				log.Error(strings.TrimSpace(string(output[:])))
			} else {
				log.Debugf("%s now using DNS servers set by DHCP", iface)
			}
			wg.Done()
		}(iface)
	}

	wg.Wait()
}
