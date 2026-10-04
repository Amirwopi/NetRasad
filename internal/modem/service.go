package modem

import (
	"fmt"
	"strings"
	"time"
)

type Service struct{}

func NewService() *Service {
	return &Service{}
}

type ConnectedDevice struct {
	MAC  string `json:"mac"`
	IP   string `json:"ip"`
	Name string `json:"name"`
}

type ModemConfig struct {
	Host     string
	Username string
	Password string
	Brand    string // "tp-link", "d-link", "generic"
}

// executeCommand is a helper to login and run a single command
func (s *Service) executeCommand(cfg ModemConfig, command string) (string, error) {
	host := cfg.Host
	if host == "" {
		host = "192.168.1.1"
	}
	client, err := NewTelnetClient(host, 23)
	if err != nil {
		return "", fmt.Errorf("failed to connect to modem: %v", err)
	}
	defer client.Close()

	// Wait for login
	_, err = client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("timeout waiting for login prompt: %v", err)
	}

	client.Send(cfg.Username)

	// Wait for password
	_, err = client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("timeout waiting for password prompt: %v", err)
	}

	client.Send(cfg.Password)

	// Wait for prompt
	_, err = client.ReadUntil([]string{"> ", "# ", "$ ", "TP-LINK", "D-Link", "TBS>>"}, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("timeout waiting for prompt after login: %v", err)
	}

	if command == "" {
		return "", nil // Just testing connection
	}

	client.Send(command)

	output, err := client.ReadUntil([]string{"> ", "# ", "$ ", "TP-LINK", "D-Link", "TBS>>"}, 10*time.Second)
	return output, err
}

func (s *Service) TestConnection(host, user, pass string) error {
	_, err := s.executeCommand(ModemConfig{Host: host, Username: user, Password: pass}, "")
	return err
}

func (s *Service) RunModemCommand(host, user, pass, cmd string) (string, error) {
	return s.executeCommand(ModemConfig{Host: host, Username: user, Password: pass}, cmd)
}

type ADSLStatus struct {
	UpstreamRate   string `json:"upstreamRate"`
	DownstreamRate string `json:"downstreamRate"`
	UpstreamSNR    string `json:"upstreamSNR"`
	DownstreamSNR  string `json:"downstreamSNR"`
	UpstreamAtt    string `json:"upstreamAtt"`
	DownstreamAtt  string `json:"downstreamAtt"`
	RawOutput      string `json:"rawOutput"`
}

func (s *Service) GetADSLStatus(host, user, pass, brand string) (*ADSLStatus, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)
	
	client.Send("set scroll auto")
	client.ReadUntil([]string{"TBS>>"}, 2*time.Second)

	client.Send("sh")
	client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	
	client.Send("cat /proc/tc3162/adsl_stats")
	output, err := client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	
	// If it fails, fallback to general adsl command
	if err != nil || strings.Contains(output, "No such file") {
		output, _ = s.executeCommand(cfg, "adsl info --show")
	}

	status := &ADSLStatus{
		RawOutput: output,
	}

	lines := strings.Split(output, "\n")
	var isDownstream, isUpstream bool
	for _, line := range lines {
		lower := strings.ToLower(line)
		
		trimmed := strings.TrimSpace(lower)
		if strings.HasPrefix(trimmed, "downstream:") {
			isDownstream = true
			isUpstream = false
			continue
		}
		if strings.HasPrefix(trimmed, "upstream:") {
			isUpstream = true
			isDownstream = false
			continue
		}

		parts := extractNumbers(line)
		if len(parts) >= 1 {
			if strings.Contains(lower, "noise margin") || strings.Contains(lower, "snr margin") {
				if isDownstream || strings.Contains(lower, "downstream") {
					status.DownstreamSNR = parts[0]
				}
				if isUpstream || strings.Contains(lower, "upstream") {
					status.UpstreamSNR = parts[0]
				}
			}
			if strings.Contains(lower, "attenuation") {
				if isDownstream || strings.Contains(lower, "downstream") {
					status.DownstreamAtt = parts[0]
				}
				if isUpstream || strings.Contains(lower, "upstream") {
					status.UpstreamAtt = parts[0]
				}
			}
			if strings.Contains(lower, "bit rate") {
				if strings.Contains(lower, "near-end interleaved") || strings.Contains(lower, "near-end fast") {
					if parts[0] != "0" {
						status.DownstreamRate = parts[0]
					}
				}
				if strings.Contains(lower, "far-end interleaved") || strings.Contains(lower, "far-end fast") {
					if parts[0] != "0" {
						status.UpstreamRate = parts[0]
					}
				}
			}
		}

		// Broadcom / standard fallback (2 numbers per line)
		if len(parts) >= 2 {
			if strings.Contains(lower, "data rate") || strings.Contains(lower, "speed") || (strings.Contains(lower, "rate") && strings.Contains(lower, "kbps")) {
				if status.DownstreamRate == "" {
					status.UpstreamRate = parts[0]
					status.DownstreamRate = parts[1]
				}
			}
			if (strings.Contains(lower, "snr") && !strings.Contains(lower, "margin downstream")) {
				if status.DownstreamSNR == "" {
					status.UpstreamSNR = parts[0]
					status.DownstreamSNR = parts[1]
				}
			}
		}
	}

	return status, nil
}

