package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"gopkg.in/yaml.v3"
	"signal-atlas/src/internal/scanner"
	"signal-atlas/src/internal/web"
)

const help = `Wi-Fi Security Scanner — local wireless security observability

Usage: wifi-scan COMMAND [options]

  discover --pcap capture.pcap [--band 2.4|5|6] [--json]
  discover --interface mon0 [--duration 15s] [--json]   Linux passive capture
  inspect --ssid NAME | --bssid ADDRESS [--scan ID] [--json]
  security [--network NAME] [--scan ID] [--json]
  audit --network NAME [--active --confirm-active --interface wlan0]
  inventory [--file allowlist.yaml]
  diff SCAN_ID SCAN_ID | diff yesterday
  export --format json|sarif|html [--scan ID] [--output PATH]
  serve [--addr 127.0.0.1:8787] [--container]
  demo                              Save a synthetic scan in an empty database

Every command accepts --db PATH (default .data/wifi-scanner.db).
Offline imports work on macOS, Linux, and Windows. No radio access occurs by default.
Active checks are bounded TCP reachability checks, never credential or injection tests.
`

func main() {
	if e := run(os.Args[1:]); e != nil {
		fmt.Fprintln(os.Stderr, "Error:", e)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Print(help)
		return nil
	}
	f := flag.NewFlagSet(args[0], flag.ContinueOnError)
	db := f.String("db", ".data/wifi-scanner.db", "SQLite file")
	pcap := f.String("pcap", "", "PCAP/PCAPNG input")
	iface := f.String("interface", "", "local wireless interface")
	duration := f.Duration("duration", 15*time.Second, "passive capture duration")
	band := f.String("band", "", "2.4, 5 or 6")
	asJSON := f.Bool("json", false, "JSON output")
	ssid := f.String("ssid", "", "SSID")
	bssid := f.String("bssid", "", "BSSID")
	network := f.String("network", "", "authorized SSID")
	id := f.String("scan", "latest", "scan ID")
	format := f.String("format", "json", "export format")
	output := f.String("output", "", "output path")
	file := f.String("file", "", "inventory YAML/JSON")
	addr := f.String("addr", "127.0.0.1:8787", "loopback listen address")
	container := f.Bool("container", false, "allow 0.0.0.0 binding for Docker; publish to host loopback only")
	active := f.Bool("active", false, "enable bounded active checks")
	confirm := f.Bool("confirm-active", false, "confirm active checks against configured targets")
	// Permit options before or after positional scan IDs without losing flag values.
	flags, pos := []string{}, []string{}
	boolFlags := map[string]bool{"--json": true, "--active": true, "--confirm-active": true, "--container": true}
	for i := 1; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && !boolFlags[a] && a != "--help" && a != "-h" {
				if i+1 < len(args) {
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			pos = append(pos, a)
		}
	}
	if e := f.Parse(flags); e != nil {
		if errors.Is(e, flag.ErrHelp) {
			return nil
		}
		return e
	}
	valid := map[string]bool{"discover": true, "inspect": true, "security": true, "audit": true, "inventory": true, "diff": true, "export": true, "serve": true, "demo": true}
	if !valid[args[0]] {
		return fmt.Errorf("unknown command %q; run wifi-scan help", args[0])
	}
	store, e := scanner.OpenStore(*db)
	if e != nil {
		return e
	}
	defer store.Close()
	inv, e := store.Inventory()
	if e != nil {
		return e
	}
	printJSON := func(v any) error { enc := json.NewEncoder(os.Stdout); enc.SetIndent("", "  "); return enc.Encode(v) }
	get := func() (*scanner.Scan, error) {
		s, e := store.Get(*id)
		if errors.Is(e, sql.ErrNoRows) {
			return nil, fmt.Errorf("no scan found; run discover --pcap FILE or demo first")
		}
		return s, e
	}
	switch args[0] {
	case "serve":
		return web.Serve(store, *addr, *container)
	case "inventory":
		if *file != "" {
			data, e := os.ReadFile(*file)
			if e != nil {
				return e
			}
			d := yaml.NewDecoder(strings.NewReader(string(data)))
			d.KnownFields(true)
			if e = d.Decode(&inv); e != nil {
				return e
			}
			if e = store.SetInventory(inv); e != nil {
				return e
			}
		}
		return printJSON(inv)
	case "demo":
		if _, e = store.Get("latest"); e == nil {
			return fmt.Errorf("demo requires an empty database; use --db .data/demo.db")
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if len(inv.Authorized) > 0 {
			return fmt.Errorf("demo requires an empty inventory")
		}
		s, demoInv := scanner.Demo()
		if e = store.SetInventory(demoInv); e != nil {
			return e
		}
		if e = store.Save(s); e != nil {
			return e
		}
		return printJSON(s)
	case "discover":
		if (*pcap == "") == (*iface == "") {
			return fmt.Errorf("choose exactly one of --pcap FILE or --interface NAME")
		}
		if *band != "" && *band != "2.4" && *band != "5" && *band != "6" {
			return fmt.Errorf("band must be 2.4, 5, or 6")
		}
		var s *scanner.Scan
		if *pcap != "" {
			var in *os.File
			in, e = os.Open(*pcap)
			if e != nil {
				return e
			}
			defer in.Close()
			s, e = scanner.ReadCapture(in, "pcap:"+filepath.Base(*pcap))
		} else {
			s, e = scanner.CaptureLive(context.Background(), *iface, *duration)
		}
		if e != nil {
			return e
		}
		if *band != "" {
			filtered := []scanner.AP{}
			for _, a := range s.APs {
				if a.Band == *band {
					filtered = append(filtered, a)
				}
			}
			s.APs = filtered
			s.Source += " (band " + *band + ")"
		}
		scanner.Analyze(s, inv)
		if e = store.Save(s); e != nil {
			return e
		}
		if *asJSON {
			return printJSON(s)
		}
		printAPs(s.APs)
		fmt.Printf("\nSaved %s · %d findings\n", s.ID, len(s.Findings))
		for _, w := range s.Warnings {
			fmt.Println(w)
		}
		return nil
	case "inspect":
		s, e := get()
		if e != nil {
			return e
		}
		if *ssid == "" && *bssid == "" {
			return fmt.Errorf("inspect needs --ssid or --bssid")
		}
		ap := []scanner.AP{}
		for _, a := range s.APs {
			if (*ssid == "" || a.SSID == *ssid) && (*bssid == "" || strings.EqualFold(a.BSSID, *bssid)) {
				ap = append(ap, a)
			}
		}
		if len(ap) == 0 {
			return fmt.Errorf("no matching access point")
		}
		return printJSON(ap)
	case "security", "audit":
		if args[0] == "audit" && *network == "" {
			return fmt.Errorf("audit needs --network NAME")
		}
		if *active {
			if args[0] != "audit" {
				return fmt.Errorf("--active is only valid with audit")
			}
			r, e := scanner.ActiveAudit(context.Background(), store, *network, *iface, *confirm)
			if e != nil {
				return e
			}
			return printJSON(r)
		}
		s, e := get()
		if e != nil {
			return e
		}
		if args[0] == "audit" {
			found := false
			for _, n := range inv.Authorized {
				if n.SSID == *network {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("network is not authorized")
			}
			scanner.Analyze(s, inv)
		}
		findings := []scanner.Finding{}
		for _, x := range s.Findings {
			if *network == "" || x.SSID == *network {
				findings = append(findings, x)
			}
		}
		return printJSON(findings)
	case "diff":
		var a, b *scanner.Scan
		if len(pos) == 1 && pos[0] == "yesterday" {
			b, e = get()
			if e != nil {
				return e
			}
			var scans []scanner.Scan
			scans, e = store.List()
			if e != nil {
				return e
			}
			cutoff := time.Now().Add(-24 * time.Hour)
			for i := range scans {
				if scans[i].CreatedAt.Before(cutoff) {
					a = &scans[i]
					break
				}
			}
			if a == nil {
				return fmt.Errorf("no scan at least 24 hours old among the latest 100 scans")
			}
		} else if len(pos) == 2 {
			a, e = store.Get(pos[0])
			if e != nil {
				return e
			}
			b, e = store.Get(pos[1])
			if e != nil {
				return e
			}
		} else {
			return fmt.Errorf("diff needs two scan IDs or yesterday")
		}
		return printJSON(scanner.Diff(a, b))
	case "export":
		if *format != "json" && *format != "sarif" && *format != "html" {
			return fmt.Errorf("format must be json, sarif, or html")
		}
		s, e := get()
		if e != nil {
			return e
		}
		var w io.Writer = os.Stdout
		if *output != "" {
			out, e := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			defer out.Close()
			w = out
		}
		return scanner.Export(w, s, *format)
	}
	return nil
}
func printAPs(aps []scanner.AP) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 3, ' ', 0)
	fmt.Fprintln(w, "SSID\tBSSID\tBAND\tCH\tRSSI\tSECURITY\tPMF\tOWNED")
	for _, a := range aps {
		r := "unknown"
		if a.RSSI != nil {
			r = fmt.Sprint(*a.RSSI)
		}
		fmt.Fprintf(w, "%q\t%s\t%s\t%d\t%s\t%s\t%s\t%t\n", a.SSID, a.BSSID, a.Band, a.Channel, r, a.Security.Protocol, a.Security.PMF, a.Authorized)
	}
	w.Flush()
}
