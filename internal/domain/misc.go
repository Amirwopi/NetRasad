package domain

import "time"

type SpeedTestOptions struct {
	DownloadOnly bool   `json:"downloadOnly,omitempty"`
	UploadOnly   bool   `json:"uploadOnly,omitempty"`
	Provider     string `json:"provider,omitempty"`
	DurationSec  int    `json:"durationSec,omitempty"`
}

type SpeedTestResult struct {
	ID          int64     `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Provider    string    `json:"provider"`
	Server      string    `json:"server"`
	DownloadBPS float64   `json:"downloadBps"`
	UploadBPS   float64   `json:"uploadBps"`
	LatencyMs   float64   `json:"latencyMs"`
	JitterMs    float64   `json:"jitterMs,omitempty"`
	DurationSec float64   `json:"durationSec"`
}

type ConnectionQualitySample struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Target    string    `json:"target"`
	LatencyMs float64   `json:"latencyMs"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

type ConnectionQualitySummary struct {
	Target          string        `json:"target"`
	UptimePercent   float64       `json:"uptimePercent"`
	AvgLatencyMs    float64       `json:"avgLatencyMs"`
	MedianLatencyMs float64       `json:"medianLatencyMs"`
	P95LatencyMs    float64       `json:"p95LatencyMs"`
	MaxLatencyMs    float64       `json:"maxLatencyMs"`
	TotalDowntime   time.Duration `json:"totalDowntime"`
	LongestDowntime time.Duration `json:"longestDowntime"`
	Samples         int           `json:"samples"`
}

type RouterDevice struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"`
	Enabled  bool   `json:"enabled"`
}

type RouterInterface struct {
	ID        int64  `json:"id"`
	DeviceID  int64  `json:"deviceId"`
	Index     int    `json:"index"`
	Name      string `json:"name"`
	IsWAN     bool   `json:"isWan"`
	Monitored bool   `json:"monitored"`
}

type RouterTrafficSample struct {
	Timestamp     time.Time `json:"timestamp"`
	DeviceID      int64     `json:"deviceId"`
	InterfaceID   int64     `json:"interfaceId"`
	DownloadBytes uint64    `json:"downloadBytes"`
	UploadBytes   uint64    `json:"uploadBytes"`
}

type UsageReport struct {
	StartTime     time.Time            `json:"startTime"`
	EndTime       time.Time            `json:"endTime"`
	Download      uint64               `json:"download"`
	Upload        uint64               `json:"upload"`
	Total         uint64               `json:"total"`
	AvgDownRate   float64              `json:"avgDownRate"`
	AvgUpRate     float64              `json:"avgUpRate"`
	PeakDownRate  float64              `json:"peakDownRate"`
	PeakUpRate    float64              `json:"peakUpRate"`
	ByInterface   []InterfaceBreakdown `json:"byInterface,omitempty"`
	ByApplication []AppBreakdown       `json:"byApplication,omitempty"`
}

type InterfaceBreakdown struct {
	InterfaceID string `json:"interfaceId"`
	Download    uint64 `json:"download"`
	Upload      uint64 `json:"upload"`
}

type AppBreakdown struct {
	ApplicationID string `json:"applicationId"`
	Name          string `json:"name"`
	Download      uint64 `json:"download"`
	Upload        uint64 `json:"upload"`
}

type SyncDevice struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	Address  string    `json:"address"`
	Paired   bool      `json:"paired"`
	LastSeen time.Time `json:"lastSeen"`
}

type SyncEvent struct {
	EventID       string    `json:"eventId"`
	DeviceID      string    `json:"deviceId"`
	Timestamp     time.Time `json:"timestamp"`
	CounterType   string    `json:"counterType"`
	Payload       []byte    `json:"payload"`
	SchemaVersion int       `json:"schemaVersion"`
}

type Notification struct {
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	Severity  Severity  `json:"severity"`
	Category  string    `json:"category"`
	Timestamp time.Time `json:"timestamp"`
}

type Severity int

const (
	SeverityInfo  Severity = 0
	SeverityWarn  Severity = 1
	SeverityError Severity = 2
)
