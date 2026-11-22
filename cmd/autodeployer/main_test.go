package main

import (
	"testing"

	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
)

func TestRunVersion(t *testing.T) {
	if got := run([]string{"version"}); got != exitcode.OK {
		t.Fatalf("version exit = %d, want %d", got, exitcode.OK)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if got := run([]string{"nope"}); got != exitcode.Usage {
		t.Fatalf("unknown command exit = %d, want %d", got, exitcode.Usage)
	}
}

func TestRunNoArgs(t *testing.T) {
	if got := run(nil); got != exitcode.Usage {
		t.Fatalf("no args exit = %d, want %d", got, exitcode.Usage)
	}
}
