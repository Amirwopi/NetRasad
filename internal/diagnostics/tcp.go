package diagnostics

import (
	"fmt"
	"net"
	"strconv"
	"time"
)

func (s *Service) TCPConnect(host string, port int, timeoutSec int) (*TCPResult, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}
	if timeoutSec <= 0 {
		timeoutSec = 5
	}

	result := &TCPResult{
		Host: host,
		Port: port,
	}

	address := net.JoinHostPort(host, strconv.Itoa(port))
	timeout := time.Duration(timeoutSec) * time.Second

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, timeout)
	elapsed := time.Since(start)
	result.RTT = float64(elapsed.Microseconds()) / 1000.0

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}

	result.Success = true
	_ = conn.Close()

	return result, nil
}
