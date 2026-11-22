package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PrakashNiraula/autodeployer/internal/envfile"
	"github.com/PrakashNiraula/autodeployer/internal/exitcode"
	"github.com/PrakashNiraula/autodeployer/internal/job"
	"github.com/PrakashNiraula/autodeployer/internal/machine"
	"github.com/PrakashNiraula/autodeployer/internal/mask"
	"github.com/PrakashNiraula/autodeployer/internal/plan"
	"github.com/PrakashNiraula/autodeployer/internal/root"
	"github.com/PrakashNiraula/autodeployer/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	g, rest := parseGlobals(args)
	if len(rest) == 0 {
		fmt.Fprintln(os.Stderr, "usage: autodeployer [--root PATH] [--json] <command>")
		return exitcode.Usage
	}
	switch rest[0] {
	case "version":
		if g.JSON {
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
		return cmdJobs(g, rest[1:])
	case "schema":
		return cmdSchema(g, rest[1:])
	case "plan":
		return cmdPlan(g, rest[1:])
	case "env":
		return cmdEnv(g, rest[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", rest[0])
		return exitcode.Usage
	}
}

type globals struct {
	Root         string
	JSON         bool
	Reveal       bool
	AllowExample bool
}

func parseGlobals(args []string) (globals, []string) {
	var g globals
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--root" && i+1 < len(args):
			g.Root = args[i+1]
			i++
		case strings.HasPrefix(a, "--root="):
			g.Root = strings.TrimPrefix(a, "--root=")
		case a == "--json":
			g.JSON = true
		case a == "--reveal":
			g.Reveal = true
		case a == "--allow-example-env":
			g.AllowExample = true
		default:
			rest = append(rest, a)
		}
	}
	return g, rest
}

func openRoot(rootFlag string) (string, error) {
	return root.Resolve(rootFlag)
}

func cmdJobs(g globals, args []string) int {
	if len(args) == 0 {
		return usage("jobs list|show <id>")
	}
	r, err := openRoot(g.Root)
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
		if g.JSON {
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
			"id":              m.ID,
			"name":            m.Name,
			"description":     m.Description,
			"path":            m.Dir,
			"readme_path":     readme,
			"workdir":         m.Workdir,
			"required_env":    m.RequiredEnv,
			"secrets":         m.Secrets,
			"steps":           m.Steps,
			"timeout_minutes": m.TimeoutMinutes,
		}
		if g.JSON {
			_ = printJSON(out)
		} else {
			fmt.Printf("%s — %s\n%s\n", m.ID, m.Name, strings.TrimSpace(m.Description))
		}
		return exitcode.OK
	default:
		return usage("jobs list|show <id>")
	}
}

func cmdSchema(g globals, args []string) int {
	if len(args) == 0 || args[0] != "validate" {
		return usage("schema validate [job|all]")
	}
	r, err := openRoot(g.Root)
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
	if g.JSON {
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

func cmdPlan(g globals, args []string) int {
	jobID, envProfile, err := parseJobEnv(args)
	if err != nil {
		return usage("plan <job> --env <profile>")
	}
	r, err := openRoot(g.Root)
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	cfg, err := machine.Load(root.DataDir(r))
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	m, err := job.Find(root.JobsDir(r), jobID)
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	res, err := plan.Build(r, m, envProfile, cfg, g.AllowExample, nil)
	if err != nil {
		if err.Error() == "ENV_NOT_ALLOWED" {
			return fail(exitcode.EnvNotAllowed, "environment %q is not allowed on this machine", envProfile)
		}
		return fail(exitcode.Internal, "%s", err)
	}
	if g.JSON {
		_ = printJSON(res)
	} else {
		fmt.Printf("plan_id=%s can_execute=%v\n", res.PlanID, res.CanExecute)
	}
	if !res.EnvValidation.OK {
		return exitcode.EnvValidation
	}
	return exitcode.OK
}

func cmdEnv(g globals, args []string) int {
	if len(args) == 0 {
		return usage("env list|get|set --job <id> --env <profile> ...")
	}
	r, err := openRoot(g.Root)
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	cfg, err := machine.Load(root.DataDir(r))
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	flags := parseKV(args[1:])
	jobID := flags["job"]
	profile := flags["env"]
	if jobID == "" {
		return usage("env ... --job <id>")
	}
	m, err := job.Find(root.JobsDir(r), jobID)
	if err != nil {
		return fail(exitcode.Schema, "%s", err)
	}
	switch args[0] {
	case "list":
		if profile == "" {
			profile = cfg.DefaultEnv
		}
		if !cfg.Allows(profile) {
			return fail(exitcode.EnvNotAllowed, "environment %q is not allowed on this machine", profile)
		}
		p := envfile.Path(root.DataDir(r), jobID, profile)
		vals, err := envfile.ParseFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				vals = map[string]string{}
			} else {
				return fail(exitcode.Internal, "%s", err)
			}
		}
		out := vals
		if !g.Reveal {
			out = mask.MaskMap(vals, m.Secrets, cfg.SecretMinMaskLength)
		}
		if g.JSON {
			_ = printJSON(out)
		} else {
			for k, v := range out {
				fmt.Printf("%s=%s\n", k, v)
			}
		}
		return exitcode.OK
	case "get":
		key := flags["key"]
		if profile == "" || key == "" {
			return usage("env get --job <id> --env <p> --key <K>")
		}
		if !cfg.Allows(profile) {
			return fail(exitcode.EnvNotAllowed, "environment %q is not allowed on this machine", profile)
		}
		vals, err := envfile.ParseFile(envfile.Path(root.DataDir(r), jobID, profile))
		if err != nil {
			return fail(exitcode.EnvValidation, "%s", err)
		}
		v := vals[key]
		if !g.Reveal && mask.IsSecretKey(key, m.Secrets) && mask.ShouldMaskValue(v, cfg.SecretMinMaskLength) {
			v = "***"
		}
		if g.JSON {
			_ = printJSON(map[string]string{key: v})
		} else {
			fmt.Println(v)
		}
		return exitcode.OK
	case "set":
		key, value := flags["key"], flags["value"]
		if profile == "" || key == "" {
			return usage("env set --job <id> --env <p> --key <K> --value <V>")
		}
		if !cfg.Allows(profile) {
			return fail(exitcode.EnvNotAllowed, "environment %q is not allowed on this machine", profile)
		}
		if err := envfile.SetKey(root.DataDir(r), jobID, profile, key, value); err != nil {
			return fail(exitcode.Internal, "%s", err)
		}
		// audit wired in 1c
		if g.JSON {
			_ = printJSON(map[string]string{"status": "ok", "key": key})
		} else {
			fmt.Println("ok")
		}
		return exitcode.OK
	default:
		return usage("env list|get|set ...")
	}
}

func parseJobEnv(args []string) (jobID, env string, err error) {
	if len(args) == 0 {
		return "", "", fmt.Errorf("missing job")
	}
	jobID = args[0]
	kv := parseKV(args[1:])
	env = kv["env"]
	if env == "" {
		return "", "", fmt.Errorf("missing --env")
	}
	return jobID, env, nil
}

func parseKV(args []string) map[string]string {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			out[strings.TrimPrefix(a, "--")] = args[i+1]
			i++
		} else if strings.HasPrefix(a, "--") && strings.Contains(a, "=") {
			k, v, _ := strings.Cut(strings.TrimPrefix(a, "--"), "=")
			out[k] = v
		}
	}
	return out
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
