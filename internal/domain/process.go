package domain

import "time"

type ApplicationIdentity struct {
	StableID       string `json:"stableId"`
	Name           string `json:"name"`
	ExecutablePath string `json:"executablePath"`
	Publisher      string `json:"publisher,omitempty"`
	ProcessType    string `json:"processType,omitempty"`
}

type Process struct {
	PID           int                 `json:"pid"`
	Identity      ApplicationIdentity `json:"identity"`
	ParentPID     int                 `json:"parentPid,omitempty"`
	UploadBytes   uint64              `json:"uploadBytes"`
	DownloadBytes uint64              `json:"downloadBytes"`
	UploadRate    float64             `json:"uploadRate"`
	DownloadRate  float64             `json:"downloadRate"`
	FirstSeen     time.Time           `json:"firstSeen"`
	LastSeen      time.Time           `json:"lastSeen"`
}

type ProcessTrafficSample struct {
	ID            int64     `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	ApplicationID string    `json:"applicationId"`
	DownloadBytes uint64    `json:"downloadBytes"`
	UploadBytes   uint64    `json:"uploadBytes"`
}

type Protocol int

const (
	ProtocolTCP  Protocol = 6
	ProtocolUDP  Protocol = 17
	ProtocolICMP Protocol = 1
)

type ConnectionState string

const (
	StateListen      ConnectionState = "LISTEN"
	StateEstablished ConnectionState = "ESTABLISHED"
	StateTimeWait    ConnectionState = "TIME_WAIT"
	StateCloseWait   ConnectionState = "CLOSE_WAIT"
	StateClosed      ConnectionState = "CLOSED"
	StateSynSent     ConnectionState = "SYN_SENT"
	StateSynRecv     ConnectionState = "SYN_RECV"
	StateFinWait1    ConnectionState = "FIN_WAIT_1"
	StateFinWait2    ConnectionState = "FIN_WAIT_2"
	StateUnknown     ConnectionState = "UNKNOWN"
)

type Connection struct {
	ID              int64           `json:"id"`
	PID             int             `json:"pid"`
	Application     string          `json:"application"`
	Executable      string          `json:"executable"`
	Protocol        Protocol        `json:"protocol"`
	LocalAddress    string          `json:"localAddress"`
	LocalPort       uint16          `json:"localPort"`
	RemoteAddress   string          `json:"remoteAddress"`
	RemotePort      uint16          `json:"remotePort"`
	State           ConnectionState `json:"state"`
	UploadRate      float64         `json:"uploadRate"`
	DownloadRate    float64         `json:"downloadRate"`
	TotalUploaded   uint64          `json:"totalUploaded"`
	TotalDownloaded uint64          `json:"totalDownloaded"`
	FirstSeen       time.Time       `json:"firstSeen"`
	LastSeen        time.Time       `json:"lastSeen"`
}
