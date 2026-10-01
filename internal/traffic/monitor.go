package traffic

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
	"github.com/netrasad/netrasad/internal/ports"
)

type MonitorOptions struct {
	SampleInterval time.Duration
	PersistInterval time.Duration
	RingCapacity int
}

func DefaultMonitorOptions() MonitorOptions {
	return MonitorOptions{
		SampleInterval:  1 * time.Second,
		PersistInterval: 5 * time.Second,
		RingCapacity:    300,
	}
}

type Monitor struct {
	source     ports.TrafficSource
	store      ports.DatabaseStorage
	classifier *Classifier
	opts       MonitorOptions
	log        *slog.Logger

	mu      sync.Mutex
	ifaces  map[string]*InterfaceState
	ring    *RingBuffer
	aggBuf  *AggregateBuffer
	running bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	metrics Metrics
}

type Metrics struct {
	mu             sync.Mutex
	SamplesRead    uint64
	SamplesDropped uint64
	DBWrites       uint64
	DBErrors       uint64
	Restarts       uint64
}

func (m *Metrics) Snapshot() Metrics {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Metrics{
		SamplesRead:    m.SamplesRead,
		SamplesDropped: m.SamplesDropped,
		DBWrites:       m.DBWrites,
		DBErrors:       m.DBErrors,
		Restarts:       m.Restarts,
	}
}

func NewMonitor(source ports.TrafficSource, store ports.DatabaseStorage, classifier *Classifier, opts MonitorOptions, log *slog.Logger) *Monitor {
	if log == nil {
		log = slog.Default()
	}
	if opts.SampleInterval <= 0 {
		opts = DefaultMonitorOptions()
	}
	if opts.PersistInterval <= 0 {
		opts.PersistInterval = 5 * time.Second
	}
	if opts.RingCapacity <= 0 {
		opts.RingCapacity = 300
	}
	return &Monitor{
		source:     source,
		store:      store,
		classifier: classifier,
		opts:       opts,
		log:        log,
		ifaces:     make(map[string]*InterfaceState),
		ring:       NewRingBuffer(opts.RingCapacity),
		aggBuf:     NewAggregateBuffer(),
	}
}

func (m *Monitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	if err := m.source.Start(ctx); err != nil {
		m.mu.Unlock()
		return err
	}
	innerCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.running = true
	m.mu.Unlock()

	m.wg.Add(2)
	go m.sampleLoop(innerCtx)
	go m.persistLoop(innerCtx)

	m.log.Info("traffic monitor started",
		"sampleInterval", m.opts.SampleInterval,
		"persistInterval", m.opts.PersistInterval)
	return nil
}

func (m *Monitor) Stop() error {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return nil
	}
	m.cancel()
	m.running = false
	m.mu.Unlock()

	m.wg.Wait()

	if m.store != nil && !m.aggBuf.IsEmpty() {
		m.flushToStore(context.Background())
	}
	return m.source.Stop()
}

func (m *Monitor) sampleLoop(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(m.opts.SampleInterval)
	defer ticker.Stop()

	m.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.poll(ctx)
		}
	}
}

func (m *Monitor) poll(ctx context.Context) {
	counters, err := m.source.ReadAllStats(ctx)
	if err != nil {
		m.log.Warn("traffic source read failed", "error", err)
		return
	}

	m.refreshInterfaces(ctx)

	now := time.Now()
	var totalDownRate, totalUpRate float64

	m.mu.Lock()
	for _, c := range counters {
		st, ok := m.ifaces[c.InterfaceID]
		if !ok {
			if iface, known := m.lookupInterface(c.InterfaceID); known {
				st = &InterfaceState{Interface: iface}
				m.ifaces[c.InterfaceID] = st
			} else {
				continue
			}
		}

		if !m.classifier.ShouldCountInterface(st.Interface.ID, st.Interface.IsLoopback) {
			continue
		}

		prevDown := st.CumDown
		prevUp := st.CumUp
		st.Update(c)
		downDelta := st.CumDown - prevDown
		upDelta := st.CumUp - prevUp

		category := m.classifier.ClassifyInterface(st.Interface)
		if category == domain.CategoryIgnored {
			continue
		}

		m.aggBuf.Add(c.InterfaceID, category, downDelta, upDelta, now)
		totalDownRate += st.DownRate
		totalUpRate += st.UpRate

		m.metrics.mu.Lock()
		m.metrics.SamplesRead++
		m.metrics.mu.Unlock()
	}
	m.mu.Unlock()

	m.ring.Push(domain.RateSample{
		Timestamp:   now,
		DownloadBPS: totalDownRate,
		UploadBPS:   totalUpRate,
	})
}

