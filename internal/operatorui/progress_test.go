package operatorui

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/operatorlog"
)

func TestProgressSequenceAndTaxonomy(t *testing.T) {
	var out, err bytes.Buffer
	r, _ := operatorlog.NewRedactor([]string{"FAKE-progress-secret"})
	l, _ := operatorlog.New(&out, &err, nil, r)
	for _, kind := range []EventType{StagePending, StageStarted, ActionStarted, ActionCompleted, StageCompleted} {
		if e := RenderEvent(l, Event{Type: kind, StageID: "S03", ActionID: "db.init"}); e != nil {
			t.Fatal(e)
		}
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 5 {
		t.Fatal("event ordering")
	}
	for i, kind := range []EventType{StagePending, StageStarted, ActionStarted, ActionCompleted, StageCompleted} {
		var rec operatorlog.Record
		json.Unmarshal([]byte(lines[i]), &rec)
		if rec.Message != string(kind) || rec.Stage != "S03" {
			t.Fatal("unstable render")
		}
	}
	for _, category := range []execution.Category{execution.Contract, execution.Precondition, execution.Start, execution.Exit, execution.Timeout, execution.Canceled, execution.OutputLimit, execution.Postcondition, execution.Dependency, execution.Internal} {
		t.Run(string(category), func(t *testing.T) {
			failure := &execution.Error{Category: category, StageID: "S03", ActionID: "db.init", Detail: "FAKE-progress-secret"}
			before := err.Len()
			if e := RenderEvent(l, Event{Type: ActionFailed, StageID: "S03", ActionID: "db.init", Failure: failure, Diagnostics: []string{"FAKE-progress-", "secret"}}); e != nil {
				t.Fatal(e)
			}
			var rec operatorlog.Record
			if e := json.Unmarshal(bytes.TrimSpace(err.Bytes()[before:]), &rec); e != nil {
				t.Fatal(e)
			}
			if rec.Code != string(category) || rec.Next == "" || strings.Contains(err.String(), "FAKE-progress") {
				t.Fatal("unsafe or unactionable failure")
			}
		})
	}
	for _, kind := range []EventType{StageBlocked, ExecutionCanceled, ReconciliationRequired} {
		if e := RenderEvent(l, Event{Type: kind, StageID: "S03", Failure: &execution.Error{Category: execution.Canceled}}); e != nil {
			t.Fatal(e)
		}
	}
	if e := RenderEvent(l, Event{Type: ActionFailed}); e == nil {
		t.Fatal("missing category accepted")
	}
	if e := RenderEvent(l, Event{Type: "unknown"}); e == nil {
		t.Fatal("unknown event accepted")
	}
	for _, event := range []Event{
		{Type: ActionStarted, StageID: "S03"},
		{Type: ActionCompleted, StageID: "S03"},
		{Type: ActionFailed, StageID: "S03", Failure: &execution.Error{Category: execution.Exit}},
		{Type: ActionCompleted, StageID: "S03", ActionID: "db.init", Failure: &execution.Error{Category: execution.Exit}},
		{Type: ExecutionCanceled, StageID: "S03", Failure: &execution.Error{Category: execution.Exit}},
	} {
		before := out.Len() + err.Len()
		if e := RenderEvent(l, event); e == nil || out.Len()+err.Len() != before {
			t.Fatal("inconsistent event reached an operator sink")
		}
	}
}
func TestTraceAdapter(t *testing.T) {
	for _, tc := range []struct {
		outcome  string
		category execution.Category
		kind     EventType
	}{{"succeeded", "", ActionCompleted}, {"failed", execution.Exit, ActionFailed}, {"blocked", execution.Dependency, StageBlocked}} {
		t.Run(tc.outcome, func(t *testing.T) {
			trace := execution.Trace{ActionID: "a", StageID: "S01", Outcome: tc.outcome, Result: execution.Result{Started: true, ExitCode: 0, Stdout: "FAKE-combined-", Stderr: "value"}}
			if tc.category != "" {
				trace.Failure = &execution.Error{Category: tc.category}
			}
			e, err := EventFromTrace(trace)
			if err != nil || e.Type != tc.kind {
				t.Fatal(e, err)
			}
			var out bytes.Buffer
			r, _ := operatorlog.NewRedactor([]string{"FAKE-combined-value"})
			l, _ := operatorlog.New(&out, &out, nil, r)
			if err = RenderEvent(l, e); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out.String(), "FAKE-") {
				t.Fatal("cross-stream trace leak")
			}
		})
	}
	if _, e := EventFromTrace(execution.Trace{Outcome: "failed"}); e == nil {
		t.Fatal("failure without category")
	}
}
