package shared

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os/exec"
	"strings"
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

	log.Debugf("Interfaces currently using DNS servers from DHCP: %s", strings.Join(interfaces, ", "))
}

func UpdateDnsServers() {
	for _, iface := range interfaces {
		iface = fmt.Sprintf("\"%s\"", iface)
		ps := fmt.Sprintf("Set-DnsClientServerAddress -InterfaceAlias %s -ServerAddresses (\"127.0.0.1\")", iface)
		cmd := exec.Command("powershell.exe", "-Command", ps)
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
		iface = fmt.Sprintf("\"%s\"", iface)
		ps := fmt.Sprintf("Set-DnsClientServerAddress -InterfaceAlias %s -ResetServerAddresses", iface)
		cmd := exec.Command("powershell.exe", "-Command", ps)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Errorf("Error restoring %s DNS servers: %s", iface, err)
			log.Errorf("Try manually with: netsh interface ipv4 set dnsservers name=%s source=dhcp", iface)
			log.Error(strings.TrimSpace(string(output[:])))
		} else {
			log.Debugf("%s now using DNS servers set by DHCP", iface)
		}
	}
}
