//go:build linux

package netinfo

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
)

const nlaTypeMask = 0x3fff

// wifiSSIDs maps interface names to the SSID they are associated with, using
// nl80211 (no root needed, and independent of NetworkManager, iwd or
// wpa_supplicant). Interfaces that are not connected are omitted.
func wifiSSIDs() map[string]string {
	c, err := dialGenetlink()
	if err != nil {
		return map[string]string{}
	}
	defer c.close()

	// The family is missing when no wireless driver is loaded.
	family, err := c.familyID("nl80211")
	if err != nil {
		return map[string]string{}
	}
	msgs, err := c.request(family, unix.NL80211_CMD_GET_INTERFACE, unix.NLM_F_DUMP, nil)
	if err != nil {
		return map[string]string{}
	}
	return ssidsFromInterfaceDump(msgs)
}

func ssidsFromInterfaceDump(msgs [][]byte) map[string]string {
	out := map[string]string{}
	for _, m := range msgs {
		if len(m) < unix.GENL_HDRLEN {
			continue
		}
		attrs := parseAttrs(m[unix.GENL_HDRLEN:])
		name := strings.TrimRight(string(attrs[unix.NL80211_ATTR_IFNAME]), "\x00")
		ssid := attrs[unix.NL80211_ATTR_SSID]
		// Hidden networks can report an all-zero SSID element.
		if name == "" || len(bytes.Trim(ssid, "\x00")) == 0 {
			continue
		}
		out[name] = string(ssid)
	}
	return out
}

type genlConn struct {
	fd  int
	seq uint32
}

func dialGenetlink() (*genlConn, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_GENERIC)
	if err != nil {
		return nil, err
	}
	if err := unix.Bind(fd, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		unix.Close(fd)
		return nil, err
	}
	// This runs on the once-a-second polling path, so never block on a reply for long.
	tv := unix.Timeval{Sec: 1}
	if err := unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
		unix.Close(fd)
		return nil, err
	}
	return &genlConn{fd: fd}, nil
}

func (c *genlConn) close() {
	unix.Close(c.fd)
}

func (c *genlConn) familyID(name string) (uint16, error) {
	attr := encodeAttr(unix.CTRL_ATTR_FAMILY_NAME, append([]byte(name), 0))
	msgs, err := c.request(unix.GENL_ID_CTRL, unix.CTRL_CMD_GETFAMILY, 0, attr)
	if err != nil {
		return 0, err
	}
	for _, m := range msgs {
		if len(m) < unix.GENL_HDRLEN {
			continue
		}
		if id := parseAttrs(m[unix.GENL_HDRLEN:])[unix.CTRL_ATTR_FAMILY_ID]; len(id) >= 2 {
			return binary.NativeEndian.Uint16(id), nil
		}
	}
	return 0, fmt.Errorf("generic netlink family %q not found", name)
}

// request sends a generic netlink command and returns the payload (generic
// netlink header plus attributes) of every reply message.
func (c *genlConn) request(family uint16, cmd uint8, flags uint16, attrs []byte) ([][]byte, error) {
	c.seq++
	msg := make([]byte, unix.NLMSG_HDRLEN+unix.GENL_HDRLEN+len(attrs))
	binary.NativeEndian.PutUint32(msg[0:4], uint32(len(msg)))
	binary.NativeEndian.PutUint16(msg[4:6], family)
	binary.NativeEndian.PutUint16(msg[6:8], unix.NLM_F_REQUEST|flags)
	binary.NativeEndian.PutUint32(msg[8:12], c.seq)
	msg[unix.NLMSG_HDRLEN] = cmd
	msg[unix.NLMSG_HDRLEN+1] = 1
	copy(msg[unix.NLMSG_HDRLEN+unix.GENL_HDRLEN:], attrs)
	if err := unix.Sendto(c.fd, msg, 0, &unix.SockaddrNetlink{Family: unix.AF_NETLINK}); err != nil {
		return nil, err
	}

	dump := flags&unix.NLM_F_DUMP != 0
	buf := make([]byte, 64*1024)
	var out [][]byte
	for {
		n, _, err := unix.Recvfrom(c.fd, buf, 0)
		if err != nil {
			return nil, err
		}
		msgs, err := syscall.ParseNetlinkMessage(buf[:n])
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			if m.Header.Seq != c.seq {
				continue
			}
			switch m.Header.Type {
			case unix.NLMSG_DONE:
				return out, nil
			case unix.NLMSG_ERROR:
				if len(m.Data) >= 4 {
					if errno := int32(binary.NativeEndian.Uint32(m.Data[:4])); errno != 0 {
						return nil, syscall.Errno(-errno)
					}
				}
				return out, nil
			}
			out = append(out, append([]byte(nil), m.Data...))
			if !dump {
				return out, nil
			}
		}
	}
}

func encodeAttr(typ uint16, data []byte) []byte {
	l := unix.NLA_HDRLEN + len(data)
	b := make([]byte, nlaAlign(l))
	binary.NativeEndian.PutUint16(b[0:2], uint16(l))
	binary.NativeEndian.PutUint16(b[2:4], typ)
	copy(b[unix.NLA_HDRLEN:], data)
	return b
}

func parseAttrs(b []byte) map[uint16][]byte {
	attrs := make(map[uint16][]byte)
	for len(b) >= unix.NLA_HDRLEN {
		l := int(binary.NativeEndian.Uint16(b[0:2]))
		if l < unix.NLA_HDRLEN || l > len(b) {
			break
		}
		attrs[binary.NativeEndian.Uint16(b[2:4])&nlaTypeMask] = b[unix.NLA_HDRLEN:l]
		next := nlaAlign(l)
		if next >= len(b) {
			break
		}
		b = b[next:]
	}
	return attrs
}

func nlaAlign(n int) int {
	return (n + unix.NLA_ALIGNTO - 1) &^ (unix.NLA_ALIGNTO - 1)
}
