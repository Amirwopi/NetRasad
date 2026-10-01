package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		name:    "initial_schema",
		sql: strings.Join([]string{
			`CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				name    TEXT NOT NULL,
				applied_at TEXT NOT NULL DEFAULT (datetime('now'))
			);`,

			`CREATE TABLE IF NOT EXISTS interfaces (
				id           TEXT PRIMARY KEY,
				name         TEXT NOT NULL,
				description  TEXT NOT NULL DEFAULT '',
				hardware_addr TEXT NOT NULL DEFAULT '',
				if_index     INTEGER NOT NULL DEFAULT 0,
				if_type      INTEGER NOT NULL DEFAULT 0,
				is_up        INTEGER NOT NULL DEFAULT 0,
				is_loopback  INTEGER NOT NULL DEFAULT 0,
				is_virtual   INTEGER NOT NULL DEFAULT 0,
				is_vpn       INTEGER NOT NULL DEFAULT 0,
				first_seen   TEXT NOT NULL DEFAULT (datetime('now')),
				last_seen    TEXT NOT NULL DEFAULT (datetime('now'))
			);`,

			`CREATE TABLE IF NOT EXISTS traffic_samples (
				id            INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp     TEXT NOT NULL,
				interface_id  TEXT NOT NULL,
				download_bytes INTEGER NOT NULL,
				upload_bytes   INTEGER NOT NULL,
				category      INTEGER NOT NULL DEFAULT 0,
				FOREIGN KEY (interface_id) REFERENCES interfaces(id)
			);`,
			`CREATE INDEX IF NOT EXISTS idx_traffic_samples_ts ON traffic_samples(timestamp);`,
			`CREATE INDEX IF NOT EXISTS idx_traffic_samples_iface ON traffic_samples(interface_id);`,

			`CREATE TABLE IF NOT EXISTS traffic_hourly (
				timestamp     TEXT NOT NULL,
				interface_id  TEXT NOT NULL,
				download_bytes INTEGER NOT NULL,
				upload_bytes   INTEGER NOT NULL,
				category      INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY (timestamp, interface_id, category)
			);`,
			`CREATE TABLE IF NOT EXISTS traffic_daily (
				timestamp     TEXT NOT NULL,
				interface_id  TEXT NOT NULL,
				download_bytes INTEGER NOT NULL,
				upload_bytes   INTEGER NOT NULL,
				category      INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY (timestamp, interface_id, category)
			);`,
			`CREATE TABLE IF NOT EXISTS traffic_weekly (
				timestamp     TEXT NOT NULL,
				interface_id  TEXT NOT NULL,
				download_bytes INTEGER NOT NULL,
				upload_bytes   INTEGER NOT NULL,
				category      INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY (timestamp, interface_id, category)
			);`,
			`CREATE TABLE IF NOT EXISTS traffic_monthly (
				timestamp     TEXT NOT NULL,
				interface_id  TEXT NOT NULL,
				download_bytes INTEGER NOT NULL,
				upload_bytes   INTEGER NOT NULL,
				category      INTEGER NOT NULL DEFAULT 0,
				PRIMARY KEY (timestamp, interface_id, category)
			);`,

			`CREATE TABLE IF NOT EXISTS applications (
				id             TEXT PRIMARY KEY,
				name           TEXT NOT NULL,
				executable_path TEXT NOT NULL DEFAULT '',
				publisher      TEXT NOT NULL DEFAULT '',
				process_type   TEXT NOT NULL DEFAULT '',
				first_seen     TEXT NOT NULL DEFAULT (datetime('now'))
			);`,

			`CREATE TABLE IF NOT EXISTS application_usage (
				id             INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp      TEXT NOT NULL,
				application_id TEXT NOT NULL,
				download_bytes INTEGER NOT NULL,
				upload_bytes   INTEGER NOT NULL,
				FOREIGN KEY (application_id) REFERENCES applications(id)
			);`,
			`CREATE INDEX IF NOT EXISTS idx_app_usage_ts ON application_usage(timestamp);`,
			`CREATE INDEX IF NOT EXISTS idx_app_usage_app ON application_usage(application_id);`,

			`CREATE TABLE IF NOT EXISTS connections (
				id             INTEGER PRIMARY KEY AUTOINCREMENT,
				pid            INTEGER NOT NULL,
				application    TEXT NOT NULL DEFAULT '',
				executable     TEXT NOT NULL DEFAULT '',
				protocol       INTEGER NOT NULL,
				local_address  TEXT NOT NULL,
				local_port     INTEGER NOT NULL,
				remote_address TEXT NOT NULL,
				remote_port    INTEGER NOT NULL,
				state          TEXT NOT NULL DEFAULT 'UNKNOWN',
				total_uploaded   INTEGER NOT NULL DEFAULT 0,
				total_downloaded INTEGER NOT NULL DEFAULT 0,
				first_seen     TEXT NOT NULL,
				last_seen      TEXT NOT NULL
			);`,

			`CREATE TABLE IF NOT EXISTS connection_quality (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp  TEXT NOT NULL,
				target     TEXT NOT NULL,
				latency_ms REAL NOT NULL DEFAULT 0,
				success    INTEGER NOT NULL DEFAULT 0,
				error      TEXT NOT NULL DEFAULT ''
			);`,
			`CREATE INDEX IF NOT EXISTS idx_cq_ts ON connection_quality(timestamp);`,
			`CREATE INDEX IF NOT EXISTS idx_cq_target ON connection_quality(target);`,

			`CREATE TABLE IF NOT EXISTS speed_tests (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp   TEXT NOT NULL,
				provider    TEXT NOT NULL DEFAULT '',
				server      TEXT NOT NULL DEFAULT '',
				download_bps REAL NOT NULL DEFAULT 0,
				upload_bps   REAL NOT NULL DEFAULT 0,
				latency_ms   REAL NOT NULL DEFAULT 0,
				jitter_ms    REAL NOT NULL DEFAULT 0,
				duration_sec REAL NOT NULL DEFAULT 0
			);`,
			`CREATE INDEX IF NOT EXISTS idx_speed_tests_ts ON speed_tests(timestamp);`,

			`CREATE TABLE IF NOT EXISTS quotas (
				id          INTEGER PRIMARY KEY AUTOINCREMENT,
				name        TEXT NOT NULL,
				period      TEXT NOT NULL,
				direction   TEXT NOT NULL,
				limit_bytes INTEGER NOT NULL,
				enabled     INTEGER NOT NULL DEFAULT 1,
				thresholds  TEXT NOT NULL DEFAULT '[]'
			);`,

			`CREATE TABLE IF NOT EXISTS quota_events (
				id         INTEGER PRIMARY KEY AUTOINCREMENT,
				quota_id   INTEGER NOT NULL,
				timestamp  TEXT NOT NULL,
				percent    REAL NOT NULL,
				used_bytes INTEGER NOT NULL,
				limit_bytes INTEGER NOT NULL,
				notified   INTEGER NOT NULL DEFAULT 0,
				FOREIGN KEY (quota_id) REFERENCES quotas(id)
			);`,
			`CREATE INDEX IF NOT EXISTS idx_quota_events_ts ON quota_events(timestamp);`,

			`CREATE TABLE IF NOT EXISTS notifications (
				id        INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp TEXT NOT NULL,
				title     TEXT NOT NULL,
				message   TEXT NOT NULL,
				severity  INTEGER NOT NULL DEFAULT 0,
				category  TEXT NOT NULL DEFAULT ''
			);`,
			`CREATE INDEX IF NOT EXISTS idx_notifications_ts ON notifications(timestamp);`,

			`CREATE TABLE IF NOT EXISTS router_devices (
				id       INTEGER PRIMARY KEY AUTOINCREMENT,
				name     TEXT NOT NULL,
				host     TEXT NOT NULL,
				port     INTEGER NOT NULL DEFAULT 161,
				protocol TEXT NOT NULL DEFAULT 'snmp',
				enabled  INTEGER NOT NULL DEFAULT 1
			);`,
			`CREATE TABLE IF NOT EXISTS router_interfaces (
				id        INTEGER PRIMARY KEY AUTOINCREMENT,
				device_id INTEGER NOT NULL,
				if_index  INTEGER NOT NULL,
				name      TEXT NOT NULL DEFAULT '',
				is_wan    INTEGER NOT NULL DEFAULT 0,
				monitored INTEGER NOT NULL DEFAULT 0,
				FOREIGN KEY (device_id) REFERENCES router_devices(id)
			);`,
			`CREATE TABLE IF NOT EXISTS router_samples (
				id             INTEGER PRIMARY KEY AUTOINCREMENT,
				timestamp      TEXT NOT NULL,
				device_id      INTEGER NOT NULL,
				interface_id   INTEGER NOT NULL,
				download_bytes INTEGER NOT NULL,
				upload_bytes   INTEGER NOT NULL,
				FOREIGN KEY (device_id) REFERENCES router_devices(id)
			);`,
			`CREATE INDEX IF NOT EXISTS idx_router_samples_ts ON router_samples(timestamp);`,

			`CREATE TABLE IF NOT EXISTS sync_devices (
				id        TEXT PRIMARY KEY,
				name      TEXT NOT NULL,
				address   TEXT NOT NULL DEFAULT '',
				paired    INTEGER NOT NULL DEFAULT 0,
				last_seen TEXT NOT NULL DEFAULT ''
			);`,
			`CREATE TABLE IF NOT EXISTS sync_events (
				event_id       TEXT PRIMARY KEY,
				device_id      TEXT NOT NULL,
				timestamp      TEXT NOT NULL,
				counter_type   TEXT NOT NULL,
				payload        BLOB NOT NULL,
				schema_version INTEGER NOT NULL DEFAULT 1
			);`,
			`CREATE INDEX IF NOT EXISTS idx_sync_events_device ON sync_events(device_id);`,

			`CREATE TABLE IF NOT EXISTS settings (
				key   TEXT PRIMARY KEY,
				value TEXT NOT NULL
			);`,
		}, "\n"),
	},
}

func (db *DB) migrate(ctx context.Context) error {
	if _, err := db.conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name    TEXT NOT NULL,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	);`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	applied, err := db.appliedMigrations(ctx)
	if err != nil {
		return err
	}

	sorted := make([]migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].version < sorted[j].version })

	for _, m := range sorted {
		if applied[m.version] {
			continue
		}
		if err := db.applyMigration(ctx, m); err != nil {
			return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
		}
		db.log.Info("migration applied", "version", m.version, "name", m.name)
	}
	return nil
}

func (db *DB) applyMigration(ctx context.Context, m migration) error {
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("exec migration sql: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`,
		m.version, m.name); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	return tx.Commit()
}

func (db *DB) appliedMigrations(ctx context.Context) (map[int]bool, error) {
	rows, err := db.conn.QueryContext(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("query migrations: %w", err)
	}
	defer rows.Close()
	applied := make(map[int]bool)
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

func (db *DB) MigrationVersion(ctx context.Context) (int, error) {
	var v sql.NullInt64
	err := db.conn.QueryRowContext(ctx,
		`SELECT MAX(version) FROM schema_migrations`).Scan(&v)
	if err != nil {
		return 0, err
	}
	return int(v.Int64), nil
}
