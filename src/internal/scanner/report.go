package scanner

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"reflect"
	"slices"
)

type Change struct {
	BSSID  string `json:"bssid"`
	SSID   string `json:"ssid"`
	Kind   string `json:"kind"`
	Detail string `json:"detail"`
}
type Difference struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Changes []Change `json:"changes"`
}

func Diff(a, b *Scan) Difference {
	d := Difference{From: a.ID, To: b.ID, Changes: []Change{}}
	old := map[string]AP{}
	current := map[string]AP{}
	for _, x := range a.APs {
		old[x.BSSID] = x
	}
	for _, x := range b.APs {
		current[x.BSSID] = x
		o, ok := old[x.BSSID]
		if !ok {
			d.Changes = append(d.Changes, Change{x.BSSID, x.SSID, "added", "Access point first appears in this comparison"})
			continue
		}
		if !reflect.DeepEqual(o.Security, x.Security) {
			d.Changes = append(d.Changes, Change{x.BSSID, x.SSID, "security", fmt.Sprintf("%s / PMF %s → %s / PMF %s", o.Security.Protocol, o.Security.PMF, x.Security.Protocol, x.Security.PMF)})
		}
		if o.Channel != x.Channel || o.Width != x.Width || o.Band != x.Band {
			d.Changes = append(d.Changes, Change{x.BSSID, x.SSID, "radio", fmt.Sprintf("%s GHz ch %d / %d MHz → %s GHz ch %d / %d MHz", o.Band, o.Channel, o.Width, x.Band, x.Channel, x.Width)})
		}
		if o.SSID != x.SSID {
			d.Changes = append(d.Changes, Change{x.BSSID, x.SSID, "ssid", fmt.Sprintf("%q → %q", o.SSID, x.SSID)})
		}
	}
	for _, x := range a.APs {
		if _, ok := current[x.BSSID]; !ok {
			d.Changes = append(d.Changes, Change{x.BSSID, x.SSID, "absent", "Not observed in the later capture; absence does not prove removal"})
		}
	}
	oldF, newF := map[string]bool{}, map[string]bool{}
	for _, f := range a.Findings {
		oldF[f.ID] = true
	}
	for _, f := range b.Findings {
		newF[f.ID] = true
		if !oldF[f.ID] {
			d.Changes = append(d.Changes, Change{f.BSSID, f.SSID, "new finding", f.Title})
		}
	}
	for _, f := range a.Findings {
		if !newF[f.ID] {
			detail := "No longer observed: " + f.Title
			if _, ok := current[f.BSSID]; !ok {
				detail += " (AP absent; remediation unverified)"
			}
			d.Changes = append(d.Changes, Change{f.BSSID, f.SSID, "finding absent", detail})
		}
	}
	return d
}
func Export(w io.Writer, s *Scan, format string) error {
	switch format {
	case "json":
		e := json.NewEncoder(w)
		e.SetIndent("", "  ")
		return e.Encode(s)
	case "sarif":
		rules := []map[string]any{}
		results := []map[string]any{}
		ids := []string{}
		for _, f := range s.Findings {
			if !slices.Contains(ids, f.RuleID) {
				ids = append(ids, f.RuleID)
				rules = append(rules, map[string]any{"id": f.RuleID, "shortDescription": map[string]string{"text": f.Title}, "help": map[string]string{"text": f.Remediation}})
			}
			level := "warning"
			if f.Severity == "critical" || f.Severity == "high" {
				level = "error"
			}
			if f.Severity == "low" {
				level = "note"
			}
			results = append(results, map[string]any{"ruleId": f.RuleID, "level": level, "message": map[string]string{"text": f.Title + " — " + f.SSID + " (" + f.BSSID + "). " + f.Remediation}, "partialFingerprints": map[string]string{"apRule/v1": f.ID}, "properties": map[string]any{"severity": f.Severity, "confidence": f.Confidence, "evidence": f.Evidence}})
		}
		return json.NewEncoder(w).Encode(map[string]any{"version": "2.1.0", "$schema": "https://json.schemastore.org/sarif-2.1.0.json", "runs": []any{map[string]any{"tool": map[string]any{"driver": map[string]any{"name": "WiFi Security Scanner", "version": s.RuleVersion, "rules": rules}}, "results": results}}})
	case "html":
		return reportTemplate.Execute(w, s)
	default:
		return fmt.Errorf("unsupported format %q; choose json, sarif, or html", format)
	}
}

var reportTemplate = template.Must(template.New("report").Parse(`<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Wi-Fi security report</title><style>body{font:16px/1.6 system-ui;max-width:1000px;margin:40px auto;padding:24px;color:#172b40}h1{color:#075b67}table{border-collapse:collapse;width:100%}td,th{padding:12px;text-align:left;border-bottom:1px solid #ccd9e0}article{border:1px solid #ccd9e0;padding:24px;margin:20px 0;border-radius:10px}code{overflow-wrap:anywhere}.meta{color:#526271}@media print{article{break-inside:avoid}}</style><h1>Wi-Fi security report</h1><p class="meta">{{.ID}} · {{.CreatedAt}} · Source: {{.Source}} · Rules {{.RuleVersion}}</p><p>Score: {{if .Score}}{{.Score}} / 100{{else}}Not assessed{{end}}. Scores cover explicitly authorized APs and observed configuration only.</p>{{range .Warnings}}<p>{{.}}</p>{{end}}<h2>Access points</h2><table><tr><th>SSID / BSSID</th><th>Radio</th><th>Security</th><th>Owned</th></tr>{{range .APs}}<tr><td>{{.SSID}}<br><code>{{.BSSID}}</code></td><td>{{.Band}} GHz / channel {{.Channel}}</td><td>{{.Security.Protocol}}<br>PMF {{.Security.PMF}}</td><td>{{.Authorized}}</td></tr>{{end}}</table><h2>Findings</h2>{{range .Findings}}<article><strong>{{.Severity}} · {{.RuleID}} · Confidence {{.Confidence}}%</strong><h3>{{.Title}}</h3><p>{{.SSID}} · {{.BSSID}}</p><ul>{{range .Evidence}}<li>{{.}}</li>{{end}}</ul><p>{{.Risk}}</p><p><strong>Remediation:</strong> {{.Remediation}}</p></article>{{else}}<p>No findings in the assessed scope. This is not a guarantee that the network is secure.</p>{{end}}</html>`))
