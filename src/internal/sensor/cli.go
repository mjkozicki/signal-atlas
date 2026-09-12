package sensor

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func RunCLI(protocol string, factory func(string) Provider, args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Printf(`%s-scan COMMAND [options]
  status                       Provider availability (no RF scanning)
  discover --duration 10       Native BLE discovery / NFC read-only polling
  demo                         Save a synthetic snapshot, no hardware needed
  history                      List the latest 100 snapshots
  inspect --scan ID            Read a snapshot (default latest)
  import --file capture.json   Import a completed snapshot
  export --format json|csv [--output PATH] [--scan ID]
  diff --from ID --to ID        Compare completed snapshots of the same mode
  serve                        Start the independent loopback API
Common: --db PATH, --python PATH. Discovery: --adapter hci0 (BLE/Linux), --reader NAME (NFC).
`, protocol)
		return nil
	}
	f := flag.NewFlagSet(protocol, flag.ContinueOnError)
	db := f.String("db", ".data/"+protocol+".db", "database path")
	python := f.String("python", "", "native provider interpreter")
	duration := f.Int("duration", 10, "scan seconds, 1–60")
	adapter := f.String("adapter", "", "Linux Bluetooth adapter")
	reader := f.String("reader", "", "exact PC/SC reader name")
	port := "8788"
	if protocol == "nfc" {
		port = "8789"
	}
	addr := f.String("addr", "127.0.0.1:"+port, "loopback address")
	id := f.String("scan", "latest", "scan ID")
	format := f.String("format", "json", "export format")
	output := f.String("output", "", "new report path")
	file := f.String("file", "", "JSON snapshot file")
	from := f.String("from", "", "earlier scan ID")
	to := f.String("to", "latest", "later scan ID")
	if e := f.Parse(args[1:]); e != nil {
		if e == flag.ErrHelp {
			return nil
		}
		return e
	}
	if f.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	valid := map[string]bool{"status": true, "discover": true, "demo": true, "history": true, "inspect": true, "import": true, "export": true, "diff": true, "serve": true}
	if !valid[args[0]] {
		return fmt.Errorf("unknown command %q", args[0])
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	provider := factory(*python)
	print := func(v any) error { e := json.NewEncoder(os.Stdout); e.SetIndent("", "  "); return e.Encode(v) }
	if args[0] == "status" {
		return print(provider.Status(ctx))
	}
	store, e := OpenStore(*db, protocol)
	if e != nil {
		return e
	}
	defer store.Close()
	manager := NewManager(store, provider)
	defer manager.Close()
	switch args[0] {
	case "serve":
		return Listen(ctx, manager, *addr)
	case "discover", "demo":
		mode := "live"
		if args[0] == "demo" {
			mode = "demo"
		}
		s, e := manager.Start(Options{Mode: mode, Duration: *duration, Adapter: *adapter, Reader: *reader})
		if e != nil {
			return e
		}
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				manager.Cancel(s.ID)
				manager.Close()
				return ctx.Err()
			case <-ticker.C:
				x, e := store.Get(s.ID)
				if e != nil {
					return e
				}
				if x.State != "running" {
					if x.State != "completed" {
						return fmt.Errorf("scan %s: %s", x.ID, x.Error)
					}
					return print(x)
				}
			}
		}
	case "history":
		x, e := store.List()
		if e != nil {
			return e
		}
		return print(x)
	case "inspect":
		s, e := store.Get(*id)
		if e != nil {
			return e
		}
		return print(s)
	case "import":
		in, e := os.Open(*file)
		if e != nil {
			return e
		}
		defer in.Close()
		data, e := io.ReadAll(io.LimitReader(in, (4<<20)+1))
		if e != nil {
			return e
		}
		if len(data) > 4<<20 {
			return fmt.Errorf("snapshot exceeds 4 MB")
		}
		var s Scan
		if e = json.Unmarshal(data, &s); e != nil {
			return e
		}
		if e = store.Import(&s); e != nil {
			return e
		}
		return print(s)
	case "export":
		if *format != "json" && *format != "csv" {
			return fmt.Errorf("format must be json or csv")
		}
		s, e := store.Get(*id)
		if e != nil {
			return e
		}
		var writer io.Writer = os.Stdout
		if *output != "" {
			f, e := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			defer f.Close()
			writer = f
		}
		return Export(writer, s, *format)
	case "diff":
		if *from == "" {
			return fmt.Errorf("diff requires --from ID")
		}
		a, e := store.Get(*from)
		if e != nil {
			return e
		}
		b, e := store.Get(*to)
		if e != nil {
			return e
		}
		diff, e := Diff(a, b)
		if e != nil {
			return e
		}
		return print(diff)
	}
	return nil
}
