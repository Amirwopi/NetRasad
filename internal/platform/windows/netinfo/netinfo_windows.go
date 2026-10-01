//go:build windows

package netinfo

import (
	"encoding/binary"
	"fmt"
	"net"
	"unsafe"

	"golang.org/x/sys/windows"
)

type AdapterInfo struct {
	IPAddresses []string
	GatewayIP string
	SSID string
}

var (
	iphlpapiDLL = windows.NewLazySystemDLL("iphlpapi.dll")
	wlanapiDLL  = windows.NewLazySystemDLL("wlanapi.dll")

	procGetAdaptersAddresses       = iphlpapiDLL.NewProc("GetAdaptersAddresses")
	procFreeMibTable               = iphlpapiDLL.NewProc("FreeMibTable")
	procConvertInterfaceGuidToLuid = iphlpapiDLL.NewProc("ConvertInterfaceGuidToLuid")

	procWlanOpenHandle     = wlanapiDLL.NewProc("WlanOpenHandle")
	procWlanCloseHandle    = wlanapiDLL.NewProc("WlanCloseHandle")
	procWlanEnumInterfaces = wlanapiDLL.NewProc("WlanEnumInterfaces")
	procWlanQueryInterface = wlanapiDLL.NewProc("WlanQueryInterface")
)

const (
	gaaFlagIncludePrefix   = 0x00000010
	gaaFlagIncludeGateways = 0x00000080
	gaaFlags               = gaaFlagIncludePrefix | gaaFlagIncludeGateways

	errorSuccess        = 0
	errorBufferOverflow = 111

	afInet  = 2
	afInet6 = 23

	ifTypeIEEE80211 = 71
)

const (
	offAdapterNext                = 8
	offAdapterAdapterName         = 16
	offAdapterFirstUnicastAddress = 24
	offAdapterFirstGatewayAddress = 200
	offAdapterLuid                = 208
	offAdapterIfType              = 92
)

const (
	offUnicastNext              = 8
	offUnicastAddressLpSockaddr = 16
)

const (
	offGatewayNext              = 0
	offGatewayAddressLpSockaddr = 8
)

const (
	sockaddrLpSockaddr      = 0
	sockaddrISockaddrLength = 8
)

const (
	wlanIntfOpcodeCurrentConnection = 7
	wlanInterfaceStateConnected     = 1
)

func GetAdapterInfo() map[uint64]AdapterInfo {
	out := map[uint64]AdapterInfo{}

	var size uint32 = 0
	r1, _, _ := procGetAdaptersAddresses.Call(
		uintptr(0),
		uintptr(gaaFlags),
		uintptr(0),
		uintptr(0),
		uintptr(unsafe.Pointer(&size)),
	)
	if r1 != errorBufferOverflow && r1 != errorSuccess {
		return out
	}
	if size == 0 {
		return out
	}

	alignedSize := (size + 7) / 8
	alignedBuf := make([]uint64, alignedSize)
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&alignedBuf[0])), size)
	for {
		r1, _, _ = procGetAdaptersAddresses.Call(
			uintptr(0),
			uintptr(gaaFlags),
			uintptr(0),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&size)),
		)
		if r1 == errorSuccess {
			break
		}
		if r1 == errorBufferOverflow {
			alignedSize = (size + 7) / 8
			alignedBuf = make([]uint64, alignedSize)
			buf = unsafe.Slice((*byte)(unsafe.Pointer(&alignedBuf[0])), size)
			continue
		}
		return out
	}

	cur := unsafe.Pointer(&buf[0])
	for cur != nil {
		luid := readUint64(cur, offAdapterLuid)
		if luid == 0 {
			cur = readPointer(cur, offAdapterNext)
			continue
		}

		info := AdapterInfo{IPAddresses: []string{}}

		ua := readPointer(cur, offAdapterFirstUnicastAddress)
		for ua != nil {
			sockaddrPtr := readPointer(ua, offUnicastAddressLpSockaddr)
			if sockaddrPtr != nil {
				if ip := sockaddrToIP(sockaddrPtr); ip != "" {
					info.IPAddresses = append(info.IPAddresses, ip)
				}
			}
			ua = readPointer(ua, offUnicastNext)
		}

		ga := readPointer(cur, offAdapterFirstGatewayAddress)
		for ga != nil && info.GatewayIP == "" {
			sockaddrPtr := readPointer(ga, offGatewayAddressLpSockaddr)
			if sockaddrPtr != nil {
				if ip := sockaddrToIPv4(sockaddrPtr); ip != "" {
					info.GatewayIP = ip
				}
			}
			ga = readPointer(ga, offGatewayNext)
		}

		out[luid] = info
		cur = readPointer(cur, offAdapterNext)
	}

	ssidByLuid := collectWiFiSSIDs()
	for luid, ssid := range ssidByLuid {
		if existing, ok := out[luid]; ok {
			existing.SSID = ssid
			out[luid] = existing
		}
	}

	return out
}

func GetWiFiSSID() string {
	ssids := collectWiFiSSIDs()
	for _, ssid := range ssids {
		if ssid != "" {
			return ssid
		}
	}
	return ""
}

