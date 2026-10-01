package ports

import (
	"context"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
)

type TrafficSource interface {
	Start(ctx context.Context) error

	Stop() error

	Interfaces(ctx context.Context) ([]domain.NetworkInterface, error)

	ReadStats(ctx context.Context, interfaceID string) (domain.TrafficCounters, error)

	ReadAllStats(ctx context.Context) ([]domain.TrafficCounters, error)
}

type ProcessTrafficSource interface {
	Start(ctx context.Context) error
	Stop() error
	Processes(ctx context.Context) ([]domain.Process, error)
}

type ConnectionSource interface {
	Connections(ctx context.Context) ([]domain.Connection, error)
}

type TrafficQuery struct {
	StartTime    time.Time
	EndTime      time.Time
	InterfaceIDs []string
	Category     domain.TrafficCategory
	Resolution   string
}

type DatabaseStorage interface {
	SaveTrafficSamples(ctx context.Context, samples []domain.TrafficSample) error

	QueryTraffic(ctx context.Context, q TrafficQuery) ([]domain.TrafficSample, error)

	SaveApplicationUsage(ctx context.Context, usage []domain.ProcessTrafficSample) error

	SaveQuotaEvent(ctx context.Context, event domain.QuotaEvent) error

	Backup(ctx context.Context, destination string) error

	Restore(ctx context.Context, source string) error

	CheckIntegrity(ctx context.Context) error

	Close() error
}

type Notifier interface {
	Notify(ctx context.Context, n domain.Notification) error
}

type SpeedTester interface {
	Run(ctx context.Context, opts domain.SpeedTestOptions) (domain.SpeedTestResult, error)
}

type RouterMonitor interface {
	Start(ctx context.Context) error
	Stop() error
	Devices(ctx context.Context) ([]domain.RouterDevice, error)
}

type SyncService interface {
	Start(ctx context.Context) error
	Stop() error
	Peers(ctx context.Context) ([]domain.SyncDevice, error)
}

var ErrUnsupported = errUnsupported{}

type errUnsupported struct{}

func (errUnsupported) Error() string { return "netrasad: capability unsupported on this platform" }
