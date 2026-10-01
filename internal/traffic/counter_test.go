package traffic

import (
	"testing"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
)

func TestCounterDelta_NormalIncrement(t *testing.T) {
	got := CounterDelta(100, 250)
	if got != 150 {
		t.Errorf("CounterDelta(100, 250) = %d, want 150", got)
	}
}

func TestCounterDelta_ZeroPrevious(t *testing.T) {
	got := CounterDelta(0, 500)
	if got != 500 {
		t.Errorf("CounterDelta(0, 500) = %d, want 500", got)
	}
}

func TestCounterDelta_Reset(t *testing.T) {
	// Counter reset to 0 after reaching 1000.
	got := CounterDelta(1000, 100)
	if got != 100 {
		t.Errorf("CounterDelta(1000, 100) = %d, want 100 (treat as reset)", got)
	}
}

func TestCounterDelta_FullWraparound(t *testing.T) {
	// 64-bit wraparound: previous near max, current small.
	prev := uint64(18446744073709551610) // near uint64 max
	curr := uint64(15)
	got := CounterDelta(prev, curr)
	if got != 15 {
		t.Errorf("CounterDelta near wraparound = %d, want 15", got)
	}
}

func TestCounterDelta_NoChange(t *testing.T) {
	got := CounterDelta(500, 500)
	if got != 0 {
		t.Errorf("CounterDelta(500, 500) = %d, want 0", got)
	}
}

func TestComputeRate_Normal(t *testing.T) {
	rate := ComputeRate(1000, 2*time.Second)
	if rate != 500 {
		t.Errorf("ComputeRate(1000, 2s) = %f, want 500", rate)
	}
}

func TestComputeRate_ZeroElapsed(t *testing.T) {
	rate := ComputeRate(1000, 0)
	if rate != 0 {
		t.Errorf("ComputeRate(1000, 0) = %f, want 0", rate)
	}
}

func TestComputeRate_NegativeElapsed(t *testing.T) {
	rate := ComputeRate(1000, -1*time.Second)
	if rate != 0 {
		t.Errorf("ComputeRate(1000, -1s) = %f, want 0 (clock adjustment)", rate)
	}
}

func TestInterfaceState_FirstReading(t *testing.T) {
	s := &InterfaceState{Started: false}
	now := time.Now()
	s.Update(testCounters(now, 100, 200))
	if !s.Started {
		t.Fatal("expected Started=true after first reading")
	}
	if s.CumDown != 0 || s.CumUp != 0 {
		t.Errorf("first reading should not produce cumulative delta, got down=%d up=%d", s.CumDown, s.CumUp)
	}
}

func TestInterfaceState_SecondReading(t *testing.T) {
	s := &InterfaceState{Started: false}
	t0 := time.Now()
	s.Update(testCounters(t0, 100, 200))
	t1 := t0.Add(2 * time.Second)
	s.Update(testCounters(t1, 600, 800))

	if s.CumDown != 500 {
		t.Errorf("cumulative download = %d, want 500", s.CumDown)
	}
	if s.CumUp != 600 {
		t.Errorf("cumulative upload = %d, want 600", s.CumUp)
	}
	if s.DownRate != 250 {
		t.Errorf("download rate = %f, want 250", s.DownRate)
	}
	if s.UpRate != 300 {
		t.Errorf("upload rate = %f, want 300", s.UpRate)
	}
}

func TestInterfaceState_CounterReset(t *testing.T) {
	s := &InterfaceState{Started: false}
	t0 := time.Now()
	s.Update(testCounters(t0, 1000, 2000))
	t1 := t0.Add(1 * time.Second)
	// Counter reset to 0, then counted 50.
	s.Update(testCounters(t1, 50, 75))

	if s.CumDown != 50 {
		t.Errorf("after reset, cumulative download = %d, want 50", s.CumDown)
	}
	if s.CumUp != 75 {
		t.Errorf("after reset, cumulative upload = %d, want 75", s.CumUp)
	}
}

func TestInterfaceState_PeakRate(t *testing.T) {
	s := &InterfaceState{Started: false}
	t0 := time.Now()
	s.Update(testCounters(t0, 0, 0))
	t1 := t0.Add(1 * time.Second)
	s.Update(testCounters(t1, 100, 50))
	t2 := t1.Add(1 * time.Second)
	s.Update(testCounters(t2, 150, 300)) // down=50, up=250

	if s.PeakDownRate != 100 {
		t.Errorf("peak down rate = %f, want 100", s.PeakDownRate)
	}
	if s.PeakUpRate != 250 {
		t.Errorf("peak up rate = %f, want 250", s.PeakUpRate)
	}
}

func testCounters(ts time.Time, down, up uint64) domain.TrafficCounters {
	return domain.TrafficCounters{
		Timestamp:     ts,
		InterfaceID:   "test0",
		DownloadBytes: down,
		UploadBytes:   up,
	}
}
