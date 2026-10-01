//go:build linux

package connections

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *Service) GetConnections() ([]Connection, error) {
	inodePID, err := buildInodePIDMap()
	if err != nil {
		inodePID = map[uint64]int{}
	}
	pidName := buildPIDNameMap(inodePID)

	var conns []Connection

	tcp4, err := parseProcNet("/proc/net/tcp", TCP, false, inodePID, pidName)
	if err != nil {
		return nil, fmt.Errorf("parse /proc/net/tcp: %w", err)
	}
	conns = append(conns, tcp4...)

	tcp6, err := parseProcNet("/proc/net/tcp6", TCP, true, inodePID, pidName)
	if err != nil {
		return nil, fmt.Errorf("parse /proc/net/tcp6: %w", err)
	}
	conns = append(conns, tcp6...)

	udp4, err := parseProcNet("/proc/net/udp", UDP, false, inodePID, pidName)
	if err != nil {
		return nil, fmt.Errorf("parse /proc/net/udp: %w", err)
	}
	conns = append(conns, udp4...)

	udp6, err := parseProcNet("/proc/net/udp6", UDP, true, inodePID, pidName)
	if err != nil {
		return nil, fmt.Errorf("parse /proc/net/udp6: %w", err)
	}
	conns = append(conns, udp6...)

	return conns, nil
}

func parseProcNet(path string, proto Protocol, isV6 bool, inodePID map[uint64]int, pidName map[int]string) ([]Connection, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var conns []Connection
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		if lineNum == 1 {
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}

		local := fields[1]
		remote := fields[2]
		stateHex := fields[3]
		inodeStr := fields[9]

		localIP, localPort, err := parseAddrPort(local, isV6)
		if err != nil {
			continue
		}
		remoteIP, remotePort, err := parseAddrPort(remote, isV6)
		if err != nil {
			continue
		}

		inode, err := strconv.ParseUint(inodeStr, 10, 64)
		if err != nil {
			continue
		}

		c := Connection{
			Protocol:      proto,
			LocalAddress:  localIP,
			LocalPort:     localPort,
			RemoteAddress: remoteIP,
			RemotePort:    remotePort,
		}

		if proto == TCP {
			c.State = tcpStateFromHex(stateHex)
		} else {
			c.State = StateNA
		}

		if pid, ok := inodePID[inode]; ok {
			c.PID = pid
			c.ProcessName = pidName[pid]
		}

		conns = append(conns, c)
	}
	return conns, scanner.Err()
}

func parseAddrPort(s string, isV6 bool) (string, int, error) {
	colon := strings.LastIndexByte(s, ':')
	if colon < 0 {
		return "", 0, fmt.Errorf("missing colon")
	}
	addrHex := s[:colon]
	portHex := s[colon+1:]

	port, err := strconv.ParseInt(portHex, 16, 32)
	if err != nil {
		return "", 0, fmt.Errorf("bad port: %w", err)
	}

	if isV6 {
		if len(addrHex) != 32 {
			return "", 0, fmt.Errorf("bad v6 addr length")
		}
		ip, err := parseIPv6Hex(addrHex)
		if err != nil {
			return "", 0, err
		}
		return ip, int(port), nil
	}

	if len(addrHex) != 8 {
		return "", 0, fmt.Errorf("bad v4 addr length")
	}
	v, err := strconv.ParseUint(addrHex, 16, 32)
	if err != nil {
		return "", 0, fmt.Errorf("bad v4 addr: %w", err)
	}
	ip := fmt.Sprintf("%d.%d.%d.%d",
		byte(v),
		byte(v>>8),
		byte(v>>16),
		byte(v>>24),
	)
	return ip, int(port), nil
}

func parseIPv6Hex(h string) (string, error) {
	if len(h) != 32 {
		return "", fmt.Errorf("bad v6 length")
	}
	var b [16]byte
	for i := 0; i < 4; i++ {
		seg := h[i*8 : (i+1)*8]
		v, err := strconv.ParseUint(seg, 16, 32)
		if err != nil {
			return "", err
		}
		b[i*4+0] = byte(v)
		b[i*4+1] = byte(v >> 8)
		b[i*4+2] = byte(v >> 16)
		b[i*4+3] = byte(v >> 24)
	}
	return fmt.Sprintf("%x:%x:%x:%x:%x:%x:%x:%x",
		(b[0]<<8)|b[1], (b[2]<<8)|b[3],
		(b[4]<<8)|b[5], (b[6]<<8)|b[7],
		(b[8]<<8)|b[9], (b[10]<<8)|b[11],
		(b[12]<<8)|b[13], (b[14]<<8)|b[15],
	), nil
}

func tcpStateFromHex(h string) TCPState {
	v, err := strconv.ParseInt(h, 16, 32)
	if err != nil {
		return StateUnknown
	}
	switch v {
	case 1:
		return StateEstablished
	case 2:
		return StateSynSent
	case 3:
		return StateSynReceived
	case 4:
		return StateFinWait1
	case 5:
		return StateFinWait2
	case 6:
		return StateTimeWait
	case 7:
		return StateClosed
	case 8:
		return StateCloseWait
	case 9:
		return StateLastAck
	case 10:
		return StateListen
	case 11:
		return StateClosing
	default:
		return StateUnknown
	}
}

func buildInodePIDMap() (map[uint64]int, error) {
	result := make(map[uint64]int)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		fdDir := filepath.Join("/proc", entry.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}
			if !strings.HasPrefix(link, "socket:[") {
				continue
			}
			inodeStr := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
			inode, err := strconv.ParseUint(inodeStr, 10, 64)
			if err != nil {
				continue
			}
			if _, exists := result[inode]; !exists {
				result[inode] = pid
			}
		}
	}
	return result, nil
}

func buildPIDNameMap(inodePID map[uint64]int) map[int]string {
	result := make(map[int]string, len(inodePID))
	seen := make(map[int]bool)
	for _, pid := range inodePID {
		if seen[pid] {
			continue
		}
		seen[pid] = true
		data, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
		if err != nil {
			continue
		}
		result[pid] = strings.TrimSpace(string(data))
	}
	return result
}
