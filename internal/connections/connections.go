package connections

type Protocol string

const (
	TCP Protocol = "tcp"
	UDP Protocol = "udp"
)

type TCPState string

const (
	StateClosed      TCPState = "CLOSED"
	StateListen      TCPState = "LISTEN"
	StateSynSent     TCPState = "SYN_SENT"
	StateSynReceived TCPState = "SYN_RECEIVED"
	StateEstablished TCPState = "ESTABLISHED"
	StateFinWait1    TCPState = "FIN_WAIT1"
	StateFinWait2    TCPState = "FIN_WAIT2"
	StateCloseWait   TCPState = "CLOSE_WAIT"
	StateClosing     TCPState = "CLOSING"
	StateLastAck     TCPState = "LAST_ACK"
	StateTimeWait    TCPState = "TIME_WAIT"
	StateDeleteTCB   TCPState = "DELETE_TCB"
	StateUnknown     TCPState = "UNKNOWN"
	StateNA TCPState = "N/A"
)

type Connection struct {
	Protocol      Protocol
	LocalAddress  string
	LocalPort     int
	RemoteAddress string
	RemotePort    int
	State         TCPState
	PID           int
	ProcessName   string
}

type Service struct {
	procMon *ProcessMonitor
}

func NewService() *Service {
	s := &Service{}
	s.procMon = NewProcessMonitor(s)
	return s
}

func (s *Service) GetProcessTraffic() []ProcessTraffic {
	if s.procMon == nil {
		return nil
	}
	return s.procMon.GetProcessStats()
}
