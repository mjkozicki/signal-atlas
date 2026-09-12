package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// Service targets are fixed loopback endpoints, never supplied by a browser request.
// The outer dashboard middleware validates Host, Origin, and the mutation header.
func serviceProxy(protocol string) http.Handler {
	port := map[string]string{"bluetooth": "8788", "nfc": "8789"}[protocol]
	target, _ := url.Parse("http://127.0.0.1:" + port)
	return &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
			r.Out.URL.Path = "/api/" + strings.TrimPrefix(r.In.URL.Path, "/api/"+protocol+"/")
			r.Out.URL.RawPath = ""
			r.Out.Header.Del("Origin")
			r.Out.Header.Del("Forwarded")
			r.Out.Header.Del("X-Forwarded-Host")
			r.Out.Header.Del("X-Forwarded-For")
		},
		Transport: &http.Transport{Proxy: nil, ResponseHeaderTimeout: 12 * time.Second, IdleConnTimeout: 30 * time.Second, MaxIdleConnsPerHost: 2},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, e error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(503)
			fmt.Fprintf(w, `{"error":%q}`, protocol+" service is offline. Start bin/"+protocol+"-scan serve, or run make serve-all.")
		},
	}
}