func extractNumbers(line string) []string {
	var nums []string
	fields := strings.Fields(line)
	for _, f := range fields {
		// remove common non-numeric chars but keep dots/numbers
		clean := strings.Trim(f, "():,a-zA-Z")
		if clean != "" && (strings.ContainsAny(clean, "0123456789")) {
			nums = append(nums, clean)
		}
	}
	return nums
}

func (s *Service) GetConnectedDevices(host, user, pass, brand string) ([]ConnectedDevice, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	
	// Default to generic arp
	var output string
	var err error

	if strings.ToLower(brand) == "d-link" || strings.ToLower(brand) == "generic" {
		// For D-Link DSL-2877AL we need to enter 'sh' then 'cat /proc/net/arp'
		client, cerr := NewTelnetClient(cfg.Host, 23)
		if cerr != nil {
			return nil, fmt.Errorf("failed to connect: %v", cerr)
		}
		defer client.Close()
		client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
		client.Send(cfg.Username)
		client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
		client.Send(cfg.Password)
		client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)
		
		client.Send("sh")
		client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
		
		client.Send("cat /proc/net/arp")
		output, err = client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
		if err != nil {
			return nil, err
		}
	} else {
		output, err = s.executeCommand(cfg, "arp show")
		if err != nil {
			return nil, err
		}
	}

	devices := []ConnectedDevice{}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, ".") && strings.Contains(line, ":") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				// /proc/net/arp format: IP HW_type Flags HW_address Mask Device
				ip := parts[0]
				mac := parts[3]
				devices = append(devices, ConnectedDevice{
					IP:   ip,
					MAC:  mac,
					Name: "Unknown",
				})
			}
		}
	}

	return devices, nil
}

type FirewallRule struct {
	Chain       string `json:"chain"`
	Target      string `json:"target"`
	Protocol    string `json:"protocol"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Extra       string `json:"extra"`
}

type NATRule struct {
	Chain       string `json:"chain"`
	Target      string `json:"target"`
	Protocol    string `json:"protocol"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Extra       string `json:"extra"`
}

func (s *Service) GetFirewallRules(host, user, pass, brand string) ([]FirewallRule, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)
	
	client.Send("sh")
	client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	
	// Execute iptables -L -n
	client.Send("iptables -L -n")
	output, err := client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 10*time.Second)
	if err != nil {
		return nil, err
	}

	return parseIptablesOutput(output, false), nil
}

type RouteEntry struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Genmask     string `json:"genmask"`
	Flags       string `json:"flags"`
	Metric      string `json:"metric"`
	Ref         string `json:"ref"`
	Use         string `json:"use"`
	Iface       string `json:"iface"`
}

type InterfaceEntry struct {
	Name      string `json:"name"`
	RxBytes   string `json:"rxBytes"`
	RxPackets string `json:"rxPackets"`
	TxBytes   string `json:"txBytes"`
	TxPackets string `json:"txPackets"`
}

func (s *Service) GetRoutes(host, user, pass, brand string) ([]RouteEntry, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)
	
	client.Send("sh")
	client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	
	client.Send("route -n")
	output, err := client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	if err != nil {
		return nil, err
	}

	var routes []RouteEntry
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Kernel") || strings.HasPrefix(line, "Destination") || line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 8 {
			routes = append(routes, RouteEntry{
				Destination: parts[0],
				Gateway:     parts[1],
				Genmask:     parts[2],
				Flags:       parts[3],
				Metric:      parts[4],
				Ref:         parts[5],
				Use:         parts[6],
				Iface:       parts[7],
			})
		}
	}
	return routes, nil
}

func (s *Service) GetInterfaces(host, user, pass, brand string) ([]InterfaceEntry, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)
	
	client.Send("sh")
	client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	
	client.Send("cat /proc/net/dev")
	output, err := client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	if err != nil {
		return nil, err
	}

	var ifaces []InterfaceEntry
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				fields := strings.Fields(parts[1])
				if len(fields) >= 9 {
					ifaces = append(ifaces, InterfaceEntry{
						Name:      name,
						RxBytes:   fields[0],
						RxPackets: fields[1],
						TxBytes:   fields[8],
						TxPackets: fields[9],
					})
				}
			}
		}
	}
	return ifaces, nil
}


