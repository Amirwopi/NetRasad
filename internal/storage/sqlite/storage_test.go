package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
	"github.com/netrasad/netrasad/internal/ports"
)

func testDB(t *testing.T) *Storage {
	t.Helper()
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewStorage(db)
}

func TestMigrations_Applied(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(ctx, path, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	version, err := db.MigrationVersion(ctx)
	if err != nil {
		t.Fatalf("MigrationVersion: %v", err)
	}
	if version != 1 {
		t.Errorf("migration version = %d, want 1", version)
	}
}

func TestSaveAndQueryTrafficSamples(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	samples := []domain.TrafficSample{
		{Timestamp: now, InterfaceID: "luid-aaa", DownloadBytes: 1000, UploadBytes: 500, Category: domain.CategoryInternet},
		{Timestamp: now.Add(5 * time.Second), InterfaceID: "luid-aaa", DownloadBytes: 2000, UploadBytes: 1000, Category: domain.CategoryInternet},
		{Timestamp: now.Add(10 * time.Second), InterfaceID: "luid-bbb", DownloadBytes: 3000, UploadBytes: 1500, Category: domain.CategoryInternet},
	}

	if err := s.SaveTrafficSamples(ctx, samples); err != nil {
		t.Fatalf("SaveTrafficSamples: %v", err)
	}

	got, err := s.QueryTraffic(ctx, ports.TrafficQuery{
		StartTime:  now.Add(-time.Hour),
		EndTime:    now.Add(time.Hour),
		Resolution: "sample",
	})
	if err != nil {
		t.Fatalf("QueryTraffic: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("got %d samples, want 3", len(got))
	}
}

func TestQueryTraffic_FilterByInterface(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()
	now := time.Now().UTC()

	samples := []domain.TrafficSample{
		{Timestamp: now, InterfaceID: "luid-aaa", DownloadBytes: 100, UploadBytes: 50},
		{Timestamp: now, InterfaceID: "luid-bbb", DownloadBytes: 200, UploadBytes: 100},
	}
	if err := s.SaveTrafficSamples(ctx, samples); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := s.QueryTraffic(ctx, ports.TrafficQuery{
		StartTime:    now.Add(-time.Hour),
		EndTime:      now.Add(time.Hour),
		InterfaceIDs: []string{"luid-aaa"},
		Resolution:   "sample",
	})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("got %d samples, want 1 (filtered)", len(got))
	}
	if got[0].InterfaceID != "luid-aaa" {
		t.Errorf("interface = %s, want luid-aaa", got[0].InterfaceID)
	}
}

func TestCheckIntegrity(t *testing.T) {
	s := testDB(t)
	if err := s.CheckIntegrity(context.Background()); err != nil {
		t.Errorf("CheckIntegrity: %v", err)
	}
}

func TestDBSize(t *testing.T) {
	s := testDB(t)
	size, err := s.DBSize(context.Background())
	if err != nil {
		t.Fatalf("DBSize: %v", err)
	}
	if size <= 0 {
		t.Errorf("DBSize = %d, want > 0", size)
	}
}

func TestSaveQuotaEvent(t *testing.T) {
	s := testDB(t)
	ctx := context.Background()

	// First create a quota so the FK is satisfied.
	_, err := s.db.conn.ExecContext(ctx,
		`INSERT INTO quotas (id, name, period, direction, limit_bytes, enabled, thresholds) VALUES (1, 'test', 'monthly', 'combined', 1000000, 1, '[0.5,0.9]')`)
	if err != nil {
		t.Fatalf("insert quota: %v", err)
	}

	event := domain.QuotaEvent{
		QuotaID:    1,
		Timestamp:  time.Now(),
		Percent:    90.0,
		UsedBytes:  900000,
		LimitBytes: 1000000,
		Notified:   true,
	}
	if err := s.SaveQuotaEvent(ctx, event); err != nil {
		t.Fatalf("SaveQuotaEvent: %v", err)
	}
}
