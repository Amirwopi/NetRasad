package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/netrasad/netrasad/internal/application"
	"github.com/netrasad/netrasad/internal/connections"
	"github.com/netrasad/netrasad/internal/diagnostics"
	"github.com/netrasad/netrasad/internal/domain"
	"github.com/netrasad/netrasad/internal/platform"
	"github.com/netrasad/netrasad/internal/platform/windows/autostart"
	"github.com/netrasad/netrasad/internal/platform/windows/pingoverlay"
	"github.com/netrasad/netrasad/internal/platform/windows/taskbar"
	"github.com/netrasad/netrasad/internal/storage/sqlite"
	"github.com/netrasad/netrasad/internal/traffic"
)

type App struct {
	log            *slog.Logger
	trafficSvc     *application.TrafficService
	storage        *sqlite.Storage
	dbPath         string
	diagSvc        *diagnostics.Service
	connSvc        *connections.Service
	taskbarOverlay *taskbar.TaskbarOverlay
	pingOverlay    *pingoverlay.PingOverlay

	mu            sync.Mutex
	ctx           context.Context
	cancel        context.CancelFunc
	eventTicker   *time.Ticker
	emitEvent     func(string, interface{})
	selectionPath string
}

func New() *App {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return &App{
		log:            logger,
		diagSvc:        diagnostics.NewService(),
		connSvc:        connections.NewService(),
		taskbarOverlay: taskbar.GetOverlay(),
		pingOverlay:    pingoverlay.GetOverlay(),
	}
}

func (a *App) Context() context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ctx
}

func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	a.ctx = ctx
	a.mu.Unlock()

	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	dbDir := filepath.Join(configDir, "NetRasad")
	a.dbPath = filepath.Join(dbDir, "netrasad.db")
	a.selectionPath = filepath.Join(dbDir, "selection.json")

	_ = os.MkdirAll(dbDir, 0755)

	db, err := sqlite.Open(ctx, a.dbPath, a.log)
	if err != nil {
		a.log.Error("failed to open database", "error", err, "path", a.dbPath)
	} else {
		a.storage = sqlite.NewStorage(db)
	}

	source, err := platform.NewTrafficSource()
	if err != nil {
		a.log.Error("failed to create traffic source", "error", err)
		return
	}

	classifier := traffic.NewClassifier(traffic.ClassificationRules{
		IgnoreLoopback: true,
	})

	opts := traffic.DefaultMonitorOptions()
	a.trafficSvc = application.NewTrafficService(source, a.storage, classifier, opts, a.log)

	a.loadSelection()

	innerCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	if err := a.trafficSvc.Start(innerCtx); err != nil {
		a.log.Error("failed to start traffic monitoring", "error", err)
		return
	}

	if a.taskbarOverlay != nil {
		_ = a.taskbarOverlay.Start()
	}

	go a.eventLoop(innerCtx)

	a.log.Info("NetRasad started", "db", a.dbPath)
}

func (a *App) Shutdown(ctx context.Context) {
	if a.cancel != nil {
		a.cancel()
	}
	if a.taskbarOverlay != nil {
		a.taskbarOverlay.Stop()
	}
	if a.pingOverlay != nil {
		a.pingOverlay.Stop()
	}
	if a.trafficSvc != nil {
		if err := a.trafficSvc.Stop(); err != nil {
			a.log.Error("traffic service stop error", "error", err)
		}
	}
	if a.storage != nil {
		if err := a.storage.Close(); err != nil {
			a.log.Error("database close error", "error", err)
		}
	}
	a.log.Info("NetRasad stopped")
}