func (s *Service) GetNATRules(host, user, pass, brand string) ([]NATRule, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)
	
	client.Send("sh")
	client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	
	// Execute iptables -t nat -L -n
	client.Send("iptables -t nat -L -n")
	output, err := client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 10*time.Second)
	if err != nil {
		return nil, err
	}

	rules := parseIptablesOutput(output, true)
	var natRules []NATRule
	for _, r := range rules {
		natRules = append(natRules, NATRule(r))
	}
	return natRules, nil
}

func parseIptablesOutput(output string, isNat bool) []FirewallRule {
	var rules []FirewallRule
	lines := strings.Split(output, "\n")
	
	var currentChain string
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		if strings.HasPrefix(line, "Chain") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				currentChain = parts[1]
			}
			continue
		}
		
		if strings.HasPrefix(line, "target") {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			target := parts[0]
			proto := parts[1]
			// opt is parts[2]
			src := parts[3]
			dest := ""
			if len(parts) >= 5 {
				dest = parts[4]
			}
			extra := ""
			if len(parts) > 5 {
				extra = strings.Join(parts[5:], " ")
			}
			
			rules = append(rules, FirewallRule{
				Chain:       currentChain,
				Target:      target,
				Protocol:    proto,
				Source:      src,
				Destination: dest,
				Extra:       extra,
			})
		}
	}
	return rules
}

type WifiStatus struct {
	SSID      string `json:"ssid"`
	Hidden    bool   `json:"hidden"`
	ExtraSSID []string `json:"extraSsid"`
}

func (s *Service) GetWifiStatus(host, user, pass, brand string) ([]WifiStatus, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)

	client.Send("sh")
	client.ReadUntil([]string{"~ $"}, 5*time.Second)
	
	client.Send("cat /var/RT2860AP0.dat")
	out, err := client.ReadUntil([]string{"~ $"}, 5*time.Second)
	if err != nil || !strings.Contains(out, "SSID=") {
		client.Send("exit")
		return nil, fmt.Errorf("could not read wifi dat file")
	}
	client.Send("exit")

	var ssids []string
	var hides []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SSID=") {
			ssids = strings.Split(strings.TrimPrefix(line, "SSID="), ";")
		} else if strings.HasPrefix(line, "HideSSID=") {
			hides = strings.Split(strings.TrimPrefix(line, "HideSSID="), ";")
		}
	}
	var res []WifiStatus
	for i, ssid := range ssids {
		if ssid == "" {
			continue
		}
		hidden := false
		if i < len(hides) && hides[i] == "1" {
			hidden = true
		}
		res = append(res, WifiStatus{
			SSID:   ssid,
			Hidden: hidden,
		})
	}
	return res, nil
}

func (s *Service) GetDNS(host, user, pass, brand string) ([]string, error) {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)

	client.Send("sh")
	client.ReadUntil([]string{"~ $"}, 5*time.Second)
	
	client.Send("cat /var/resolv.conf")
	out, err := client.ReadUntil([]string{"~ $"}, 5*time.Second)
	client.Send("exit")
	if err != nil {
		return nil, err
	}
	var dns []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver ") {
			dns = append(dns, strings.TrimPrefix(line, "nameserver "))
		}
	}
	return dns, nil
}

func (s *Service) ChangeWifiConfig(host, user, pass, brand, ssid, wifiPass string) error {
	return fmt.Errorf("Wi-Fi configuration for %s is locked in Read-Only mode for safety. Please configure via web panel.", brand)
}

func (s *Service) Reboot(host, user, pass, brand string) error {
	cfg := ModemConfig{Host: host, Username: user, Password: pass, Brand: brand}
	
	client, err := NewTelnetClient(cfg.Host, 23)
	if err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}
	defer client.Close()
	client.ReadUntil([]string{"Login:", "login:", "Username:", "username:"}, 5*time.Second)
	client.Send(cfg.Username)
	client.ReadUntil([]string{"Password:", "password:"}, 5*time.Second)
	client.Send(cfg.Password)
	client.ReadUntil([]string{"> ", "# ", "$ ", "TBS>>"}, 5*time.Second)
	
	client.Send("sh")
	client.ReadUntil([]string{"$ ", "# ", "~ $", "TBS>>", "> "}, 5*time.Second)
	
	// Execute reboot safely
	client.Send("reboot")
	
	// We don't need to read output since it reboots
	return nil
}
