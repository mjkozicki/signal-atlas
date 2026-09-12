// Package snapshots manages reversible visibility and explicit snapshot deletion.
package snapshots

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const Schema = `CREATE TABLE IF NOT EXISTS snapshot_hidden (id TEXT PRIMARY KEY);`

var ErrRunning = errors.New("cancel the running scan and wait for it to finish before hiding or deleting it")

// Mutate changes only a concrete snapshot. The transaction protects the running
// job check and keeps visibility metadata consistent with physical row deletion.
func Mutate(db *sql.DB, table, id string, hidden *bool) error {
	if table != "scans" && table != "sensor_scans" {
		return fmt.Errorf("invalid snapshot table")
	}
	if id == "" || id == "latest" {
		return fmt.Errorf("a concrete snapshot ID is required")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var body string
	if err = tx.QueryRow("SELECT body FROM "+table+" WHERE id=?", id).Scan(&body); err != nil {
		return err
	}
	var state struct {
		State string `json:"state"`
	}
	if err = json.Unmarshal([]byte(body), &state); err != nil {
		return err
	}
	if state.State == "running" {
		return ErrRunning
	}
	if hidden != nil && *hidden {
		_, err = tx.Exec("INSERT INTO snapshot_hidden(id) VALUES (?) ON CONFLICT DO NOTHING", id)
	} else {
		_, err = tx.Exec("DELETE FROM snapshot_hidden WHERE id=?", id)
		if err == nil && hidden == nil {
			_, err = tx.Exec("DELETE FROM "+table+" WHERE id=?", id)
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

type Store interface {
	SetHidden(string, bool) error
	Delete(string) error
}

// Register is always wrapped by the owning service's local-origin middleware.
func Register(mux *http.ServeMux, store Store) {
	reply := func(w http.ResponseWriter, status int, value any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(value)
	}
	fail := func(w http.ResponseWriter, err error) {
		code := http.StatusBadRequest
		if errors.Is(err, sql.ErrNoRows) {
			code = http.StatusNotFound
		}
		if errors.Is(err, ErrRunning) {
			code = http.StatusConflict
		}
		reply(w, code, map[string]string{"error": err.Error()})
	}
	mux.HandleFunc("PATCH /api/scans/{id}", func(w http.ResponseWriter, r *http.Request) {
		var change struct {
			Hidden *bool `json:"hidden"`
		}
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
		d.DisallowUnknownFields()
		if err := d.Decode(&change); err != nil {
			fail(w, err)
			return
		}
		if err := d.Decode(new(any)); err != io.EOF {
			fail(w, fmt.Errorf("expected one JSON document"))
			return
		}
		if change.Hidden == nil {
			fail(w, fmt.Errorf("hidden must be true or false"))
			return
		}
		if err := store.SetHidden(r.PathValue("id"), *change.Hidden); err != nil {
			fail(w, err)
			return
		}
		reply(w, 200, map[string]any{"id": r.PathValue("id"), "hidden": *change.Hidden})
	})
	mux.HandleFunc("DELETE /api/scans/{id}", func(w http.ResponseWriter, r *http.Request) {
		if err := store.Delete(r.PathValue("id")); err != nil {
			fail(w, err)
			return
		}
		reply(w, 200, map[string]string{"id": r.PathValue("id"), "state": "deleted"})
	})
}
