// Package exitcode defines the stable CLI exit codes (plan v4, section 3.3).
// The GUI and automation depend on these values; do not renumber.
package exitcode

const (
	OK              = 0  // success
	JobFailed       = 1  // job ran but a step failed
	Usage           = 2  // usage / CLI error
	Schema          = 3  // job.yaml schema, discovery or path-safety error
	EnvValidation   = 4  // required env missing or empty
	JobLocked       = 5  // job lock held, concurrent run denied
	Timeout         = 6  // timed out and tree-killed
	Cancelled       = 7  // cancelled and tree-killed
	PromoteFailed   = 8  // promote refused or failed after checks
	EnvNotAllowed   = 9  // env profile not allowed on this machine
	Internal        = 10 // unexpected internal error
	PlanMismatch    = 11 // plan_id no longer matches current state
	PromoteLockHeld = 12 // promote lock held, wait exhausted
	CancelNotFound  = 13 // cancel target not found or not running
	ScheduleDrift   = 14 // schedule reconcile drift (with --strict)
)
