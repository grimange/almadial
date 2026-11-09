// Package operatorui provides text-only preparation and observability, never execution.
package operatorui

import (
	"errors"
	"regexp"

	"github.com/grimange/almadial/internal/execution"
	"github.com/grimange/almadial/internal/operatorlog"
)

type EventType string

const (
	StagePending           EventType = "stage_pending"
	StageStarted           EventType = "stage_started"
	ActionStarted          EventType = "action_started"
	ActionCompleted        EventType = "action_completed"
	ActionFailed           EventType = "action_failed"
	StageCompleted         EventType = "stage_completed"
	StageBlocked           EventType = "stage_blocked"
	ExecutionCanceled      EventType = "execution_canceled"
	ReconciliationRequired EventType = "reconciliation_required"
)

type Event struct {
	Type              EventType
	StageID, ActionID string
	Failure           *execution.Error
	Diagnostics       []string
}

func next(category execution.Category) string {
	switch category {
	case execution.Contract:
		return "Correct the action contract and regenerate the plan."
	case execution.Precondition:
		return "Resolve required eligibility checks; review a fresh plan."
	case execution.Start:
		return "Check the executable and its execution permissions."
	case execution.Exit:
		return "Inspect redacted diagnostics and reconcile state before retry."
	case execution.Timeout:
		return "Check bounded diagnostics and reconcile the interrupted action."
	case execution.Canceled:
		return "Execution stopped; inspect state before any future resume."
	case execution.OutputLimit:
		return "Review diagnostic limits and reconcile the action."
	case execution.Postcondition:
		return "Investigate the failed result assertion; do not assume success."
	case execution.Dependency:
		return "Resolve the failed predecessor before continuing."
	default:
		return "Stop and request review of the action and current state."
	}
}

// RenderEvent consumes one event in caller order. It performs no actions and
// retains no diagnostics. The same redaction/bounds apply to every output sink.
var eventID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)

func RenderEvent(l *operatorlog.Logger, e Event) error {
	if l == nil || !eventID.MatchString(e.StageID) || (e.ActionID != "" && !eventID.MatchString(e.ActionID)) {
		return errors.New("progress: logger and valid stage/action identifiers required")
	}
	level := operatorlog.Info
	switch e.Type {
	case StagePending, StageStarted, ActionStarted, ActionCompleted, StageCompleted:
	case ActionFailed, StageBlocked, ReconciliationRequired, ExecutionCanceled:
		level = operatorlog.Error
	default:
		return errors.New("progress: unknown event type")
	}
	if (e.Type == ActionStarted || e.Type == ActionCompleted || e.Type == ActionFailed) && e.ActionID == "" {
		return errors.New("progress: action identifier required")
	}
	if level == operatorlog.Info && e.Failure != nil {
		return errors.New("progress: non-failure event cannot report a failure")
	}
	code := "none"
	guidance := "Observe the next reported event."
	if e.Failure != nil {
		switch e.Failure.Category {
		case execution.Contract, execution.Precondition, execution.Start, execution.Exit, execution.Timeout, execution.Canceled, execution.OutputLimit, execution.Postcondition, execution.Dependency, execution.Internal:
		default:
			return errors.New("progress: unknown failure category")
		}
		code = string(e.Failure.Category)
		guidance = next(e.Failure.Category)
		if e.Type == ExecutionCanceled && e.Failure.Category != execution.Canceled {
			return errors.New("progress: canceled event requires cancellation category")
		}
	} else if level == operatorlog.Error {
		return errors.New("progress: failure category required")
	}
	rec := operatorlog.Record{Level: level, Stage: e.StageID, Action: e.ActionID, Code: code, Message: string(e.Type), Next: guidance}
	if len(e.Diagnostics) > 0 {
		rec.Message += " diagnostics: "
	}
	// Do not render arbitrary error.Error() or caller-supplied Detail as trusted text.
	return l.Write(rec, e.Diagnostics...)
}

// EventFromTrace adapts actual ALM-13 results without inventing stage completion.
func EventFromTrace(t execution.Trace) (Event, error) {
	e := Event{StageID: t.StageID, ActionID: t.ActionID, Failure: t.Failure, Diagnostics: []string{t.Result.Stdout, t.Result.Stderr}}
	switch t.Outcome {
	case "succeeded":
		if t.Failure != nil {
			return Event{}, errors.New("progress: inconsistent trace")
		}
		e.Type = ActionCompleted
	case "failed":
		e.Type = ActionFailed
	case "blocked":
		e.Type = StageBlocked
	default:
		return Event{}, errors.New("progress: unknown trace outcome")
	}
	if t.Outcome != "succeeded" && t.Failure == nil {
		return Event{}, errors.New("progress: missing trace failure")
	}
	return e, nil
}
