package traffic

import (
	"testing"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
)

func TestRingBuffer_PushAndAll(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Push(domain.RateSample{Timestamp: time.Unix(1, 0), DownloadBPS: 10})
	rb.Push(domain.RateSample{Timestamp: time.Unix(2, 0), DownloadBPS: 20})
	rb.Push(domain.RateSample{Timestamp: time.Unix(3, 0), DownloadBPS: 30})

	all := rb.All()
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3", len(all))
	}
	if all[0].DownloadBPS != 10 || all[2].DownloadBPS != 30 {
		t.Errorf("order wrong: got %f, %f, %f", all[0].DownloadBPS, all[1].DownloadBPS, all[2].DownloadBPS)
	}
}

func TestRingBuffer_Overwrite(t *testing.T) {
	rb := NewRingBuffer(3)
	for i := 0; i < 5; i++ {
		rb.Push(domain.RateSample{Timestamp: time.Unix(int64(i), 0), DownloadBPS: float64(i) * 10})
	}
	all := rb.All()
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3 (capacity)", len(all))
	}
	// Should contain the last 3: 20, 30, 40
	if all[0].DownloadBPS != 20 || all[1].DownloadBPS != 30 || all[2].DownloadBPS != 40 {
		t.Errorf("expected 20,30,40 got %f,%f,%f", all[0].DownloadBPS, all[1].DownloadBPS, all[2].DownloadBPS)
	}
}

func TestRingBuffer_Last(t *testing.T) {
	rb := NewRingBuffer(3)
	if _, ok := rb.Last(); ok {
		t.Error("empty buffer Last should return false")
	}
	rb.Push(domain.RateSample{DownloadBPS: 42})
	last, ok := rb.Last()
	if !ok || last.DownloadBPS != 42 {
		t.Errorf("Last = %f, ok=%v, want 42, true", last.DownloadBPS, ok)
	}
}

func TestRingBuffer_Clear(t *testing.T) {
	rb := NewRingBuffer(3)
	rb.Push(domain.RateSample{DownloadBPS: 1})
	rb.Clear()
	if len(rb.All()) != 0 {
		t.Error("Clear should empty the buffer")
	}
}

func TestAggregateBuffer_AddAndFlush(t *testing.T) {
	ab := NewAggregateBuffer()
	now := time.Now()

	ab.Add("iface1", domain.CategoryInternet, 100, 50, now)
	ab.Add("iface1", domain.CategoryInternet, 200, 100, now) // accumulate
	ab.Add("iface2", domain.CategoryInternet, 300, 150, now)

	if ab.IsEmpty() {
		t.Fatal("buffer should not be empty after Add")
	}

	samples := ab.Flush()
	if len(samples) != 2 {
		t.Fatalf("Flush returned %d samples, want 2", len(samples))
	}
	if !ab.IsEmpty() {
		t.Error("buffer should be empty after Flush")
	}

	// Find iface1 sample
	var s1 *domain.TrafficSample
	for i := range samples {
		if samples[i].InterfaceID == "iface1" {
			s1 = &samples[i]
		}
	}
	if s1 == nil {
		t.Fatal("iface1 sample not found")
	}
	if s1.DownloadBytes != 300 {
		t.Errorf("iface1 download = %d, want 300 (accumulated)", s1.DownloadBytes)
	}
	if s1.UploadBytes != 150 {
		t.Errorf("iface1 upload = %d, want 150 (accumulated)", s1.UploadBytes)
	}
}
