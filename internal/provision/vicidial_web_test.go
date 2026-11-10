//go:build linux

package provision

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type vicidialRoundTrip func(*http.Request) (*http.Response, error)

func (f vicidialRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestVICIdialLocalHTTPPostconditions(t *testing.T) {
	for _, defect := range []string{"", "static PHP", "PHP failure", "admin bypass", "missing agent form", "configuration exposed", "redirect", "oversized body"} {
		t.Run(defect, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: vicidialRoundTrip(func(r *http.Request) (*http.Response, error) {
				requests++
				if r.Method != "GET" || r.URL.Host != "127.0.0.1" || r.URL.Scheme != "http" || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" || r.URL.RawQuery != "" {
					t.Fatal("unsafe HTTP request")
				}
				status, body, header := 403, "forbidden", http.Header{"Content-Type": []string{"text/html"}}
				switch r.URL.Path {
				case "/vicidial/admin.php":
					status = 401
					header.Set("WWW-Authenticate", `Basic realm="VICIDIAL"`)
				case "/agc/vicidial.php":
					status = 200
					body = `<form><input name="phone_login"></form>`
				}
				switch defect {
				case "static PHP":
					body = "<?php echo 'configuration';"
				case "PHP failure":
					body = "Fatal error: uncaught TypeError"
				case "admin bypass":
					if r.URL.Path == "/vicidial/admin.php" {
						status = 200
					}
				case "missing agent form":
					if r.URL.Path == "/agc/vicidial.php" {
						body = "default Apache page"
					}
				case "configuration exposed":
					if strings.HasSuffix(r.URL.Path, "options.php") {
						status = 200
					}
				case "redirect":
					status = 302
					header.Set("Location", "https://example.invalid/")
				case "oversized body":
					body = strings.Repeat("x", (2<<20)+1)
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
			})}
			err := inspectVICIdialHTTP(context.Background(), client)
			if (err == nil) != (defect == "") {
				t.Fatal(err)
			}
			if defect == "" && requests != 4 {
				t.Fatal("missing resource checks")
			}
		})
	}
}
func TestVICIdialWebSocketPolicy(t *testing.T) {
	good := `tcp LISTEN 0 511 127.0.0.1:80 0.0.0.0:* users:(("httpd",pid=2500,fd=3),("httpd",pid=2400,fd=3))`
	unrelated := `tcp LISTEN 0 80 127.0.0.1:3306 0.0.0.0:* users:(("mariadbd",pid=1500,fd=1))`
	for _, tc := range []struct {
		name, raw string
		pid       int
		valid     bool
	}{
		{"local", good, 2400, true}, {"accepted database preserved", good + "\n" + unrelated, 2400, true},
		{"public IPv4", strings.Replace(good, "127.0.0.1:80", "0.0.0.0:80", 1), 2400, false},
		{"public IPv6", strings.Replace(good, "127.0.0.1:80", "[::]:80", 1), 2400, false},
		{"additional public HTTPS", good + "\n" + strings.Replace(good, "127.0.0.1:80", "0.0.0.0:443", 1), 2400, false},
		{"wrong process", strings.ReplaceAll(good, "httpd", "foreign"), 2400, false},
		{"wrong PID", good, 9999, false}, {"PID substring", strings.ReplaceAll(good, "pid=2400,", "pid=24001,"), 2400, false},
		{"duplicate listener", good + "\n" + good, 2400, false}, {"no listener", unrelated, 2400, false},
		{"unapproved HTTP port", good + "\n" + strings.Replace(good, "127.0.0.1:80", "127.0.0.1:8080", 1), 2400, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := verifyVICIdialWebListeners(tc.raw, tc.pid); (err == nil) != tc.valid {
				t.Fatal(err)
			}
		})
	}
}
func TestVICIdialWebEnvironment(t *testing.T) {
	const options = "OPTIONS=-f /etc/httpd/conf/almadial-s05.conf"
	for _, raw := range []string{options, `"` + options + `"`, `LANG=C "` + options + `"`, `"` + options + `" LANG=C`} {
		if !vicidialWebEnvironment(raw) {
			t.Fatal("accepted environment rejected", raw)
		}
	}
	for _, raw := range []string{"LANG=C", options + " -DFOREIGN", options + " OPTIONS=-f /etc/httpd/conf/httpd.conf", options + " LD_PRELOAD=/tmp/foreign", `"` + options + `""`, options + " LANG=C OTHER=value"} {
		if vicidialWebEnvironment(raw) {
			t.Fatal("foreign environment accepted", raw)
		}
	}
}
