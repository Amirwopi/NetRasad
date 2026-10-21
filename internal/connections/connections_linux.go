//go:build linux

package connections

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var procNetTables = []struct {
	path  string
	proto Protocol
	isV6  bool
}{
	{"/proc/net/tcp", TCP, false},
	{"/proc/net/tcp6", TCP, true},
	{"/proc/net/udp", UDP, false},
	{"/proc/net/udp6", UDP, true},
}

type socketEntry struct {
	conn  Connection
	inode uint64
}

func (s *Service) GetConnections() ([]Connection, error) {
	var entries []socketEntry
	for _, t := range procNetTables {
		e, err := parseProcNetFile(t.path, t.proto, t.isV6)
		if err != nil {
			// The IPv6 tables do not exist when IPv6 is disabled at boot.
			if t.isV6 && errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("parse %s: %w", t.path, err)
		}
		entries = append(entries, e...)
	}

	owners.resolve(entries)

	conns := make([]Connection, len(entries))
	for i, e := range entries {
		conns[i] = e.conn
	}
	return conns, nil
}

func parseProcNetFile(path string, proto Protocol, isV6 bool) ([]socketEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseProcNet(f, proto, isV6)
}

func parseProcNet(r io.Reader, proto Protocol, isV6 bool) ([]socketEntry, error) {
	var entries []socketEntry
	scanner := bufio.NewScanner(r)
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

		entries = append(entries, socketEntry{conn: c, inode: inode})
	}
	return entries, scanner.Err()
}

func parseAddrPort(s string, isV6 bool) (string, int, error) {
	colon := strings.LastIndexByte(s, ':')
	if colon < 0 {
		return "", 0, fmt.Errorf("missing colon")
	}
	addrHex := s[:colon]
	portHex := s[colon+1:]

	port, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return "", 0, fmt.Errorf("bad port: %w", err)
	}

	want := 8
	if isV6 {
		want = 32
	}
	if len(addrHex) != want {
		return "", 0, fmt.Errorf("bad address length %d", len(addrHex))
	}

	// The kernel prints the address as 32-bit words, each holding raw
	// network-order bytes formatted as a host-endian integer.
	ip := make(net.IP, len(addrHex)/2)
	for i := 0; i < len(ip); i += 4 {
		v, err := strconv.ParseUint(addrHex[i*2:i*2+8], 16, 32)
		if err != nil {
			return "", 0, fmt.Errorf("bad address: %w", err)
		}
		binary.NativeEndian.PutUint32(ip[i:i+4], uint32(v))
	}
	return ip.String(), int(port), nil
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

// Finding which process owns a socket means walking every /proc/<pid>/fd,
// which is by far the most expensive step and runs every second, so owners
// are cached and only rescanned (at most every ownerRescanInterval) once
// sockets appear that the last scan has not seen.
const ownerRescanInterval = 2 * time.Second

var owners = &socketOwners{}

type socketOwners struct {
	mu       sync.Mutex
	seen     map[uint64]bool
	pids     map[uint64]int
	names    map[int]string
	lastScan time.Time
}

func (o *socketOwners) resolve(entries []socketEntry) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.needsRescan(entries) {
		o.pids = buildInodePIDMap()
		o.names = buildPIDNameMap(o.pids)
		o.lastScan = time.Now()
		// Sockets owned by other users can never be attributed without root;
		// remembering them keeps them from triggering a rescan every time.
		o.seen = make(map[uint64]bool, len(entries)+len(o.pids))
		for _, e := range entries {
			o.seen[e.inode] = true
		}
		for inode := range o.pids {
			o.seen[inode] = true
		}
	}

	for i := range entries {
		if pid, ok := o.pids[entries[i].inode]; ok {
			entries[i].conn.PID = pid
			entries[i].conn.ProcessName = o.names[pid]
		}
	}
}

func (o *socketOwners) needsRescan(entries []socketEntry) bool {
	if o.pids == nil {
		return true
	}
	if time.Since(o.lastScan) < ownerRescanInterval {
		return false
	}
	for _, e := range entries {
		// Inode 0 marks sockets that no longer belong to a process (TIME_WAIT).
		if e.inode != 0 && !o.seen[e.inode] {
			return true
		}
	}
	return false
}

func buildInodePIDMap() map[uint64]int {
	result := make(map[uint64]int)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return result
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
	return result
}

func buildPIDNameMap(inodePID map[uint64]int) map[int]string {
	result := make(map[int]string, len(inodePID))
	for _, pid := range inodePID {
		if _, done := result[pid]; done {
			continue
		}
		result[pid] = processName(pid)
	}
	return result
}

// processName prefers the executable's file name, as the Windows build does;
// /proc/<pid>/comm is cut at 15 characters and often names a helper thread.
func processName(pid int) string {
	dir := filepath.Join("/proc", strconv.Itoa(pid))
	if exe, err := os.Readlink(filepath.Join(dir, "exe")); err == nil {
		exe = strings.TrimSuffix(exe, " (deleted)")
		if name := filepath.Base(exe); name != "." && name != "/" {
			return name
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, "comm"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
