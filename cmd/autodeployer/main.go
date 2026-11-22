package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/job"
	"github.com/PrakashNiraula/autodeployer/internal/machine"
	"github.com/PrakashNiraula/autodeployer/internal/root"
	"github.com/PrakashNiraula/autodeployer/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	rootFlag, jsonOut, rest := parseGlobals(args)
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "usage: autodeployer [--root PATH] [--json] <command>")
		return exitcode.Usage
	}
	switch rest[0] {
	case "version":
		if jsonOut {
			_ = printJSON(map[string]any{
				"version":        version.Binary,
				"schema_version": version.JobSchema,
				"contract":       "v4",
			})
		} else {
			fmt.Printf("autodeployer %s (job schema %d)\n", version.Binary, version.JobSchema)
		}
		return exitcode.OK
	case "jobs":
		return cmdJobs(rootFlag, jsonOut, rest[1:])
	case "schema":
		return cmdSchema(rootFlag, jsonOut, rest[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", rest[0])
		return exitcode.Usage
	}
}

func parseGlobals(args []string) (rootFlag string, jsonOut bool, rest []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--root" && i+1 < len(args):
			rootFlag = args[i+1]
			i++
		case strings.HasPrefix(a, "--root="):
			rootFlag = strings.TrimPrefix(a, "--root=")
		case a == "--json":
			jsonOut = true
		default:
			rest = append(rest, a)
		}
	}
	return rootFlag, jsonOut, rest
}

func openRoot(rootFlag string) (string, error) {
	return root.Resolve(rootFlag)
}

func cmdJobs(rootFlag string, jsonOut bool, args []string) int {
	if len(args) == 0 {
		return usage("jobs list|show <id>")
	}
	r, err := openRoot(rootFlag)
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	jobsDir := root.JobsDir(r)
	switch args[0] {
	case "list":
		list, err := job.Discover(jobsDir)
		if err != nil {
			return fail(exitcode.Schema, "%s", err)
		}
		if jsonOut {
			_ = printJSON(list)
		} else {
			for _, s := range list {
				fmt.Printf("%s\t%s\n", s.ID, s.Name)
			}
		}
		return exitcode.OK
	case "show":
		if len(args) < 2 {
			return usage("jobs show <id>")
		}
		m, err := job.Find(jobsDir, args[1])
		if err != nil {
			return fail(exitcode.Schema, "%s", err)
		}
		readme := filepath.Join(m.Dir, "README.md")
		out := map[string]any{
			"id":           m.ID,
			"name":         m.Name,
			"description":  m.Description,
			"path":         m.Dir,
			"readme_path":  readme,
			"workdir":      m.Workdir,
			"required_env": m.RequiredEnv,
			"secrets":      m.Secrets,
			"steps":        m.Steps,
			"timeout_minutes": m.TimeoutMinutes,
		}
		if jsonOut {
			_ = printJSON(out)
		} else {
			fmt.Printf("%s — %s\n%s\n", m.ID, m.Name, strings.TrimSpace(m.Description))
		}
		return exitcode.OK
	default:
		return usage("jobs list|show <id>")
	}
}

func cmdSchema(rootFlag string, jsonOut bool, args []string) int {
	if len(args) == 0 || args[0] != "validate" {
		return usage("schema validate [job|all]")
	}
	r, err := openRoot(rootFlag)
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	// ensure machine loads (defaults ok)
	if _, err := machine.Load(root.DataDir(r)); err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	jobsDir := root.JobsDir(r)
	target := "all"
	if len(args) > 1 {
		target = args[1]
	}
	var results []map[string]any
	var allOK = true
	validateOne := func(id string) {
		m, err := job.Find(jobsDir, id)
		entry := map[string]any{"job_id": id}
		if err != nil {
			allOK = false
			entry["ok"] = false
			entry["errors"] = []string{err.Error()}
			results = append(results, entry)
			return
		}
		vr := m.Validate()
		entry["ok"] = vr.OK
		entry["errors"] = vr.Errors
		entry["warnings"] = vr.Warnings
		if !vr.OK {
			allOK = false
		}
		results = append(results, entry)
	}
	if target == "all" {
		list, err := job.Discover(jobsDir)
		if err != nil {
			return fail(exitcode.Schema, "%s", err)
		}
		for _, s := range list {
			validateOne(s.ID)
		}
	} else {
		validateOne(target)
	}
	payload := map[string]any{"ok": allOK, "results": results}
	if jsonOut {
		_ = printJSON(payload)
	} else {
		for _, res := range results {
			id := res["job_id"]
			if res["ok"] == true {
				fmt.Printf("%s: OK\n", id)
			} else {
				fmt.Printf("%s: FAIL\n", id)
				for _, e := range res["errors"].([]string) {
					fmt.Printf("  - %s\n", e)
				}
			}
		}
	}
	if !allOK {
		return exitcode.Schema
	}
	return exitcode.OK
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func usage(msg string) int {
	fmt.Fprintf(os.Stderr, "usage: autodeployer %s\n", msg)
	return exitcode.Usage
}

func fail(code int, format string, args ...any) int {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	return code
}
