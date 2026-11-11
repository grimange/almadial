//go:build linux

package provision

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/grimange/almadial/internal/execution"
)

// Anonymous GETs only: no administrator credentials, sessions, first-login
// setup, record creation, dialer activation or real endpoint configuration.
func vicidialWebClient() *http.Client {
	return &http.Client{
		Timeout:       10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		Transport:     &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 5 * time.Second}).DialContext, DisableKeepAlives: true, MaxResponseHeaderBytes: 64 << 10},
	}
}
func inspectVICIdialHTTP(ctx context.Context, client *http.Client) error {
	if ctx == nil || client == nil {
		return errVICIdial
	}
	bounded := *client
	bounded.Timeout = 10 * time.Second
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	bounded.Jar = nil
	for _, path := range []string{"/vicidial/admin.php", "/agc/vicidial.php", "/vicidial/options.php", "/agc/options.php"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1"+path, nil)
		if err != nil {
			return errVICIdial
		}
		req.Header.Set("Accept", "text/html")
		res, err := bounded.Do(req)
		if err != nil {
			return errVICIdial
		}
		body, readErr := io.ReadAll(io.LimitReader(res.Body, (2<<20)+1))
		closeErr := res.Body.Close()
		if readErr != nil || closeErr != nil || len(body) > 2<<20 || !verifyVICIdialHTTPResponse(path, res.StatusCode, res.Header, body) {
			return errVICIdial
		}
	}
	return nil
}
func verifyVICIdialHTTPResponse(path string, status int, header http.Header, body []byte) bool {
	text := strings.ToLower(string(body))
	if strings.Contains(text, "<?php") || strings.Contains(text, "fatal error") || strings.Contains(text, "uncaught ") || strings.Contains(text, "database connect error") {
		return false
	}
	switch path {
	case "/vicidial/admin.php":
		return status == http.StatusUnauthorized && strings.Contains(strings.ToLower(header.Get("WWW-Authenticate")), "basic")
	case "/agc/vicidial.php":
		return status == http.StatusOK && strings.Contains(strings.ToLower(header.Get("Content-Type")), "text/html") && strings.Contains(text, "phone_login") && strings.Contains(text, "<form")
	case "/vicidial/options.php", "/agc/options.php":
		return status == http.StatusForbidden
	}
	return false
}

// ss is independent of Apache configuration text. Reject public HTTP/HTTPS,
// duplicate local listeners and unexpected processes on the qualification port.
func verifyVICIdialWebListeners(raw string, pid int) error {
	return verifyVICIdialWebListenersTLS(raw, pid, false)
}
func verifyVICIdialWebListenersTLS(raw string, pid int, tlsHandoff bool) error {
	count := 0
	if pid < 2 {
		return errVICIdial
	}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return errVICIdial
		}
		local := fields[4]
		host, port, err := net.SplitHostPort(local)
		if err != nil {
			return errVICIdial
		}
		relevant := port == "80" || port == "443" || strings.Contains(line, "\"httpd\"") || strings.Contains(line, "\"php-fpm\"")
		if !relevant {
			continue
		}
		if fields[0] != "tcp" || host != "127.0.0.1" || (port != "80" && !(tlsHandoff && port == "443")) || !strings.Contains(line, "\"httpd\"") || !strings.Contains(line, "pid="+strconv.Itoa(pid)+",") {
			return errVICIdial
		}
		count++
	}
	if count != 1 && !(tlsHandoff && count == 2) {
		return errVICIdial
	}
	return nil
}

func inspectVICIdialWebService(ctx context.Context, runner execution.CommandRunner) (string, error) {
	return inspectVICIdialWebServiceTLS(ctx, runner, false)
}
func inspectVICIdialWebServiceTLS(ctx context.Context, runner execution.CommandRunner, tlsHandoff bool) (string, error) {
	run := func(c execution.Command) (string, error) {
		r, e := runner.Run(ctx, c)
		if e != nil || !r.Started || r.ExitCode != 0 || r.StdoutTruncated || r.StderrTruncated {
			return "", errVICIdial
		}
		return r.Stdout, nil
	}
	if ctx == nil || runner == nil {
		return "", errVICIdial
	}
	raw, err := run(dbCommand("/usr/bin/systemctl", "show", "httpd.service", "--property=LoadState,ActiveState,MainPID,FragmentPath,DropInPaths,Environment"))
	if err != nil {
		return "", errVICIdial
	}
	fields, e := verifyVICIdialWebUnit(raw, true)
	if e != nil {
		return "", e
	}
	pid, _ := strconv.Atoi(fields["MainPID"])
	sockets, e := run(dbCommand("/usr/sbin/ss", "-H", "-lntup"))
	if e != nil || verifyVICIdialWebListenersTLS(sockets, pid, tlsHandoff) != nil {
		return "", errVICIdial
	}
	// Require the distribution FPM service to be active. The separate application
	// HTTP check verifies that the configured Unix-socket handler serves PHP.
	fpm, e := run(dbCommand("/usr/bin/systemctl", "is-active", "php-fpm.service"))
	if e != nil || strings.TrimSpace(fpm) != "active" {
		return "", errVICIdial
	}
	return hash(struct{ Unit, Options string }{fields["FragmentPath"], fields["Environment"]}), nil
}

func vicidialWebEnvironment(raw string) bool {
	const options = "OPTIONS=-f /etc/httpd/conf/almadial-s05.conf"
	// systemctl quotes values containing spaces; accept only this closed option
	// and the existing vendor LANG. Additional environment overrides are foreign.
	for _, value := range []string{`"` + options + `"`, options} {
		if strings.Count(raw, value) == 1 {
			rest := strings.TrimSpace(strings.Replace(raw, value, "", 1))
			if rest == "" || rest == "LANG=C" {
				return true
			}
		}
	}
	return false
}

func verifyVICIdialWebUnit(raw string, active bool) (map[string]string, error) {
	fields := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if _, exists := fields[key]; !ok || exists {
			return nil, errVICIdial
		}
		fields[key] = value
	}
	if fields["LoadState"] != "loaded" || fields["FragmentPath"] != "/usr/lib/systemd/system/httpd.service" || active && fields["ActiveState"] != "active" || !active && fields["ActiveState"] != "inactive" && fields["ActiveState"] != "active" {
		return nil, errVICIdial
	}
	pid, e := strconv.Atoi(fields["MainPID"])
	if e != nil || active && pid < 2 || !active && pid < 0 || !vicidialWebEnvironment(fields["Environment"]) {
		return nil, errVICIdial
	}
	dropins := strings.Fields(fields["DropInPaths"])
	found := false
	for _, p := range dropins {
		switch p {
		case "/etc/systemd/system/httpd.service.d/almadial-s05.conf":
			found = true
		case "/etc/systemd/system/httpd.service.d/php-fpm.conf":
		default:
			return nil, errVICIdial
		}
	}
	if !found {
		return nil, errVICIdial
	}
	return fields, nil
}
