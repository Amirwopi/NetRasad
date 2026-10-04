package modem

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type TelnetClient struct {
	conn net.Conn
}

func NewTelnetClient(host string, port int) (*TelnetClient, error) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return nil, err
	}
	return &TelnetClient{conn: conn}, nil
}

func (c *TelnetClient) Close() error {
	return c.conn.Close()
}

func (c *TelnetClient) ReadUntil(prompts []string, timeout time.Duration) (string, error) {
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	var buf bytes.Buffer
	temp := make([]byte, 1024)
	for {
		n, err := c.conn.Read(temp)
		if n > 0 {
			// strip IAC (0xFF) basic commands to avoid messing up output
			var filtered []byte
			for i := 0; i < n; i++ {
				if temp[i] == 255 { // IAC
					if i+2 < n {
						i += 2 // Skip 3 bytes of negotiation (IAC DO/DONT/WILL/WONT command)
					} else {
						i += int(n) - i // Just skip the rest if we can't parse
					}
					continue
				}
				filtered = append(filtered, temp[i])
			}
			buf.Write(filtered)
			
			str := buf.String()
			for _, p := range prompts {
				if strings.Contains(str, p) {
					return str, nil
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return buf.String(), nil
			}
			return buf.String(), err
		}
	}
}

func (c *TelnetClient) Send(cmd string) error {
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err := c.conn.Write([]byte(cmd + "\r\n"))
	return err
}
