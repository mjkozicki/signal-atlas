package scanner

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"signal-atlas/src/internal/snapshots"

	_ "modernc.org/sqlite"
)

type Store struct{ DB *sql.DB }

func OpenStore(path string) (*Store, error) {
	if path != ":memory:" {
		if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			return nil, e
		}
		f, e := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
		if e != nil {
			return nil, e
		}
		f.Close()
		if e = os.Chmod(path, 0600); e != nil {
			return nil, e
		}
	}
	db, e := sql.Open("sqlite", path)
	if e != nil {
		return nil, e
	}
	db.SetMaxOpenConns(1)
	_, e = db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS scans (id TEXT PRIMARY KEY, created_at TEXT NOT NULL, body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS settings (key TEXT PRIMARY KEY, body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS audit_log (id INTEGER PRIMARY KEY, created_at TEXT NOT NULL, event TEXT NOT NULL, body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS audit_leases (network TEXT PRIMARY KEY, started_at INTEGER NOT NULL);` + snapshots.Schema)
	if e != nil {
		db.Close()
		return nil, e
	}
	return &Store{db}, nil
}
func (s *Store) Close() error { return s.DB.Close() }
func (s *Store) Save(scan *Scan) error {
	b := make([]byte, 8)
	if _, e := rand.Read(b); e != nil {
		return e
	}
	scan.Hidden = false
	scan.ID = "scan-" + hex.EncodeToString(b)
	if scan.CreatedAt.IsZero() {
		scan.CreatedAt = time.Now().UTC()
	}
	data, e := json.Marshal(scan)
	if e != nil {
		return e
	}
	_, e = s.DB.Exec("INSERT INTO scans VALUES (?, ?, ?)", scan.ID, scan.CreatedAt.Format(time.RFC3339Nano), string(data))
	return e
}
func (s *Store) Get(id string) (*Scan, error) {
	var body string
	var e error
	var hidden bool
	if id == "" || id == "latest" {
		e = s.DB.QueryRow("SELECT body, EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=scans.id) FROM scans WHERE NOT EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=scans.id) ORDER BY rowid DESC LIMIT 1").Scan(&body, &hidden)
	} else {
		e = s.DB.QueryRow("SELECT body, EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=scans.id) FROM scans WHERE id=?", id).Scan(&body, &hidden)
	}
	if e != nil {
		return nil, e
	}
	var scan Scan
	e = json.Unmarshal([]byte(body), &scan)
	scan.Hidden = hidden
	return &scan, e
}
func (s *Store) List(includeHidden ...bool) ([]Scan, error) {
	include := len(includeHidden) > 0 && includeHidden[0]
	rows, e := s.DB.Query("SELECT body, EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=scans.id) FROM scans WHERE ? OR NOT EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=scans.id) ORDER BY rowid DESC LIMIT 100", include)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Scan{}
	for rows.Next() {
		var b string
		var hidden bool
		if e = rows.Scan(&b, &hidden); e != nil {
			return nil, e
		}
		var scan Scan
		if e = json.Unmarshal([]byte(b), &scan); e != nil {
			return nil, e
		}
		scan.Hidden = hidden
		out = append(out, scan)
	}
	return out, rows.Err()
}
func (s *Store) Inventory() (Inventory, error) {
	var b string
	e := s.DB.QueryRow("SELECT body FROM settings WHERE key='inventory'").Scan(&b)
	if errors.Is(e, sql.ErrNoRows) {
		return Inventory{Authorized: []AuthorizedNetwork{}}, nil
	}
	if e != nil {
		return Inventory{}, e
	}
	var inv Inventory
	e = json.Unmarshal([]byte(b), &inv)
	return inv, e
}
func (s *Store) SetInventory(inv Inventory) error {
	if e := inv.Validate(); e != nil {
		return e
	}
	b, e := json.Marshal(inv)
	if e != nil {
		return e
	}
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.Exec("INSERT INTO settings VALUES ('inventory',?) ON CONFLICT(key) DO UPDATE SET body=excluded.body", string(b)); e != nil {
		return e
	}
	if _, e = tx.Exec("INSERT INTO audit_log(created_at,event,body) VALUES (?,?,?)", time.Now().UTC().Format(time.RFC3339Nano), "inventory.updated", string(b)); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Log(event string, value any) error {
	b, e := json.Marshal(value)
	if e != nil {
		return e
	}
	_, e = s.DB.Exec("INSERT INTO audit_log(created_at,event,body) VALUES (?,?,?)", time.Now().UTC().Format(time.RFC3339Nano), event, string(b))
	return e
}
func (s *Store) AcquireAudit(network string) error {
	result, e := s.DB.Exec(`INSERT INTO audit_leases VALUES (?,?) ON CONFLICT(network) DO UPDATE SET started_at=excluded.started_at WHERE audit_leases.started_at < ?`, network, time.Now().Unix(), time.Now().Add(-5*time.Minute).Unix())
	if e != nil {
		return e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if n == 0 {
		return fmt.Errorf("active audit limited to once per network every five minutes")
	}
	return nil
}

func (s *Store) SetHidden(id string, hidden bool) error {
	return snapshots.Mutate(s.DB, "scans", id, &hidden)
}
func (s *Store) Delete(id string) error { return snapshots.Mutate(s.DB, "scans", id, nil) }
