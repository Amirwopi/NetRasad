package application

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
	"github.com/netrasad/netrasad/internal/ports"
	"github.com/netrasad/netrasad/internal/traffic"
)

type TrafficService struct {
	monitor *traffic.Monitor
	source  ports.TrafficSource
	store   ports.DatabaseStorage
	log     *slog.Logger
	mu      sync.Mutex
	running bool
}

func NewTrafficService(source ports.TrafficSource, store ports.DatabaseStorage, classifier *traffic.Classifier, opts traffic.MonitorOptions, log *slog.Logger) *TrafficService {
	monitor := traffic.NewMonitor(source, store, classifier, opts, log)
	return &TrafficService{
		monitor: monitor,
		source:  source,
		store:   store,
		log:     log,
	}
}

func (s *TrafficService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	if err := s.monitor.Start(ctx); err != nil {
		return err
	}
	s.running = true
	return nil
}

func (s *TrafficService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.running = false
	return s.monitor.Stop()
}

func (s *TrafficService) GetDashboardState() domain.TrafficSnapshot {
	return s.monitor.Snapshot()
}

func (s *TrafficService) GetInterfaces(ctx context.Context) ([]domain.NetworkInterface, error) {
	return s.source.Interfaces(ctx)
}

func (s *TrafficService) GetRateHistory() []domain.RateSample {
	return s.monitor.RateHistory()
}

func (s *TrafficService) GetTrafficHistory(ctx context.Context, start, end time.Time, resolution string) ([]domain.TrafficSample, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.QueryTraffic(ctx, ports.TrafficQuery{
		StartTime:  start,
		EndTime:    end,
		Resolution: resolution,
	})
}

func (s *TrafficService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *TrafficService) GetMetrics() traffic.Metrics {
	return s.monitor.Metrics()
}

func (s *TrafficService) GetActiveInterfaces() []domain.NetworkInterface {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	all, err := s.source.Interfaces(ctx)
	if err != nil || len(all) == 0 {
		return s.monitor.ActiveInterfaces()
	}
	out := make([]domain.NetworkInterface, 0, len(all))
	for _, iface := range all {
		if traffic.IsActiveInterface(iface) {
			out = append(out, iface)
		}
	}
	return out
}

func (s *TrafficService) SetSelectedInterfaces(ids []string) {
	s.monitor.SetSelectedInterfaces(ids)
}

func (s *TrafficService) GetSelectedInterfaces() []string {
	return s.monitor.GetSelectedInterfaces()
}
