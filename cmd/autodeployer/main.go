// Command autodeployer is the headless core CLI. Commands are added phase by phase;
// see docs/TASKS.md.
package main

import (
	"fmt"
	"os"

	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: autodeployer <command> [flags]")
		return exitcode.Usage
	}
	switch args[0] {
	case "version":
		fmt.Printf("autodeployer %s (job schema %d)\n", version.Binary, version.JobSchema)
		return exitcode.OK
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", args[0])
		return exitcode.Usage
	}
}
