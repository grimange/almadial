package main

import "testing"

func TestALM34TargetBoundary(t *testing.T) {
	const task = "it8aq4utqix0uz8sjgxrtssk"
	const machine = "d8e73788e2a9328a8655aa707be8816a"
	if !authorizedTarget(task, "almadial-alm34-bootstrap-dev", machine, "media") {
		t.Fatal("authorized developer target rejected")
	}
	for _, c := range []struct{ task, vm, machine, scenario string }{
		{task, "almadial-alm34-operator", machine, "media"},
		{task, "almadial-alma10", machine, "media"},
		{task, "almadial-alm34-bootstrap-dev", "00000000000000000000000000000000", "media"},
		{task, "almadial-alm34-bootstrap-dev", machine, "reboot"},
		{"unknown", "almadial-alm34-bootstrap-dev", machine, "media"},
	} {
		if authorizedTarget(c.task, c.vm, c.machine, c.scenario) {
			t.Fatal("unauthorized target accepted", c.vm, c.scenario)
		}
	}
}

func TestRemoteTargetBoundary(t *testing.T) {
	const task = "a1doi4qsyn1lspqtydw84smh"
	const machine = "436ca8ae883439638507ab64b02d98e8"
	if !authorizedTarget(task, "almadial-alm46-remote-dev", machine, "remote-media") {
		t.Fatal("development target rejected")
	}
	for _, vm := range []string{"almadial-alm34-operator", "almadial-alma10", "dialer01"} {
		if authorizedTarget(task, vm, machine, "remote-media") {
			t.Fatal("protected/unknown target admitted")
		}
	}
	if authorizedTarget(task, "almadial-alm46-remote-dev", machine, "media") {
		t.Fatal("temporary loopback overlay admitted")
	}
}
