package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/netrasad/netrasad/internal/domain"
	"github.com/netrasad/netrasad/internal/ports"
)

type Storage struct {
	db *DB
}

func NewStorage(db *DB) *Storage {
	return &Storage{db: db}
}

func (s *Storage) SaveTrafficSamples(ctx context.Context, samples []domain.TrafficSample) error {
	if len(samples) == 0 {
		return nil
	}
	tx, err := s.db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	upsertIface, err := tx.PrepareContext(ctx,
		`INSERT INTO interfaces (id, name, description, hardware_addr, if_index, if_type, is_up, is_loopback, is_virtual, is_vpn, last_seen)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET last_seen=datetime('now')`)
	if err != nil {
		return fmt.Errorf("prepare interface upsert: %w", err)
	}
	defer upsertIface.Close()

	insertSample, err := tx.PrepareContext(ctx,
		`INSERT INTO traffic_samples (timestamp, interface_id, download_bytes, upload_bytes, category)
		 VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare sample insert: %w", err)
	}
	defer insertSample.Close()

	seenIfaces := make(map[string]bool)
	for _, sm := range samples {
		if !seenIfaces[sm.InterfaceID] {
			if _, err := upsertIface.ExecContext(ctx,
				sm.InterfaceID, sm.InterfaceID, "", "", 0, 0, 1, 0, 0, 0,
				sm.Timestamp.UTC().Format(time.RFC3339)); err != nil {
				return fmt.Errorf("upsert interface: %w", err)
			}
			seenIfaces[sm.InterfaceID] = true
		}
		if _, err := insertSample.ExecContext(ctx,
			sm.Timestamp.UTC().Format(time.RFC3339),
			sm.InterfaceID,
			sm.DownloadBytes,
			sm.UploadBytes,
			int(sm.Category)); err != nil {
			return fmt.Errorf("insert sample: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Storage) QueryTraffic(ctx context.Context, q ports.TrafficQuery) ([]domain.TrafficSample, error) {
	var (
		query string
		args  []interface{}
	)

	table := resolutionTable(q.Resolution)
	query = `SELECT timestamp, interface_id, download_bytes, upload_bytes, category FROM ` + table + ` WHERE timestamp >= ? AND timestamp <= ?`
	args = append(args, q.StartTime.UTC().Format(time.RFC3339), q.EndTime.UTC().Format(time.RFC3339))

	if len(q.InterfaceIDs) > 0 {
		placeholders := make([]string, len(q.InterfaceIDs))
		for i, id := range q.InterfaceIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query += ` AND interface_id IN (` + joinPlaceholders(placeholders) + `)`
	}
	query += ` ORDER BY timestamp ASC`

	rows, err := s.db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query traffic: %w", err)
	}
	defer rows.Close()

	var out []domain.TrafficSample
	for rows.Next() {
		var sm domain.TrafficSample
		var ts string
		var cat int
		if err := rows.Scan(&ts, &sm.InterfaceID, &sm.DownloadBytes, &sm.UploadBytes, &cat); err != nil {
			return nil, err
		}
		sm.Timestamp, _ = time.Parse(time.RFC3339, ts)
		sm.Category = domain.TrafficCategory(cat)
		out = append(out, sm)
	}
	return out, rows.Err()
}

func resolutionTable(res string) string {
	switch res {
	case "hourly":
		return "traffic_hourly"
	case "daily":
		return "traffic_daily"
	case "weekly":
		return "traffic_weekly"
	case "monthly":
		return "traffic_monthly"
	default:
		return "traffic_samples"
	}
}

func joinPlaceholders(p []string) string {
	if len(p) == 0 {
		return ""
	}
	out := p[0]
	for _, s := range p[1:] {
		out += "," + s
	}
	return out
}

func (s *Storage) SaveApplicationUsage(ctx context.Context, usage []domain.ProcessTrafficSample) error {
	if len(usage) == 0 {
		return nil
	}
	tx, err := s.db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO application_usage (timestamp, application_id, download_bytes, upload_bytes)
		 VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, u := range usage {
		if _, err := stmt.ExecContext(ctx,
			u.Timestamp.UTC().Format(time.RFC3339),
			u.ApplicationID, u.DownloadBytes, u.UploadBytes); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Storage) SaveQuotaEvent(ctx context.Context, event domain.QuotaEvent) error {
	_, err := s.db.conn.ExecContext(ctx,
		`INSERT INTO quota_events (quota_id, timestamp, percent, used_bytes, limit_bytes, notified)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		event.QuotaID, event.Timestamp.UTC().Format(time.RFC3339),
		event.Percent, event.UsedBytes, event.LimitBytes, event.Notified)
	return err
}

func (s *Storage) Backup(ctx context.Context, destination string) error {
	_, err := s.db.conn.ExecContext(ctx, `VACUUM INTO ?`, destination)
	return err
}

func (s *Storage) Restore(ctx context.Context, source string) error {
	var count int
	row := s.db.conn.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='schema_migrations'`)
	if err := row.Scan(&count); err != nil {
		return fmt.Errorf("restore: source validation: %w", err)
	}
	if count == 0 {
		return fmt.Errorf("restore: source does not appear to be a NetRasad database")
	}
	return nil
}

func (s *Storage) CheckIntegrity(ctx context.Context) error {
	row := s.db.conn.QueryRowContext(ctx, `PRAGMA integrity_check`)
	var result string
	if err := row.Scan(&result); err != nil {
		return err
	}
	if result != "ok" {
		return fmt.Errorf("integrity check failed: %s", result)
	}
	return nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) DBSize(ctx context.Context) (int64, error) {
	var size sql.NullInt64
	err := s.db.conn.QueryRowContext(ctx,
		`SELECT page_count * page_size FROM pragma_page_count(), pragma_page_size()`).Scan(&size)
	if err != nil {
		return 0, err
	}
	return size.Int64, nil
}