func (a *App) eventLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if a.trafficSvc == nil {
				continue
			}
			snap := a.trafficSvc.GetDashboardState()

			if a.taskbarOverlay != nil {
				a.taskbarOverlay.UpdateRates(snap.DownloadRate, snap.UploadRate)
			}

			if a.connSvc != nil {
				procs := a.connSvc.GetProcessTraffic()
				if a.emitEvent != nil {
					a.emitEvent("traffic:process_updated", procs)
				}
				if a.taskbarOverlay != nil && len(procs) > 0 {
					sortedProcs := make([]connections.ProcessTraffic, len(procs))
					copy(sortedProcs, procs)
					sort.Slice(sortedProcs, func(i, j int) bool {
						return (sortedProcs[i].DownloadBps + sortedProcs[i].UploadBps) > (sortedProcs[j].DownloadBps + sortedProcs[j].UploadBps)
					})
					topEntries := make([]taskbar.TopProcessEntry, 0, 3)
					for _, p := range sortedProcs {
						if len(topEntries) >= 3 {
							break
						}
						topEntries = append(topEntries, taskbar.TopProcessEntry{
							Name:        p.ProcessName,
							DownloadBps: p.DownloadBps,
							UploadBps:   p.UploadBps,
						})
					}
					a.taskbarOverlay.UpdateTopProcesses(topEntries)
				}
			}

			if a.emitEvent != nil {
				a.emitEvent("traffic:updated", snap)
			}
		}
	}
}

func (a *App) SetTaskbarWidgetEnabled(enabled bool) {
	if a.taskbarOverlay != nil {
		a.taskbarOverlay.SetEnabled(enabled)
	}
}

func (a *App) GetTaskbarWidgetEnabled() bool {
	if a.taskbarOverlay == nil {
		return false
	}
	return a.taskbarOverlay.IsEnabled()
}

func (a *App) SetTaskbarConfig(cfg taskbar.TaskbarOverlayConfig) {
	if a.taskbarOverlay != nil {
		a.taskbarOverlay.SetConfig(cfg)
	}
}

func (a *App) GetTaskbarConfig() taskbar.TaskbarOverlayConfig {
	if a.taskbarOverlay == nil {
		return taskbar.TaskbarOverlayConfig{}
	}
	return a.taskbarOverlay.GetConfig()
}

func (a *App) SetPingOverlayConfig(cfg pingoverlay.Config) {
	if a.pingOverlay != nil {
		a.pingOverlay.SetConfig(cfg)
	}
}

func (a *App) GetPingOverlayConfig() pingoverlay.Config {
	if a.pingOverlay == nil {
		return pingoverlay.Config{}
	}
	return a.pingOverlay.GetConfig()
}

func (a *App) SetAutoStart(enable bool) error {
	return autostart.SetAutoStart(enable)
}

func (a *App) IsAutoStartEnabled() bool {
	return autostart.IsAutoStartEnabled()
}

var emitEventMu sync.Mutex
var globalEmitEvent func(string, interface{})

func (a *App) SetEmitEvent(fn func(string, interface{})) {
	emitEventMu.Lock()
	a.emitEvent = fn
	emitEventMu.Unlock()
}

var _ emitEventHolder = (*App)(nil)

type emitEventHolder interface {
	SetEmitEvent(func(string, interface{}))
}


func (a *App) GetDashboardState() domain.TrafficSnapshot {
	if a.trafficSvc == nil {
		return domain.TrafficSnapshot{Interfaces: map[string]domain.InterfaceStats{}}
	}
	return a.trafficSvc.GetDashboardState()
}

func (a *App) GetInterfaces() ([]domain.NetworkInterface, error) {
	if a.trafficSvc == nil {
		return nil, nil
	}
	return a.trafficSvc.GetInterfaces(context.Background())
}

func (a *App) GetRateHistory() []domain.RateSample {
	if a.trafficSvc == nil {
		return nil
	}
	return a.trafficSvc.GetRateHistory()
}

func (a *App) GetTrafficHistory(startISO, endISO, resolution string) ([]domain.TrafficSample, error) {
	if a.trafficSvc == nil {
		return nil, nil
	}
	start, err := time.Parse(time.RFC3339, startISO)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse(time.RFC3339, endISO)
	if err != nil {
		return nil, err
	}
	return a.trafficSvc.GetTrafficHistory(context.Background(), start, end, resolution)
}

func (a *App) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"platform":       "windows",
		"trafficMonitor": true,
		"processMonitor": true,
		"routerMonitor":  false,
		"sync":           false,
		"dbPath":         a.dbPath,
	}
}

func (a *App) GetProcessTraffic() []connections.ProcessTraffic {
	if a.connSvc == nil {
		return nil
	}
	return a.connSvc.GetProcessTraffic()
}

func (a *App) GetMetrics() map[string]uint64 {
	if a.trafficSvc == nil {
		return nil
	}
	m := a.trafficSvc.GetMetrics()
	return map[string]uint64{
		"samplesRead":    m.SamplesRead,
		"samplesDropped": m.SamplesDropped,
		"dbWrites":       m.DBWrites,
		"dbErrors":       m.DBErrors,
		"restarts":       m.Restarts,
	}
}

