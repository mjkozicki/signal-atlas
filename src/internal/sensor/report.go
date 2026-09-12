package sensor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

type Change struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}

func Diff(a, b *Scan) ([]Change, error) {
	if a.Protocol != b.Protocol || a.Mode != b.Mode {
		return nil, fmt.Errorf("compare scans of the same protocol and mode")
	}
	if a.State != "completed" || b.State != "completed" {
		return nil, fmt.Errorf("only completed scans can be compared")
	}
	out := []Change{}
	before, after := map[string]Device{}, map[string]bool{}
	for _, d := range a.Devices {
		before[d.ID] = d
	}
	for _, d := range b.Devices {
		after[d.ID] = true
		old, ok := before[d.ID]
		if !ok {
			out = append(out, Change{d.ID, d.Name, "added", "Identifier first observed in this comparison"})
			continue
		}
		if old.Name != d.Name || !reflect.DeepEqual(old.Details, d.Details) {
			out = append(out, Change{d.ID, d.Name, "changed", "Name or advertised/read metadata changed"})
		}
		if !reflect.DeepEqual(old.RSSI, d.RSSI) {
			out = append(out, Change{d.ID, d.Name, "signal", "Received signal observation changed"})
		}
	}
	for _, d := range a.Devices {
		if !after[d.ID] {
			out = append(out, Change{d.ID, d.Name, "not observed", "Absent from the later scan; identifier rotation, range, or presentation may explain absence"})
		}
	}
	return out, nil
}
func safeCSV(s string) string {
	trim := strings.TrimLeft(s, " \t\r\n")
	if len(trim) > 0 && strings.ContainsRune("=+-@", rune(trim[0])) {
		return "'" + s
	}
	return s
}
func Export(w io.Writer, s *Scan, format string) error {
	if format == "json" {
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		return e.Encode(s)
	}
	if format != "csv" {
		return fmt.Errorf("format must be json or csv")
	}
	writer := csv.NewWriter(w)
	if e := writer.Write([]string{"protocol", "mode", "id", "name", "first_seen", "last_seen", "observations", "rssi_dbm", "details_json"}); e != nil {
		return e
	}
	for _, d := range s.Devices {
		rssi := ""
		if d.RSSI != nil {
			rssi = strconv.Itoa(*d.RSSI)
		}
		details, e := json.Marshal(d.Details)
		if e != nil {
			return e
		}
		row := []string{s.Protocol, s.Mode, d.ID, d.Name, d.FirstSeen.String(), d.LastSeen.String(), strconv.Itoa(d.Observations), rssi, string(details)}
		for i, v := range row {
			if i != 7 {
				row[i] = safeCSV(v)
			}
		}
		if e = writer.Write(row); e != nil {
			return e
		}
	}
	writer.Flush()
	return writer.Error()
}
