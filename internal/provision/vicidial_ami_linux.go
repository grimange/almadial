//go:build linux

package provision

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

// The AMI probe uses a bounded local connection with events disabled. It cannot
// originate a channel, change configuration, or dispatch a dialer workflow.
func verifyVICIdialAMI(ctx context.Context, secret string) error {
	if ctx == nil || !vicidialHex(secret, 20) {
		return errVICIdial
	}
	c, e := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", "127.0.0.1:5038")
	if e != nil {
		return errVICIdial
	}
	defer c.Close()
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if c.SetDeadline(deadline) != nil {
		return errVICIdial
	}
	// Cancellation also interrupts a blocked read when the caller has no deadline.
	stop := context.AfterFunc(ctx, func() { _ = c.Close() })
	defer stop()
	return probeVICIdialAMI(c, secret)
}

func amiLine(r *bufio.Reader) (string, error) {
	line, e := r.ReadString('\n')
	if e != nil || len(line) > 4096 || !strings.HasSuffix(line, "\r\n") {
		return "", errVICIdial
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

func amiResponse(r *bufio.Reader, id string) (map[string]string, error) {
	fields := map[string]string{}
	for n := 0; n < 128; n++ {
		line, e := amiLine(r)
		if e != nil {
			return nil, e
		}
		if line == "" {
			if fields["ActionID"] != id || fields["Response"] == "" {
				return nil, errVICIdial
			}
			return fields, nil
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok || key == "" || fields[key] != "" {
			return nil, errVICIdial
		}
		fields[key] = strings.TrimSpace(value)
	}
	return nil, errVICIdial
}

func probeVICIdialAMI(conn io.ReadWriter, secret string) error {
	if conn == nil || !vicidialHex(secret, 20) {
		return errVICIdial
	}
	// LimitReader bounds the entire exchange, including malformed peers that
	// omit line delimiters. No received diagnostic or secret is returned.
	r := bufio.NewReaderSize(io.LimitReader(conn, 64<<10), 4096)
	banner, e := amiLine(r)
	if e != nil || !strings.HasPrefix(banner, "Asterisk Call Manager/") {
		return errVICIdial
	}
	request := func(id, action, extra string) (map[string]string, error) {
		message := "Action: " + action + "\r\nActionID: " + id + "\r\n" + extra + "\r\n"
		if n, e := io.WriteString(conn, message); e != nil || n != len(message) {
			return nil, errVICIdial
		}
		return amiResponse(r, id)
	}
	login, e := request("s05-login", "Login", "Username: almadial-s05\r\nSecret: "+secret+"\r\nEvents: off\r\n")
	if e != nil || login["Response"] != "Success" {
		return errVICIdial
	}
	ping, e := request("s05-ping", "Ping", "")
	if e != nil || ping["Response"] != "Success" || ping["Ping"] != "Pong" {
		return errVICIdial
	}
	status, e := request("s05-status", "CoreStatus", "")
	if e != nil || status["Response"] != "Success" || status["CoreStartupDate"] == "" || status["CoreStartupTime"] == "" {
		return errVICIdial
	}
	denied, e := request("s05-denied", "Command", "Command: core show version\r\n")
	if e != nil || denied["Response"] != "Error" || !strings.EqualFold(denied["Message"], "Permission denied") {
		return errVICIdial
	}
	logoff, e := request("s05-logoff", "Logoff", "")
	if e != nil || logoff["Response"] != "Goodbye" {
		return errVICIdial
	}
	return nil
}

func verifyVICIdialManagerUsers(raw string) error {
	// Exact one-account inventory, independent of the configuration file.
	seen := false
	count := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line == "username" || strings.Trim(line, "-") == "" {
			continue
		}
		if line == "1 manager users configured." {
			count = true
			continue
		}
		if line == "almadial-s05" && !seen {
			seen = true
			continue
		}
		return fmt.Errorf("%w: unexpected local AMI account inventory", errVICIdial)
	}
	if !seen || !count {
		return errVICIdial
	}
	return nil
}
