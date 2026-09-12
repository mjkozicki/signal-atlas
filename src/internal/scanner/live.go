package scanner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func command(ctx context.Context, name string, args ...string) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, e := exec.CommandContext(cctx, name, args...).CombinedOutput()
	if e != nil {
		return nil, fmt.Errorf("%s failed: %w: %s", name, e, strings.TrimSpace(string(out)))
	}
	return out, nil
}
func CaptureLive(ctx context.Context, iface string, duration time.Duration) (*Scan, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("live passive capture requires Linux with a preconfigured monitor interface; on %s, import a radiotap PCAP/PCAPNG instead", runtime.GOOS)
	}
	if duration < time.Second || duration > time.Minute {
		return nil, fmt.Errorf("capture duration must be 1–60 seconds")
	}
	if _, e := net.InterfaceByName(iface); e != nil {
		return nil, e
	}
	info, e := command(ctx, "iw", "dev", iface, "info")
	if e != nil {
		return nil, e
	}
	if !strings.Contains(string(info), "type monitor") {
		return nil, fmt.Errorf("%s is not a monitor interface; scanner never changes your adapter mode", iface)
	}
	f, e := os.CreateTemp("", "wifi-scan-*.pcap")
	if e != nil {
		return nil, e
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)
	cctx, cancel := context.WithTimeout(ctx, duration+10*time.Second)
	defer cancel()
	out, e := exec.CommandContext(cctx, "dumpcap", "-i", iface, "-p", "-P", "-s", "2048", "-f", "type mgt subtype beacon or type mgt subtype probe-resp", "-a", "duration:"+strconv.Itoa(int(duration.Seconds())), "-a", "filesize:32768", "-c", "250000", "-w", name).CombinedOutput()
	if e != nil {
		return nil, fmt.Errorf("passive capture failed (check dumpcap permissions): %w: %s", e, strings.TrimSpace(string(out)))
	}
	file, e := os.Open(name)
	if e != nil {
		return nil, e
	}
	defer file.Close()
	return ReadCapture(file, "live:"+iface)
}

type Connection struct {
	SSID      string
	BSSID     string
	Interface string
}

func LocalConnection(ctx context.Context, iface string) (Connection, error) {
	c := Connection{Interface: iface}
	if runtime.GOOS != "linux" {
		return c, fmt.Errorf("active checks require Linux iw connection validation; unsupported platforms fail closed")
	}
	i, e := net.InterfaceByName(iface)
	if e != nil {
		return c, e
	}
	if i.Flags&net.FlagUp == 0 {
		return c, fmt.Errorf("interface is down")
	}
	out, e := command(ctx, "iw", "dev", iface, "link")
	if e != nil {
		return c, e
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Connected to ") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				c.BSSID = strings.ToLower(fields[2])
			}
		}
		if strings.HasPrefix(line, "SSID: ") {
			c.SSID = strings.TrimPrefix(line, "SSID: ")
		}
	}
	if c.BSSID == "" || c.SSID == "" {
		return c, fmt.Errorf("cannot verify an associated Wi-Fi connection")
	}
	return c, nil
}
func AuthorizeActive(inv Inventory, network string, c Connection) (AuthorizedNetwork, error) {
	if e := inv.Validate(); e != nil {
		return AuthorizedNetwork{}, e
	}
	for _, n := range inv.Authorized {
		if n.SSID == network {
			if c.SSID != network || !inv.Owns(AP{SSID: c.SSID, BSSID: c.BSSID}) {
				return n, fmt.Errorf("connected SSID and BSSID do not match the exact authorized network")
			}
			if len(n.Targets) == 0 {
				return n, fmt.Errorf("no explicit active targets configured for %q", network)
			}
			return n, nil
		}
	}
	return AuthorizedNetwork{}, fmt.Errorf("network %q is not authorized", network)
}

type CheckResult struct {
	IP        string `json:"ip"`
	Port      int    `json:"port"`
	Reachable bool   `json:"reachable"`
	Detail    string `json:"detail"`
}

func ActiveAudit(ctx context.Context, store *Store, network, iface string, confirmed bool) ([]CheckResult, error) {
	if !confirmed {
		return nil, fmt.Errorf("active audit needs --confirm-active plus an exact authorized SSID/BSSID and explicit targets")
	}
	inv, e := store.Inventory()
	if e != nil {
		return nil, e
	}
	c, e := LocalConnection(ctx, iface)
	if e != nil {
		return nil, e
	}
	n, e := AuthorizeActive(inv, network, c)
	if e != nil {
		return nil, e
	}
	if e = store.AcquireAudit(network); e != nil {
		return nil, e
	}
	if e = store.Log("active.started", map[string]any{"network": network, "interface": iface, "bssid": c.BSSID, "targets": n.Targets}); e != nil {
		return nil, e
	}
	results := []CheckResult{}
	for _, t := range n.Targets {
		for _, p := range t.Ports {
			fresh, err := LocalConnection(ctx, iface)
			if err != nil {
				return results, err
			}
			if _, err = AuthorizeActive(inv, network, fresh); err != nil {
				return results, err
			}
			conn, err := dialBound(ctx, iface, t.IP, p)
			r := CheckResult{IP: t.IP, Port: p, Reachable: err == nil, Detail: "TCP connection accepted; reachability alone does not prove a vulnerability"}
			if err != nil {
				r.Detail = err.Error()
			} else {
				conn.Close()
			}
			results = append(results, r)
			if err = store.Log("active.result", r); err != nil {
				return results, err
			}
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			case <-time.After(250 * time.Millisecond):
			}
		}
	}
	if e = store.Log("active.completed", map[string]any{"network": network, "results": len(results)}); e != nil {
		return results, e
	}
	return results, nil
}
