package sensor

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type Manager struct {
	Store    *Store
	Provider Provider
	mu       sync.Mutex
	jobs     map[string]context.CancelFunc
	wg       sync.WaitGroup
	closed   bool
}

func NewManager(s *Store, p Provider) *Manager {
	return &Manager{Store: s, Provider: p, jobs: map[string]context.CancelFunc{}}
}
func (m *Manager) Start(o Options) (*Scan, error) {
	if e := o.Validate(m.Store.Protocol); e != nil {
		return nil, e
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, fmt.Errorf("service is shutting down")
	}
	scan, e := m.Store.Begin(o)
	if e != nil {
		return nil, e
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(o.Duration+15)*time.Second)
	m.jobs[scan.ID] = cancel
	m.wg.Add(1)
	// The caller gets an immutable initial snapshot; only the worker mutates its copy.
	initial := *scan
	go func() {
		defer m.wg.Done()
		defer cancel()
		var result Result
		var err error
		if o.Mode == "demo" {
			result = m.Provider.Demo()
		} else {
			result, err = m.Provider.Scan(ctx, o)
		}
		if err == nil {
			err = ValidateResult(&result)
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		now := time.Now().UTC()
		scan.EndedAt = &now
		switch {
		case ctx.Err() == context.Canceled:
			scan.State = "canceled"
			scan.Error = "Scan canceled; incomplete observations were not saved."
		case err != nil:
			scan.State = "failed"
			scan.Error = err.Error()
		default:
			scan.State = "completed"
			scan.Devices = result.Devices
			scan.Warnings = result.Warnings
		}
		if e := m.Store.Finish(scan); e != nil {
			fmt.Fprintf(os.Stderr, "%s scan persistence error: %v\n", m.Store.Protocol, e)
		}
		delete(m.jobs, scan.ID)
	}()
	return &initial, nil
}
func (m *Manager) Cancel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cancel, ok := m.jobs[id]
	if !ok {
		return fmt.Errorf("scan is not running in this service process")
	}
	cancel()
	return nil
}
func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	for _, cancel := range m.jobs {
		cancel()
	}
	m.mu.Unlock()
	m.wg.Wait()
}
