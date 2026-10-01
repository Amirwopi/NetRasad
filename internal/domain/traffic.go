package domain

import "time"

type InterfaceType uint32

const (
	InterfaceTypeUnknown  InterfaceType = 0
	InterfaceTypeEthernet InterfaceType = 6
	InterfaceTypeWiFi     InterfaceType = 71
	InterfaceTypeLoopback InterfaceType = 24
	InterfaceTypeTunnel   InterfaceType = 131
	InterfaceTypePPP      InterfaceType = 23
	InterfaceTypeOther    InterfaceType = 1
)

type NetworkInterface struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	HardwareAddr string        `json:"hardwareAddr"`
	Index        uint32        `json:"index"`
	Type         InterfaceType `json:"type"`
	IsUp         bool          `json:"isUp"`
	IsLoopback   bool          `json:"isLoopback"`
	IsVirtual    bool          `json:"isVirtual"`
	IsVPN        bool          `json:"isVPN"`

	MediaConnected bool `json:"mediaConnected"`

	IPAddresses []string `json:"ipAddresses"`

	GatewayIP string `json:"gatewayIP"`

	SSID string `json:"ssid"`
}

type TrafficCounters struct {
	Timestamp     time.Time `json:"timestamp"`
	InterfaceID   string    `json:"interfaceId"`
	DownloadBytes uint64    `json:"downloadBytes"`
	UploadBytes   uint64    `json:"uploadBytes"`
}

type TrafficDirection int

const (
	DirectionDownload TrafficDirection = -1
	DirectionUpload   TrafficDirection = 1
)

type TrafficCategory int

const (
	CategoryInternet TrafficCategory = 0
	CategoryLAN      TrafficCategory = 1
	CategoryLoopback TrafficCategory = 2
	CategoryIgnored  TrafficCategory = 3
)

type TrafficSample struct {
	ID            int64           `json:"id"`
	Timestamp     time.Time       `json:"timestamp"`
	InterfaceID   string          `json:"interfaceId"`
	DownloadBytes uint64          `json:"downloadBytes"`
	UploadBytes   uint64          `json:"uploadBytes"`
	Category      TrafficCategory `json:"category"`
}

type RateSample struct {
	Timestamp   time.Time `json:"timestamp"`
	DownloadBPS float64   `json:"downloadBps"`
	UploadBPS   float64   `json:"uploadBps"`
}

type InterfaceStats struct {
	InterfaceID   string    `json:"interfaceId"`
	DownloadBytes uint64    `json:"downloadBytes"`
	UploadBytes   uint64    `json:"uploadBytes"`
	DownloadRate  float64   `json:"downloadRate"`
	UploadRate    float64   `json:"uploadRate"`
	PeakDownRate  float64   `json:"peakDownRate"`
	PeakUpRate    float64   `json:"peakUpRate"`
	LastUpdate    time.Time `json:"lastUpdate"`
}

type TrafficSnapshot struct {
	Timestamp     time.Time                 `json:"timestamp"`
	TotalDownload uint64                    `json:"totalDownload"`
	TotalUpload   uint64                    `json:"totalUpload"`
	DownloadRate  float64                   `json:"downloadRate"`
	UploadRate    float64                   `json:"uploadRate"`
	Interfaces    map[string]InterfaceStats `json:"interfaces"`
}