func collectWiFiSSIDs() map[uint64]string {
	out := map[uint64]string{}

	var negotiatedVersion uint32
	var handle uintptr
	r1, _, _ := procWlanOpenHandle.Call(
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&negotiatedVersion)),
		uintptr(unsafe.Pointer(&handle)),
	)
	if r1 != errorSuccess || handle == 0 {
		return out
	}
	defer procWlanCloseHandle.Call(handle, uintptr(0))

	var ifaceListPtr unsafe.Pointer
	r1, _, _ = procWlanEnumInterfaces.Call(
		handle,
		uintptr(0),
		uintptr(unsafe.Pointer(&ifaceListPtr)),
	)
	if r1 != errorSuccess || ifaceListPtr == nil {
		return out
	}
	defer windows.LocalFree(windows.Handle(ifaceListPtr))

	const (
		ifaceListArrayOffset = 8
		ifaceInfoSize        = 532
		ifaceInfoStateOffset = 528
		ifaceInfoGuidOffset  = 0
		guidSize             = 16
	)

	numItems := *(*uint32)(ifaceListPtr)
	base := unsafe.Add(ifaceListPtr, ifaceListArrayOffset)

	for i := uint32(0); i < numItems; i++ {
		entry := unsafe.Add(base, uintptr(i)*ifaceInfoSize)
		state := *(*uint32)(unsafe.Add(entry, ifaceInfoStateOffset))
		if state != wlanInterfaceStateConnected {
			continue
		}

		var guid [guidSize]byte
		copy(guid[:], (*[guidSize]byte)(unsafe.Add(entry, ifaceInfoGuidOffset))[:])

		var luid uint64
		r1, _, _ = procConvertInterfaceGuidToLuid.Call(
			uintptr(unsafe.Pointer(&guid[0])),
			uintptr(unsafe.Pointer(&luid)),
		)

		var dataSize uint32
		var dataPtr unsafe.Pointer
		var opcodeType uint32
		r1, _, _ = procWlanQueryInterface.Call(
			handle,
			uintptr(unsafe.Pointer(&guid[0])),
			uintptr(wlanIntfOpcodeCurrentConnection),
			uintptr(0),
			uintptr(unsafe.Pointer(&dataSize)),
			uintptr(unsafe.Pointer(&dataPtr)),
			uintptr(unsafe.Pointer(&opcodeType)),
		)
		if r1 != errorSuccess || dataPtr == nil {
			continue
		}
		defer windows.LocalFree(windows.Handle(dataPtr))

		const (
			connAttrAssocOffset = 520
			dot11SsidLenOffset  = connAttrAssocOffset
			dot11SsidDataOffset = connAttrAssocOffset + 4
			dot11SsidMaxLen     = 32
		)

		ssidLen := *(*uint32)(unsafe.Add(dataPtr, dot11SsidLenOffset))
		if ssidLen == 0 || ssidLen > dot11SsidMaxLen {
			continue
		}
		ssidBytes := (*[dot11SsidMaxLen]byte)(unsafe.Add(dataPtr, dot11SsidDataOffset))[:ssidLen:ssidLen]
		ssid := string(ssidBytes)

		out[luid] = ssid
	}

	return out
}

func sockaddrToIP(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	family := *(*uint16)(p)
	switch family {
	case afInet:
		return sockaddrToIPv4(p)
	case afInet6:
		return sockaddrToIPv6(p)
	}
	return ""
}

func sockaddrToIPv4(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	family := *(*uint16)(p)
	if family != afInet {
		return ""
	}
	ipBytes := (*[4]byte)(unsafe.Add(p, 4))[:]
	return net.IP(ipBytes).String()
}

func sockaddrToIPv6(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	family := *(*uint16)(p)
	if family != afInet6 {
		return ""
	}
	ipBytes := (*[16]byte)(unsafe.Add(p, 8))[:]
	return net.IP(ipBytes).String()
}

func readPointer(base unsafe.Pointer, offset uintptr) unsafe.Pointer {
	if base == nil {
		return nil
	}
	b := (*[8]byte)(unsafe.Add(base, offset))
	addr := binary.LittleEndian.Uint64(b[:])
	if addr == 0 {
		return nil
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(&addr))
}

func readUint64(base unsafe.Pointer, offset uintptr) uint64 {
	if base == nil {
		return 0
	}
	b := (*[8]byte)(unsafe.Add(base, offset))
	return binary.LittleEndian.Uint64(b[:])
}

func readUint32(base unsafe.Pointer, offset uintptr) uint32 {
	if base == nil {
		return 0
	}
	b := (*[4]byte)(unsafe.Add(base, offset))
	return binary.LittleEndian.Uint32(b[:])
}

func readUint16(base unsafe.Pointer, offset uintptr) uint16 {
	if base == nil {
		return 0
	}
	b := (*[2]byte)(unsafe.Add(base, offset))
	return binary.LittleEndian.Uint16(b[:])
}

func readByteSlice(base unsafe.Pointer, offset, n uintptr) []byte {
	if base == nil || n == 0 {
		return nil
	}
	src := (*[1 << 30]byte)(unsafe.Add(base, offset))[:n:n]
	out := make([]byte, n)
	copy(out, src)
	return out
}

func readUint16BE(base unsafe.Pointer, offset uintptr) uint16 {
	if base == nil {
		return 0
	}
	b := (*[2]byte)(unsafe.Add(base, offset))
	return binary.BigEndian.Uint16(b[:])
}

func formatUint16String(p unsafe.Pointer, max uintptr) string {
	if p == nil || max == 0 {
		return ""
	}
	buf := (*[1 << 16]uint16)(p)[: max/2 : max/2]
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return windows.UTF16ToString(buf[:n])
}

var (
	_ = fmt.Sprintf
	_ = binary.BigEndian
)
