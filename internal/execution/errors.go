// Package execution supplies bounded local primitives, not installation authorization.
package execution

import "fmt"

// Category is a stable failure code. Success is represented by a nil error.
type Category string

const (
	Contract      Category = "contract"
	Precondition  Category = "precondition"
	Start         Category = "process_start"
	Exit          Category = "nonzero_exit"
	Timeout       Category = "timeout"
	Canceled      Category = "canceled"
	OutputLimit   Category = "output_limit"
	Postcondition Category = "postcondition"
	Dependency    Category = "dependency"
	Internal      Category = "internal"
)

// Error deliberately excludes raw subprocess and assertion errors, argv and paths.
// Only safe context causes are unwrapped. Match a category with errors.Is(err,
// &Error{Category: Timeout}); use errors.As to inspect action/stage/exit status.
type Error struct {
	Category     Category `json:"category"`
	ActionID     string   `json:"action_id"`
	StageID      string   `json:"stage_id"`
	DependencyID string   `json:"dependency_id,omitempty"`
	AssertionID  string   `json:"assertion_id,omitempty"`
	Detail       string   `json:"detail"`
	ExitCode     int      `json:"exit_code"` // -1 when unavailable; signal termination also has no numeric exit code
	cause        error
}

func (e *Error) Error() string {
	return fmt.Sprintf("stage %s action %s: %s (exit %d)", e.StageID, e.ActionID, e.Category, e.ExitCode) + ": " + e.Detail
}
func (e *Error) Unwrap() error { return e.cause }
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t.Category == e.Category
}
func failure(k Category) *Error {
	descriptions := map[Category]string{
		Contract:      "invalid action, graph or command contract; review required fields and bindings",
		Precondition:  "required eligibility assertion failed; command was not started",
		Start:         "executable could not be started; check its existence and execution permissions",
		Exit:          "process exited unsuccessfully; inspect bounded diagnostics",
		Timeout:       "execution deadline exceeded; process group termination requested",
		Canceled:      "execution canceled; no further commands will be started",
		OutputLimit:   "diagnostic limit exceeded; review limits and bounded output",
		Postcondition: "process completed but a required result assertion failed",
		Dependency:    "required predecessor did not succeed; action was blocked",
		Internal:      "unexpected execution result; action cannot be considered verified",
	}
	return &Error{Category: k, ExitCode: -1, Detail: descriptions[k]}
}

// All detail arguments are static package-owned guidance, never caller values.
func contract(detail string) *Error { e := failure(Contract); e.Detail = detail; return e }
