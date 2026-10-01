//go:build windows

package connections

import (
	"encoding/binary"
	"fmt"
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	tcpTableOwnerPIDAll = 5
	udpTableOwnerPID    = 2
)

type mibTCPRowOwnerPID struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
	OwningPID  uint32
}

type mibUDPRowOwnerPID struct {
	LocalAddr uint32
	LocalPort uint32
	OwningPID uint32
}

var (
	modIphlpapi        = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetExtTcpTable = modIphlpapi.NewProc("GetExtendedTcpTable")
	procGetExtUdpTable = modIphlpapi.NewProc("GetExtendedUdpTable")
)

func (s *Service) GetConnections() ([]Connection, error) {
	pidName, err := buildPIDNameMap()
	if err != nil {
		pidName = map[int]string{}
	}

	var conns []Connection

	tcp4, err := getTCPTable(windows.AF_INET)
	if err != nil {
		return nil, fmt.Errorf("tcp4: %w", err)
	}
	conns = append(conns, parseTCPTable(tcp4, windows.AF_INET, pidName)...)

	tcp6, err := getTCPTable(windows.AF_INET6)
	if err != nil {
		return nil, fmt.Errorf("tcp6: %w", err)
	}
	conns = append(conns, parseTCPTable(tcp6, windows.AF_INET6, pidName)...)

	udp4, err := getUDPTable(windows.AF_INET)
	if err != nil {
		return nil, fmt.Errorf("udp4: %w", err)
	}
	conns = append(conns, parseUDPTable(udp4, windows.AF_INET, pidName)...)

	udp6, err := getUDPTable(windows.AF_INET6)
	if err != nil {
		return nil, fmt.Errorf("udp6: %w", err)
	}
	conns = append(conns, parseUDPTable(udp6, windows.AF_INET6, pidName)...)

	return conns, nil
}

func getTCPTable(af uint32) ([]byte, error) {
	var size uint32
	r1, _, _ := procGetExtTcpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(af),
		uintptr(tcpTableOwnerPIDAll),
		0,
	)
	if r1 != 122 && r1 != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable size probe failed: %w", windows.Errno(r1))
	}

	buf := make([]byte, size)
	r1, _, _ = procGetExtTcpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(af),
		uintptr(tcpTableOwnerPIDAll),
		0,
	)
	if r1 != 0 {
		return nil, fmt.Errorf("GetExtendedTcpTable failed: %w", windows.Errno(r1))
	}
	return buf, nil
}

func getUDPTable(af uint32) ([]byte, error) {
	var size uint32
	r1, _, _ := procGetExtUdpTable.Call(
		0,
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(af),
		uintptr(udpTableOwnerPID),
		0,
	)
	if r1 != 122 && r1 != 0 {
		return nil, fmt.Errorf("GetExtendedUdpTable size probe failed: %w", windows.Errno(r1))
	}

	buf := make([]byte, size)
	r1, _, _ = procGetExtUdpTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		1,
		uintptr(af),
		uintptr(udpTableOwnerPID),
		0,
	)
	if r1 != 0 {
		return nil, fmt.Errorf("GetExtendedUdpTable failed: %w", windows.Errno(r1))
	}
	return buf, nil
}

func parseTCPTable(buf []byte, af uint32, pidName map[int]string) []Connection {
	if len(buf) < 4 {
		return nil
	}
	numEntries := binary.LittleEndian.Uint32(buf[:4])

	var rowSize int
	if af == windows.AF_INET {
		rowSize = int(unsafe.Sizeof(mibTCPRowOwnerPID{}))
	} else {
		rowSize = 56
	}

	data := buf[4:]
	var conns []Connection
	for i := uint32(0); i < numEntries; i++ {
		offset := int(i) * rowSize
		if offset+rowSize > len(data) {
			break
		}
		row := data[offset : offset+rowSize]

		var c Connection
		c.Protocol = TCP

		if af == windows.AF_INET {
			c.State = tcpState(row[0:4])
			c.LocalAddress = ipv4String(binary.LittleEndian.Uint32(row[4:8]))
			c.LocalPort = ntohsField(binary.LittleEndian.Uint32(row[8:12]))
			c.RemoteAddress = ipv4String(binary.LittleEndian.Uint32(row[12:16]))
			c.RemotePort = ntohsField(binary.LittleEndian.Uint32(row[16:20]))
			c.PID = int(binary.LittleEndian.Uint32(row[20:24]))
		} else {
			c.State = tcpState(row[0:4])
			c.LocalAddress = ipv6String(row[4:20])
			c.LocalPort = ntohsField(binary.LittleEndian.Uint32(row[20:24]))
			c.RemoteAddress = ipv6String(row[24:40])
			c.RemotePort = ntohsField(binary.LittleEndian.Uint32(row[40:44]))
			c.PID = int(binary.LittleEndian.Uint32(row[44:48]))
		}

		c.ProcessName = pidName[c.PID]
		conns = append(conns, c)
	}
	return conns
}

func parseUDPTable(buf []byte, af uint32, pidName map[int]string) []Connection {
	if len(buf) < 4 {
		return nil
	}
	numEntries := binary.LittleEndian.Uint32(buf[:4])

	var rowSize int
	if af == windows.AF_INET {
		rowSize = int(unsafe.Sizeof(mibUDPRowOwnerPID{}))
	} else {
		rowSize = 28
	}

	data := buf[4:]
	var conns []Connection
	for i := uint32(0); i < numEntries; i++ {
		offset := int(i) * rowSize
		if offset+rowSize > len(data) {
			break
		}
		row := data[offset : offset+rowSize]

		var c Connection
		c.Protocol = UDP
		c.State = StateNA

		if af == windows.AF_INET {
			c.LocalAddress = ipv4String(binary.LittleEndian.Uint32(row[0:4]))
			c.LocalPort = ntohsField(binary.LittleEndian.Uint32(row[4:8]))
			c.PID = int(binary.LittleEndian.Uint32(row[8:12]))
		} else {
			c.LocalAddress = ipv6String(row[0:16])
			c.LocalPort = ntohsField(binary.LittleEndian.Uint32(row[16:20]))
			c.PID = int(binary.LittleEndian.Uint32(row[20:24]))
		}
		c.RemoteAddress = ""
		c.RemotePort = 0
		c.ProcessName = pidName[c.PID]
		conns = append(conns, c)
	}
	return conns
}

func tcpState(b []byte) TCPState {
	if len(b) < 4 {
		return StateUnknown
	}
	switch binary.LittleEndian.Uint32(b) {
	case 1:
		return StateClosed
	case 2:
		return StateListen
	case 3:
		return StateSynSent
	case 4:
		return StateSynReceived
	case 5:
		return StateEstablished
	case 6:
		return StateFinWait1
	case 7:
		return StateFinWait2
	case 8:
		return StateCloseWait
	case 9:
		return StateClosing
	case 10:
		return StateLastAck
	case 11:
		return StateTimeWait
	case 12:
		return StateDeleteTCB
	default:
		return StateUnknown
	}
}

func ipv4String(addr uint32) string {
	return net.IPv4(
		byte(addr),
		byte(addr>>8),
		byte(addr>>16),
		byte(addr>>24),
	).String()
}

func ipv6String(b []byte) string {
	ip := make(net.IP, 16)
	copy(ip, b)
	return ip.String()
}

func ntohsField(field uint32) int {
	v := uint16(field & 0xFFFF)
	return int(((v >> 8) & 0xFF) | ((v & 0xFF) << 8))
}
