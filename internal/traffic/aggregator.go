package traffic

import (
	"sync"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
)

type RingBuffer struct {
	mu      sync.Mutex
	samples []domain.RateSample
	head    int
	count   int
	cap     int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity < 1 {
		capacity = 1
	}
	return &RingBuffer{
		samples: make([]domain.RateSample, capacity),
		cap:     capacity,
	}
}

func (r *RingBuffer) Push(s domain.RateSample) {
	r.mu.Lock()
	r.samples[r.head] = s
	r.head = (r.head + 1) % r.cap
	if r.count < r.cap {
		r.count++
	}
	r.mu.Unlock()
}

func (r *RingBuffer) All() []domain.RateSample {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.RateSample, r.count)
	start := (r.head - r.count + r.cap) % r.cap
	for i := 0; i < r.count; i++ {
		out[i] = r.samples[(start+i)%r.cap]
	}
	return out
}

func (r *RingBuffer) Last() (domain.RateSample, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.count == 0 {
		return domain.RateSample{}, false
	}
	idx := (r.head - 1 + r.cap) % r.cap
	return r.samples[idx], true
}

func (r *RingBuffer) Clear() {
	r.mu.Lock()
	r.head = 0
	r.count = 0
	r.mu.Unlock()
}

type AggregateBuffer struct {
	mu      sync.Mutex
	buckets map[string]*aggBucket
}

type aggBucket struct {
	interfaceID   string
	category      domain.TrafficCategory
	downloadBytes uint64
	uploadBytes   uint64
	startTime     time.Time
}

func NewAggregateBuffer() *AggregateBuffer {
	return &AggregateBuffer{buckets: make(map[string]*aggBucket)}
}

func (a *AggregateBuffer) Add(interfaceID string, category domain.TrafficCategory, downBytes, upBytes uint64, ts time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	b, ok := a.buckets[interfaceID]
	if !ok {
		b = &aggBucket{interfaceID: interfaceID, category: category, startTime: ts}
		a.buckets[interfaceID] = b
	}
	b.downloadBytes += downBytes
	b.uploadBytes += upBytes
}

func (a *AggregateBuffer) Flush() []domain.TrafficSample {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]domain.TrafficSample, 0, len(a.buckets))
	for _, b := range a.buckets {
		out = append(out, domain.TrafficSample{
			Timestamp:     b.startTime,
			InterfaceID:   b.interfaceID,
			DownloadBytes: b.downloadBytes,
			UploadBytes:   b.uploadBytes,
			Category:      b.category,
		})
	}
	a.buckets = make(map[string]*aggBucket)
	return out
}

func (a *AggregateBuffer) IsEmpty() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.buckets) == 0
}