func (a *App) CheckDatabase() error {
	if a.storage == nil {
		return nil
	}
	return a.storage.CheckIntegrity(context.Background())
}


func (a *App) Ping(host string, count int) (*diagnostics.PingResult, error) {
	if a.diagSvc == nil {
		return nil, fmt.Errorf("diagnostics service not initialized")
	}
	return a.diagSvc.Ping(host, count)
}

func (a *App) Traceroute(host string, maxHops int) (*diagnostics.TracerouteResult, error) {
	if a.diagSvc == nil {
		return nil, fmt.Errorf("diagnostics service not initialized")
	}
	return a.diagSvc.Traceroute(host, maxHops)
}

func (a *App) DNSLookup(host string) (*diagnostics.DNSResult, error) {
	if a.diagSvc == nil {
		return nil, fmt.Errorf("diagnostics service not initialized")
	}
	return a.diagSvc.DNSLookup(host)
}

func (a *App) TCPConnect(host string, port int, timeoutSec int) (*diagnostics.TCPResult, error) {
	if a.diagSvc == nil {
		return nil, fmt.Errorf("diagnostics service not initialized")
	}
	return a.diagSvc.TCPConnect(host, port, timeoutSec)
}

func (a *App) GetDefaultGateway() (*diagnostics.GatewayInfo, error) {
	if a.diagSvc == nil {
		return nil, fmt.Errorf("diagnostics service not initialized")
	}
	return a.diagSvc.GetDefaultGateway()
}


func (a *App) GetConnections() ([]connections.Connection, error) {
	if a.connSvc == nil {
		return nil, fmt.Errorf("connections service not initialized")
	}
	return a.connSvc.GetConnections()
}


func (a *App) GetActiveInterfaces() []domain.NetworkInterface {
	if a.trafficSvc == nil {
		return nil
	}
	result := a.trafficSvc.GetActiveInterfaces()
	a.log.Info("GetActiveInterfaces", "count", len(result))
	return result
}

func (a *App) GetAllInterfacesDebug() ([]map[string]interface{}, error) {
	if a.trafficSvc == nil {
		return nil, fmt.Errorf("traffic service not initialized")
	}
	all, err := a.trafficSvc.GetInterfaces(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(all))
	for _, iface := range all {
		entry := map[string]interface{}{
			"id":             iface.ID,
			"name":           iface.Name,
			"type":           iface.Type,
			"isUp":           iface.IsUp,
			"isLoopback":     iface.IsLoopback,
			"isVirtual":      iface.IsVirtual,
			"mediaConnected": iface.MediaConnected,
			"isActive":       iface.IsUp && !iface.IsLoopback && !iface.IsVirtual && iface.MediaConnected,
			"gatewayIP":      iface.GatewayIP,
			"ssid":           iface.SSID,
			"ipAddresses":    iface.IPAddresses,
		}
		out = append(out, entry)
	}
	a.log.Info("GetAllInterfacesDebug", "total", len(all), "returning", len(out))
	return out, nil
}

func (a *App) SetSelectedInterfaces(ids []string) error {
	if a.trafficSvc == nil {
		return fmt.Errorf("traffic service not initialized")
	}
	a.trafficSvc.SetSelectedInterfaces(ids)
	return a.saveSelection(ids)
}

func (a *App) GetSelectedInterfaces() []string {
	if a.trafficSvc == nil {
		return nil
	}
	return a.trafficSvc.GetSelectedInterfaces()
}

func (a *App) loadSelection() {
	if a.selectionPath == "" {
		return
	}
	data, err := os.ReadFile(a.selectionPath)
	if err != nil {
		return
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		a.log.Warn("failed to parse selection file", "error", err)
		return
	}
	a.trafficSvc.SetSelectedInterfaces(ids)
	a.log.Info("loaded interface selection", "count", len(ids))
}

func (a *App) saveSelection(ids []string) error {
	if a.selectionPath == "" {
		return nil
	}
	data, err := json.Marshal(ids)
	if err != nil {
		return fmt.Errorf("marshal selection: %w", err)
	}
	return os.WriteFile(a.selectionPath, data, 0644)
}
