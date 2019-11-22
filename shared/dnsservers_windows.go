package shared

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
	"sync"
)

var interfaces = []string{}

func RememberDnsServers() {
	cmd := exec.Command("netsh", "interface", "ip", "show", "dnsservers")

	stdout, err := cmd.Output()
	if err != nil {
		log.Panicf("Could not check current interfaces: %s", err)
	}

	lines := strings.Split(string(stdout[:]), "\r\n")

	var iface = ""
	for _, line := range lines {
		line := strings.TrimSpace(line)

		if line == "" {
			iface = ""
			continue
		}

		// Configuration for interface "Local Area Connection* 1"
		if strings.HasPrefix(line, "Configuration for interface ") {
			parts := strings.SplitN(line, "\"", 3)
			iface = parts[1]
		}

		if iface != "" && strings.HasPrefix(line, "DNS servers configured through DHCP: ") {
			interfaces = append(interfaces, iface)
		}
	}

	log.Infof("Interfaces currently using DNS servers from DHCP: %s", strings.Join(interfaces, ", "))
}

func UpdateDnsServers() {
	wg := &sync.WaitGroup{}
	for _, iface := range interfaces {
		wg.Add(1)

		go func(iface string) {
			iface = fmt.Sprintf("\"%s\"", iface)
			ps := fmt.Sprintf("Set-DnsClientServerAddress -InterfaceAlias %s -ServerAddresses (\"127.0.0.1\")", iface)
			cmd := exec.Command("powershell.exe", "-Command", ps)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Errorf("Error setting %s DNS servers: %s", iface, err)
				log.Errorf("Output was: %s", string(output[:]))
			} else {
				log.Infof("%s now using Better DNS", iface)
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
			iface = fmt.Sprintf("\"%s\"", iface)
			ps := fmt.Sprintf("Set-DnsClientServerAddress -InterfaceAlias %s -ResetServerAddresses", iface)
			cmd := exec.Command("powershell.exe", "-Command", ps)
			output, err := cmd.CombinedOutput()
			if err != nil {
				log.Errorf("Error restoring %s DNS servers: %s", iface, err)
				log.Errorf("Try manually with: netsh interface ipv4 set dnsservers name=%s source=dhcp", iface)
				log.Error(strings.TrimSpace(string(output[:])))
			} else {
				log.Infof("%s now using DNS servers set by DHCP", iface)
			}
			wg.Done()
		}(iface)
	}

	wg.Wait()
}
