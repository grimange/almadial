//go:build linux

package provision

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestVICIdialAMIExchange(t *testing.T) {
	secret := strings.Repeat("b", 20)
	base := []string{
		"Response: Success\r\nActionID: s05-login\r\nMessage: Authentication accepted\r\n\r\n",
		"Response: Success\r\nActionID: s05-ping\r\nPing: Pong\r\n\r\n",
		"Response: Success\r\nActionID: s05-status\r\nCoreStartupDate: 2026-11-10\r\nCoreStartupTime: 00:00:00\r\n\r\n",
		"Response: Error\r\nActionID: s05-denied\r\nMessage: Permission denied\r\n\r\n",
		"Response: Goodbye\r\nActionID: s05-logoff\r\nMessage: Thanks for all the fish.\r\n\r\n",
	}
	for _, tc := range []struct {
		name     string
		index    int
		response string
		valid    bool
	}{
		{"least privilege", -1, "", true},
		{"authentication denied", 0, "Response: Error\r\nActionID: s05-login\r\n\r\n", false},
		{"wrong action identity", 1, strings.ReplaceAll(base[1], "s05-ping", "other"), false},
		{"not pong", 1, strings.ReplaceAll(base[1], "Pong", "Unknown"), false},
		{"missing native startup status", 2, "Response: Success\r\nActionID: s05-status\r\n\r\n", false},
		{"excessive command permission", 3, "Response: Follows\r\nActionID: s05-denied\r\nOutput: Asterisk 22.12.0\r\n\r\n", false},
		{"other command failure", 3, strings.ReplaceAll(base[3], "Permission denied", "No such command"), false},
		{"duplicate response", 1, "Response: Error\r\n" + base[1], false},
		{"overlong response", 1, "Message: " + strings.Repeat("x", 70<<10), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			_ = client.SetDeadline(time.Now().Add(2 * time.Second))
			done := make(chan error, 1)
			go func() {
				defer server.Close()
				if _, e := io.WriteString(server, "Asterisk Call Manager/11.0.0\r\n"); e != nil {
					done <- e
					return
				}
				r := bufio.NewReader(server)
				for i, response := range base {
					request := ""
					for {
						line, e := r.ReadString('\n')
						if e != nil {
							done <- e
							return
						}
						request += line
						if line == "\r\n" {
							break
						}
					}
					actions := []string{"Login", "Ping", "CoreStatus", "Command", "Logoff"}
					if !strings.Contains(request, "Action: "+actions[i]+"\r\n") || strings.Contains(request, "Originate") {
						done <- fmt.Errorf("unexpected AMI action")
						return
					}
					if i == 0 && (!strings.Contains(request, "Events: off\r\n") || !strings.Contains(request, "Secret: "+secret+"\r\n")) {
						done <- fmt.Errorf("incorrect authentication request")
						return
					}
					if i == tc.index {
						response = tc.response
					}
					if _, e := io.WriteString(server, response); e != nil {
						done <- e
						return
					}
				}
				done <- nil
			}()
			err := probeVICIdialAMI(client, secret)
			_ = client.Close()
			serverErr := <-done
			if (err == nil) != tc.valid {
				t.Fatal("unexpected AMI qualification result", err)
			}
			if tc.valid && serverErr != nil {
				t.Fatal(serverErr)
			}
			if err != nil && strings.Contains(err.Error(), secret) {
				t.Fatal("credential in diagnostic")
			}
		})
	}
}

func TestVICIdialManagerInventory(t *testing.T) {
	valid := "  username\n----------\nalmadial-s05\n-------------------\n1 manager users configured.\n"
	if verifyVICIdialManagerUsers(valid) != nil {
		t.Fatal("closed local AMI inventory rejected")
	}
	for _, s := range []string{"There are no manager users.", strings.ReplaceAll(valid, "almadial-s05", "admin"), strings.ReplaceAll(valid, "1 manager", "2 manager"), valid + "admin\n", strings.ReplaceAll(valid, "almadial-s05", "almadial-s05\nalmadial-s05")} {
		if verifyVICIdialManagerUsers(s) == nil {
			t.Fatal("unexpected account inventory accepted")
		}
	}
}