func (m *Monitor) refreshInterfaces(ctx context.Context) {
	ifaces, err := m.source.Interfaces(ctx)
	if err != nil {
		m.log.Debug("interface refresh failed", "error", err)
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, iface := range ifaces {
		if _, ok := m.ifaces[iface.ID]; !ok {
			m.ifaces[iface.ID] = &InterfaceState{Interface: iface}
		} else {
			m.ifaces[iface.ID].Interface = iface
		}
	}
}

func (m *Monitor) lookupInterface(id string) (domain.NetworkInterface, bool) {
	if st, ok := m.ifaces[id]; ok {
		return st.Interface, true
	}
	return domain.NetworkInterface{}, false
}

func (m *Monitor) persistLoop(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(m.opts.PersistInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.flushToStore(ctx)
		}
	}
}

func (m *Monitor) flushToStore(ctx context.Context) {
	if m.store == nil || m.aggBuf.IsEmpty() {
		return
	}
	samples := m.aggBuf.Flush()
	if len(samples) == 0 {
		return
	}
	if err := m.store.SaveTrafficSamples(ctx, samples); err != nil {
		m.log.Error("database write failed", "error", err, "samples", len(samples))
		m.metrics.mu.Lock()
		m.metrics.DBErrors++
		m.metrics.mu.Unlock()
		return
	}
	m.metrics.mu.Lock()
	m.metrics.DBWrites++
	m.metrics.mu.Unlock()
	m.log.Debug("persisted traffic samples", "count", len(samples))
}

func (m *Monitor) Snapshot() domain.TrafficSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	ifaces := make(map[string]domain.InterfaceStats, len(m.ifaces))
	var totalDown, totalUp uint64
	var totalDownRate, totalUpRate float64
	for id, st := range m.ifaces {
		if !IsActiveInterface(st.Interface) {
			continue
		}
		if !m.classifier.ShouldCountInterface(st.Interface.ID, st.Interface.IsLoopback) {
			continue
		}
		ifaces[id] = st.ToStats()
		totalDown += st.CumDown
		totalUp += st.CumUp
		totalDownRate += st.DownRate
		totalUpRate += st.UpRate
	}
	return domain.TrafficSnapshot{
		Timestamp:     time.Now(),
		TotalDownload: totalDown,
		TotalUpload:   totalUp,
		DownloadRate:  totalDownRate,
		UploadRate:    totalUpRate,
		Interfaces:    ifaces,
	}
}

func (m *Monitor) RateHistory() []domain.RateSample {
	return m.ring.All()
}

func (m *Monitor) Metrics() Metrics {
	return m.metrics.Snapshot()
}

func (m *Monitor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func IsActiveInterface(iface domain.NetworkInterface) bool {
	if iface.IsLoopback || iface.IsVirtual {
		return false
	}
	return iface.IsUp || iface.MediaConnected
}

func IsMonitorableInterface(iface domain.NetworkInterface) bool {
	if iface.IsLoopback || iface.IsVirtual {
		return false
	}
	return iface.Type == 6 || iface.Type == 71 || iface.IsUp || iface.MediaConnected
}

func (m *Monitor) ActiveInterfaces() []domain.NetworkInterface {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.NetworkInterface, 0, len(m.ifaces))
	for _, st := range m.ifaces {
		if IsMonitorableInterface(st.Interface) {
			out = append(out, st.Interface)
		}
	}
	return out
}

func (m *Monitor) SetSelectedInterfaces(ids []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.classifier.SetOnlyInterfaces(ids)
	m.log.Info("selected interfaces updated", "count", len(ids))
}

func (m *Monitor) GetSelectedInterfaces() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.classifier.GetOnlyInterfaces()
}
