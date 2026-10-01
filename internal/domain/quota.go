package domain

import "time"

type QuotaPeriod string

const (
	QuotaDaily   QuotaPeriod = "daily"
	QuotaWeekly  QuotaPeriod = "weekly"
	QuotaMonthly QuotaPeriod = "monthly"
	QuotaCustom  QuotaPeriod = "custom"
)

type QuotaDirection string

const (
	QuotaDownload QuotaDirection = "download"
	QuotaUpload   QuotaDirection = "upload"
	QuotaCombined QuotaDirection = "combined"
)

type Quota struct {
	ID         int64          `json:"id"`
	Name       string         `json:"name"`
	Period     QuotaPeriod    `json:"period"`
	Direction  QuotaDirection `json:"direction"`
	LimitBytes uint64         `json:"limitBytes"`
	Enabled    bool           `json:"enabled"`
	Thresholds []float64      `json:"thresholds"`
}

type QuotaEvent struct {
	ID         int64     `json:"id"`
	QuotaID    int64     `json:"quotaId"`
	Timestamp  time.Time `json:"timestamp"`
	Percent    float64   `json:"percent"`
	UsedBytes  uint64    `json:"usedBytes"`
	LimitBytes uint64    `json:"limitBytes"`
	Notified   bool      `json:"notified"`
}

type QuotaStatus struct {
	Quota       Quota     `json:"quota"`
	UsedBytes   uint64    `json:"usedBytes"`
	Percent     float64   `json:"percent"`
	Remaining   uint64    `json:"remaining"`
	PeriodStart time.Time `json:"periodStart"`
	PeriodEnd   time.Time `json:"periodEnd"`
	Exceeded    bool      `json:"exceeded"`
}
