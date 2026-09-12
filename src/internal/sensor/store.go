package sensor

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"os"
	"path/filepath"
	"time"
	"signal-atlas/src/internal/snapshots"
)

var ErrBusy = errors.New("a scan is already running for this service")

type Store struct {
	DB       *sql.DB
	Protocol string
}

func OpenStore(path, protocol string) (*Store, error) {
	if protocol != "bluetooth" && protocol != "nfc" {
		return nil, fmt.Errorf("unsupported protocol")
	}
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
	fail := func(e error) (*Store, error) { db.Close(); return nil, e }
	var wifi int
	if e = db.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='scans'").Scan(&wifi); e != nil {
		return fail(e)
	}
	if wifi > 0 {
		return fail(fmt.Errorf("use a separate %s database, not a Wi-Fi database", protocol))
	}
	_, e = db.Exec(`PRAGMA busy_timeout=5000; PRAGMA journal_mode=WAL;
 CREATE TABLE IF NOT EXISTS sensor_meta (key TEXT PRIMARY KEY,value TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS sensor_scans (id TEXT PRIMARY KEY,body TEXT NOT NULL);
 CREATE TABLE IF NOT EXISTS sensor_lease (key TEXT PRIMARY KEY,scan_id TEXT NOT NULL,expires INTEGER NOT NULL);` + snapshots.Schema)
	if e != nil {
		return fail(e)
	}
	if _, e = db.Exec("INSERT INTO sensor_meta VALUES ('protocol',?) ON CONFLICT DO NOTHING", protocol); e != nil {
		return fail(e)
	}
	var stored string
	if e = db.QueryRow("SELECT value FROM sensor_meta WHERE key='protocol'").Scan(&stored); e != nil {
		return fail(e)
	}
	if stored != protocol {
		return fail(fmt.Errorf("database belongs to %s, not %s", stored, protocol))
	}
	return &Store{db, protocol}, nil
}
func (s *Store) Close() error { return s.DB.Close() }
func newID() string {
	b := make([]byte, 12)
	if _, e := rand.Read(b); e != nil {
		panic(e)
	}
	return hex.EncodeToString(b)
}
func (s *Store) RecoverExpired() error {
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	rows, e := tx.Query("SELECT s.body FROM sensor_scans s JOIN sensor_lease l ON l.scan_id=s.id WHERE l.expires<?", time.Now().Unix())
	if e != nil {
		return e
	}
	expired := []Scan{}
	for rows.Next() {
		var b string
		if e = rows.Scan(&b); e != nil {
			rows.Close()
			return e
		}
		var x Scan
		if e = json.Unmarshal([]byte(b), &x); e != nil {
			rows.Close()
			return e
		}
		expired = append(expired, x)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, x := range expired {
		now := time.Now().UTC()
		x.State = "failed"
		x.Error = "Scan lease expired; the provider or service was interrupted."
		x.EndedAt = &now
		b, _ := json.Marshal(x)
		if _, e = tx.Exec("UPDATE sensor_scans SET body=? WHERE id=?", string(b), x.ID); e != nil {
			return e
		}
	}
	if _, e = tx.Exec("DELETE FROM sensor_lease WHERE expires<?", time.Now().Unix()); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Begin(o Options) (*Scan, error) {
	if e := s.RecoverExpired(); e != nil {
		return nil, e
	}
	x := &Scan{ID: newID(), Protocol: s.Protocol, Mode: o.Mode, Source: o.Mode, State: "running", CreatedAt: time.Now().UTC(), Duration: o.Duration, Devices: []Device{}, Warnings: []string{}}
	tx, e := s.DB.Begin()
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	result, e := tx.Exec("INSERT INTO sensor_lease VALUES ('active',?,?) ON CONFLICT DO NOTHING", x.ID, time.Now().Add(time.Duration(o.Duration+30)*time.Second).Unix())
	if e != nil {
		return nil, e
	}
	n, e := result.RowsAffected()
	if e != nil {
		return nil, e
	}
	if n == 0 {
		return nil, ErrBusy
	}
	b, _ := json.Marshal(x)
	if _, e = tx.Exec("INSERT INTO sensor_scans VALUES (?,?)", x.ID, string(b)); e != nil {
		return nil, e
	}
	if e = tx.Commit(); e != nil {
		return nil, e
	}
	return x, nil
}
func (s *Store) Finish(x *Scan) error {
	tx, e := s.DB.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var id string
	if e = tx.QueryRow("SELECT scan_id FROM sensor_lease WHERE key='active'").Scan(&id); e != nil {
		return e
	}
	if id != x.ID {
		return fmt.Errorf("scan no longer owns the lease")
	}
	b, e := json.Marshal(x)
	if e != nil {
		return e
	}
	if _, e = tx.Exec("UPDATE sensor_scans SET body=? WHERE id=?", string(b), x.ID); e != nil {
		return e
	}
	if _, e = tx.Exec("DELETE FROM sensor_lease WHERE scan_id=?", x.ID); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) Import(x *Scan) error {
	if x.Protocol != s.Protocol {
		return fmt.Errorf("capture protocol must be %s", s.Protocol)
	}
	if x.State != "completed" || (x.Mode != "demo" && x.Mode != "live") {
		return fmt.Errorf("only completed demo/live snapshots can be imported")
	}
	r := Result{Devices: x.Devices, Warnings: x.Warnings}
	if e := ValidateResult(&r); e != nil {
		return e
	}
	if x.CreatedAt.IsZero() {
		return fmt.Errorf("missing capture timestamp")
	}
	x.Hidden = false
	x.ID = newID()
	x.Source = "imported " + x.Mode
	x.Devices = r.Devices
	x.Warnings = r.Warnings
	x.Error = ""
	b, e := json.Marshal(x)
	if e != nil {
		return e
	}
	_, e = s.DB.Exec("INSERT INTO sensor_scans VALUES (?,?)", x.ID, string(b))
	return e
}
func (s *Store) Get(id string) (*Scan, error) {
	if e := s.RecoverExpired(); e != nil {
		return nil, e
	}
	var b string
	var e error
	var hidden bool
	if id == "latest" {
		e = s.DB.QueryRow("SELECT body, EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=sensor_scans.id) FROM sensor_scans WHERE NOT EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=sensor_scans.id) ORDER BY rowid DESC LIMIT 1").Scan(&b, &hidden)
	} else {
		e = s.DB.QueryRow("SELECT body, EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=sensor_scans.id) FROM sensor_scans WHERE id=?", id).Scan(&b, &hidden)
	}
	if e != nil {
		return nil, e
	}
	var x Scan
	e = json.Unmarshal([]byte(b), &x)
	x.Hidden = hidden
	return &x, e
}
func (s *Store) List(includeHidden ...bool) ([]Scan, error) {
	if e := s.RecoverExpired(); e != nil {
		return nil, e
	}
	include := len(includeHidden) > 0 && includeHidden[0]
	rows, e := s.DB.Query("SELECT body, EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=sensor_scans.id) FROM sensor_scans WHERE ? OR NOT EXISTS(SELECT 1 FROM snapshot_hidden h WHERE h.id=sensor_scans.id) ORDER BY rowid DESC LIMIT 100", include)
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
		var x Scan
		if e = json.Unmarshal([]byte(b), &x); e != nil {
			return nil, e
		}
		x.Hidden = hidden
		out = append(out, x)
	}
	return out, rows.Err()
}

func (s *Store) SetHidden(id string, hidden bool) error {
	if e := s.RecoverExpired(); e != nil {
		return e
	}
	return snapshots.Mutate(s.DB, "sensor_scans", id, &hidden)
}
func (s *Store) Delete(id string) error {
	if e := s.RecoverExpired(); e != nil {
		return e
	}
	return snapshots.Mutate(s.DB, "sensor_scans", id, nil)
}
