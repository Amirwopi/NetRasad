package diagnostics

import (
	"testing"
)

func TestPing_EmptyHost(t *testing.T) {
	s := NewService()
	_, err := s.Ping("", 4)
	if err == nil {
		t.Fatal("expected error for empty host, got nil")
	}
}

func TestParseWindowsPing(t *testing.T) {
	output := `
Pinging 8.8.8.8 with 32 bytes of data:
Reply from 8.8.8.8: bytes=32 time=1ms TTL=57
Reply from 8.8.8.8: bytes=32 time=2ms TTL=57
Reply from 8.8.8.8: bytes=32 time<1ms TTL=57
Reply from 8.8.8.8: bytes=32 time=1ms TTL=57

Ping statistics for 8.8.8.8:
    Packets: Sent = 4, Received = 4, Lost = 0 (0% loss),
Approximate round trip times in milli-seconds:
    Minimum = 0ms, Maximum = 2ms, Average = 1ms
`
	r := &PingResult{
		Host: "8.8.8.8",
		Sent: 4,
		RTTs: []float64{},
	}
	parseWindowsPing(r, output)

	if r.Received != 4 {
		t.Errorf("Received = %d, want 4", r.Received)
	}
	if len(r.RTTs) != 4 {
		t.Fatalf("len(RTTs) = %d, want 4", len(r.RTTs))
	}
	// time<1ms should parse as 0
	if r.RTTs[2] != 0 {
		t.Errorf("RTTs[2] = %v, want 0 (from time<1ms)", r.RTTs[2])
	}
	if r.MinRTT != 0 {
		t.Errorf("MinRTT = %v, want 0", r.MinRTT)
	}
	if r.MaxRTT != 2 {
		t.Errorf("MaxRTT = %v, want 2", r.MaxRTT)
	}
	if r.AvgRTT != 1 {
		t.Errorf("AvgRTT = %v, want 1", r.AvgRTT)
	}
}

func TestParseWindowsPing_Timeout(t *testing.T) {
	output := `
Pinging 10.0.0.99 with 32 bytes of data:
Request timed out.
Request timed out.
Request timed out.
Request timed out.

Ping statistics for 10.0.0.99:
    Packets: Sent = 4, Received = 0, Lost = 4 (100% loss),
`
	r := &PingResult{
		Host: "10.0.0.99",
		Sent: 4,
		RTTs: []float64{},
	}
	parseWindowsPing(r, output)

	if r.Received != 0 {
		t.Errorf("Received = %d, want 0", r.Received)
	}
	if len(r.RTTs) != 0 {
		t.Errorf("len(RTTs) = %d, want 0", len(r.RTTs))
	}
}

func TestParseLinuxPing(t *testing.T) {
	output := `
PING 8.8.8.8 (8.8.8.8) 56(84) bytes of data.
64 bytes from 8.8.8.8: icmp_seq=1 ttl=57 time=0.123 ms
64 bytes from 8.8.8.8: icmp_seq=2 ttl=57 time=0.456 ms
64 bytes from 8.8.8.8: icmp_seq=3 ttl=57 time=0.789 ms
64 bytes from 8.8.8.8: icmp_seq=4 ttl=57 time=0.321 ms

--- 8.8.8.8 ping statistics ---
4 packets transmitted, 4 received, 0% packet loss, time 3004ms
rtt min/avg/max/mdev = 0.123/0.422/0.789/0.243 ms
`
	r := &PingResult{
		Host: "8.8.8.8",
		Sent: 4,
		RTTs: []float64{},
	}
	parseLinuxPing(r, output)

	if r.Received != 4 {
		t.Errorf("Received = %d, want 4", r.Received)
	}
	if len(r.RTTs) != 4 {
		t.Fatalf("len(RTTs) = %d, want 4", len(r.RTTs))
	}
	if r.RTTs[0] != 0.123 {
		t.Errorf("RTTs[0] = %v, want 0.123", r.RTTs[0])
	}
	if r.MinRTT != 0.123 {
		t.Errorf("MinRTT = %v, want 0.123", r.MinRTT)
	}
	if r.AvgRTT != 0.422 {
		t.Errorf("AvgRTT = %v, want 0.422", r.AvgRTT)
	}
	if r.MaxRTT != 0.789 {
		t.Errorf("MaxRTT = %v, want 0.789", r.MaxRTT)
	}
}

