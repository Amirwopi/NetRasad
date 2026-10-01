package traffic

import (
	"time"

	"github.com/netrasad/netrasad/internal/domain"
)

func CounterDelta(previous, current uint64) uint64 {
	if current >= previous {
		return current - previous
	}
	return current
}

func ComputeRate(deltaBytes uint64, elapsed time.Duration) float64 {
	secs := elapsed.Seconds()
	if secs <= 0 {
		return 0
	}
	return float64(deltaBytes) / secs
}

type InterfaceState struct {
	Interface    domain.NetworkInterface
	LastCounters domain.TrafficCounters
	CumDown      uint64
	CumUp        uint64
	DownRate     float64
	UpRate       float64
	PeakDownRate float64
	PeakUpRate   float64
	LastUpdate   time.Time
	Started      bool
}

func (s *InterfaceState) Update(counters domain.TrafficCounters) {
	if !s.Started {
		s.LastCounters = counters
		s.Started = true
		s.LastUpdate = counters.Timestamp
		return
	}

	elapsed := counters.Timestamp.Sub(s.LastCounters.Timestamp)
	downDelta := CounterDelta(s.LastCounters.DownloadBytes, counters.DownloadBytes)
	upDelta := CounterDelta(s.LastCounters.UploadBytes, counters.UploadBytes)

	s.CumDown += downDelta
	s.CumUp += upDelta
	s.DownRate = ComputeRate(downDelta, elapsed)
	s.UpRate = ComputeRate(upDelta, elapsed)

	if s.DownRate > s.PeakDownRate {
		s.PeakDownRate = s.DownRate
	}
	if s.UpRate > s.PeakUpRate {
		s.PeakUpRate = s.UpRate
	}

	s.LastCounters = counters
	s.LastUpdate = counters.Timestamp
}

func (s *InterfaceState) ToStats() domain.InterfaceStats {
	return domain.InterfaceStats{
		InterfaceID:   s.Interface.ID,
		DownloadBytes: s.CumDown,
		UploadBytes:   s.CumUp,
		DownloadRate:  s.DownRate,
		UploadRate:    s.UpRate,
		PeakDownRate:  s.PeakDownRate,
		PeakUpRate:    s.PeakUpRate,
		LastUpdate:    s.LastUpdate,
	}
}