func TestParseWindowsTracert(t *testing.T) {
	output := `
Tracing route to 8.8.8.8 over a maximum of 30 hops:

  1     1 ms     1 ms     1 ms  192.168.1.1
  2     2 ms     3 ms     2 ms  10.0.0.1
  3    <1 ms    <1 ms    <1 ms  172.16.0.1
  4     *        *        *     Request timed out.
  5     5 ms     5 ms     5 ms  8.8.8.8

Trace complete.
`
	r := &TracerouteResult{
		Hops: []TracerouteHop{},
	}
	parseWindowsTracert(r, output)

	if len(r.Hops) != 5 {
		t.Fatalf("len(Hops) = %d, want 5", len(r.Hops))
	}

	// Hop 1
	if r.Hops[0].Hop != 1 {
		t.Errorf("Hops[0].Hop = %d, want 1", r.Hops[0].Hop)
	}
	if r.Hops[0].IP != "192.168.1.1" {
		t.Errorf("Hops[0].IP = %s, want 192.168.1.1", r.Hops[0].IP)
	}
	if r.Hops[0].RTT1 != 1 {
		t.Errorf("Hops[0].RTT1 = %v, want 1", r.Hops[0].RTT1)
	}

	// Hop 3 — <1 ms should parse as 0
	if r.Hops[2].RTT1 != 0 {
		t.Errorf("Hops[2].RTT1 = %v, want 0 (from <1 ms)", r.Hops[2].RTT1)
	}

	// Hop 4 — timeout
	if r.Hops[3].RTT1 != -1 {
		t.Errorf("Hops[3].RTT1 = %v, want -1 (timeout)", r.Hops[3].RTT1)
	}
}

func TestParseWindowsRTT(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"1", 1},
		{"42", 42},
		{"<1", 0},
		{"*", -1},
		{"ms", -1},
		{"abc", -1},
	}
	for _, tt := range tests {
		got := parseWindowsRTT(tt.input)
		if got != tt.want {
			t.Errorf("parseWindowsRTT(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseLinuxTraceroute(t *testing.T) {
	output := `
traceroute to 8.8.8.8 (8.8.8.8), 30 hops max, 60 byte packets
 1  192.168.1.1  0.123 ms  0.456 ms  0.789 ms
 2  10.0.0.1  1.234 ms  1.345 ms  1.456 ms
 3  * * *
 4  8.8.8.8  5.678 ms  5.789 ms  5.890 ms
`
	r := &TracerouteResult{
		Hops: []TracerouteHop{},
	}
	parseLinuxTraceroute(r, output)

	if len(r.Hops) != 4 {
		t.Fatalf("len(Hops) = %d, want 4", len(r.Hops))
	}

	if r.Hops[0].IP != "192.168.1.1" {
		t.Errorf("Hops[0].IP = %s, want 192.168.1.1", r.Hops[0].IP)
	}
	if r.Hops[0].RTT1 != 0.123 {
		t.Errorf("Hops[0].RTT1 = %v, want 0.123", r.Hops[0].RTT1)
	}

	// Hop 3 — all timeouts
	if r.Hops[2].RTT1 != -1 || r.Hops[2].RTT2 != -1 || r.Hops[2].RTT3 != -1 {
		t.Errorf("Hops[2] RTTs = %v/%v/%v, want -1/-1/-1", r.Hops[2].RTT1, r.Hops[2].RTT2, r.Hops[2].RTT3)
	}
}

func TestTraceroute_EmptyHost(t *testing.T) {
	s := NewService()
	_, err := s.Traceroute("", 30)
	if err == nil {
		t.Fatal("expected error for empty host, got nil")
	}
}

func TestDNSLookup_EmptyHost(t *testing.T) {
	s := NewService()
	_, err := s.DNSLookup("")
	if err == nil {
		t.Fatal("expected error for empty host, got nil")
	}
}

func TestTCPConnect_EmptyHost(t *testing.T) {
	s := NewService()
	_, err := s.TCPConnect("", 80, 5)
	if err == nil {
		t.Fatal("expected error for empty host, got nil")
	}
}

func TestTCPConnect_InvalidPort(t *testing.T) {
	s := NewService()
	tests := []int{0, -1, 70000, 99999}
	for _, port := range tests {
		_, err := s.TCPConnect("localhost", port, 5)
		if err == nil {
			t.Errorf("expected error for port %d, got nil", port)
		}
	}
}

func TestTCPConnect_LocalhostRefused(t *testing.T) {
	// Port 1 is almost certainly not listening, so we expect a refused
	// connection — but the function returns nil error with Success=false.
	s := NewService()
	result, err := s.TCPConnect("127.0.0.1", 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Error("expected Success=false for refused connection")
	}
	if result.Error == "" {
		t.Error("expected non-empty Error for refused connection")
	}
}

func TestIsDigit(t *testing.T) {
	if !isDigit('0') {
		t.Error("isDigit('0') = false, want true")
	}
	if !isDigit('9') {
		t.Error("isDigit('9') = false, want true")
	}
	if isDigit('a') {
		t.Error("isDigit('a') = true, want false")
	}
	if isDigit(' ') {
		t.Error("isDigit(' ') = true, want false")
	}
}
