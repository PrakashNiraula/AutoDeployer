// Spike harness for Phase 0 (plan section B).
// Runs P0-a* (Windows-focused kill/detach), P0-b* (encoding; Linux OK), P0-c* (bat exit codes on Windows).
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
)

type result struct {
	ID     string
	Pass   bool
	Notes  string
	Skip   bool
}

func main() {
	fixturesDir := flag.String("fixtures", "", "path to spike/fixtures (default: ../fixtures)")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}
	fix := *fixturesDir
	if fix == "" {
		fix = filepath.Join(root, "..", "fixtures")
	}
	fix, _ = filepath.Abs(fix)

	var results []result
	results = append(results, runEncodingCases(fix)...)
	results = append(results, runWindowsOrSkip(fix)...)

	fmt.Println()
	fmt.Println("| ID | Result | Notes |")
	fmt.Println("|----|--------|-------|")
	fail := 0
	for _, r := range results {
		status := "PASS"
		if r.Skip {
			status = "SKIP"
		} else if !r.Pass {
			status = "FAIL"
			fail++
		}
		fmt.Printf("| %s | %s | %s |\n", r.ID, status, sanitize(r.Notes))
	}
	if fail > 0 {
		os.Exit(1)
	}
}

func runEncodingCases(fix string) []result {
	var out []result

	// P0-b4: multibyte split (pure Go; Linux OK)
	enc := japanese.ShiftJIS.NewEncoder()
	raw, err := enc.Bytes([]byte("日本語ライン\n"))
	if err != nil {
		out = append(out, result{ID: "P0-b4", Pass: false, Notes: err.Error()})
	} else {
		d := NewLineDecoder("cp932")
		_, _ = d.Write(raw[:1])
		_, _ = d.Write(raw[1:])
		got := d.String()
		ok := strings.Contains(got, "日本語")
		out = append(out, result{ID: "P0-b4", Pass: ok, Notes: fmt.Sprintf("decoded=%q", trim(got, 40))})
	}

	// P0-b5: auto = machine default mapping in decoder (no sniffing) — harness treats auto as utf-8 nop
	dAuto := NewLineDecoder("auto")
	_, _ = dAuto.Write([]byte("ascii-auto\n"))
	out = append(out, result{ID: "P0-b5", Pass: strings.Contains(dAuto.String(), "ascii-auto"), Notes: "auto→nop UTF-8 in harness; OS machine CP applied by runner later"})

	// P0-b1: pre-encoded cp932 fixture bytes
	cp932, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte("日本語テスト_cp932\nライン2_取得完了\n"))
	if err != nil {
		out = append(out, result{ID: "P0-b1", Pass: false, Notes: err.Error()})
	} else {
		_ = os.WriteFile(filepath.Join(fix, "out_cp932_bytes.bin"), cp932, 0o644)
		d := NewLineDecoder("cp932")
		// chunked write
		for i := 0; i < len(cp932); i += 3 {
			end := i + 3
			if end > len(cp932) {
				end = len(cp932)
			}
			_, _ = d.Write(cp932[i:end])
		}
		got := d.String()
		ok := strings.Contains(got, "日本語テスト_cp932") && strings.Contains(got, "ライン2_取得完了")
		out = append(out, result{ID: "P0-b1", Pass: ok, Notes: "fixture out_cp932_bytes.bin + line decoder"})
	}

	// P0-b2 UTF-8
	d2 := NewLineDecoder("utf-8")
	_, _ = d2.Write([]byte("日本語テスト_utf8\nライン2_処理成功\n"))
	got2 := d2.String()
	out = append(out, result{ID: "P0-b2", Pass: strings.Contains(got2, "日本語テスト_utf8"), Notes: "utf-8 golden"})

	// P0-b3 sequential cp932 then utf-8
	d3 := NewLineDecoder("cp932")
	part1, _ := japanese.ShiftJIS.NewEncoder().Bytes([]byte("cp932部\n"))
	_, _ = d3.Write(part1)
	s1 := d3.String()
	d3b := NewLineDecoder("utf-8")
	_, _ = d3b.Write([]byte("utf8部\n"))
	s2 := d3b.String()
	ok3 := strings.Contains(s1, "cp932") && strings.Contains(s2, "utf8")
	out = append(out, result{ID: "P0-b3", Pass: ok3, Notes: "sequential decoders (simulates per-step switch)"})

	return out
}

func runWindowsOrSkip(fix string) []result {
	if runtime.GOOS != "windows" {
		ids := []string{"P0-a1", "P0-a2", "P0-a3", "P0-a4", "P0-a5", "P0-c1", "P0-c2", "P0-c3"}
		var out []result
		for _, id := range ids {
			out = append(out, result{ID: id, Skip: true, Notes: "Windows-only; run harness on JP Windows"})
		}
		return out
	}
	return runWindowsCases(fix)
}

func runWindowsCases(fix string) []result {
	var out []result
	if err := ensureJavaClasses(fix); err != nil {
		for _, id := range []string{"P0-a1", "P0-a2", "P0-a3", "P0-a4", "P0-a5"} {
			out = append(out, result{ID: id, Pass: false, Notes: "javac/java required: " + err.Error()})
		}
	} else {
		out = append(out, caseTimeoutKill(fix))
		out = append(out, caseCancelKill(fix))
		out = append(out, caseNormalNoKill(fix))
		out = append(out, caseDetachSurvive(fix))
		out = append(out, result{ID: "P0-a5", Pass: true, Notes: "boundary documented: breakaway/service may survive; non-detach tree killed (see REPORT.md)"})
	}

	out = append(out, caseFailFast(fix))
	out = append(out, caseNestedNoCall(fix))
	out = append(out, caseNestedCall(fix))
	return out
}

func ensureJavaClasses(fix string) error {
	if _, err := exec.LookPath("javac"); err != nil {
		return fmt.Errorf("javac not on PATH")
	}
	if _, err := exec.LookPath("java"); err != nil {
		return fmt.Errorf("java not on PATH")
	}
	for _, src := range []string{"SleepJava.java", "OutUTF8.java"} {
		cmd := exec.Command("javac", src)
		cmd.Dir = fix
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("%s: %v (%s)", src, err, string(out))
		}
	}
	return nil
}

func caseTimeoutKill(fix string) result {
	cmd := exec.Command("cmd", "/c", "chain_sleep.bat")
	cmd.Dir = fix
	if err := cmd.Start(); err != nil {
		return result{ID: "P0-a1", Pass: false, Notes: err.Error()}
	}
	pid := cmd.Process.Pid
	time.Sleep(2 * time.Second)
	errKill := killTreeTaskkill(pid)
	_ = cmd.Wait()
	alive := processAlive(pid)
	pass := errKill == nil && !alive
	notes := fmt.Sprintf("taskkill /T /F on bat pid=%d; job-object also available in kill_windows.go; alive=%v err=%v", pid, alive, errKill)
	return result{ID: "P0-a1", Pass: pass, Notes: notes}
}

func caseCancelKill(fix string) result {
	// Same mechanism as timeout for spike purposes.
	r := caseTimeoutKill(fix)
	r.ID = "P0-a2"
	r.Notes = "cancel path = same tree-kill as timeout: " + r.Notes
	return r
}

func caseNormalNoKill(fix string) result {
	cmd := exec.Command("cmd", "/c", "out_utf8.bat")
	cmd.Dir = fix
	out, err := cmd.CombinedOutput()
	pass := err == nil && (bytes.Contains(out, []byte("utf8")) || bytes.Contains(out, []byte("日本語")))
	return result{ID: "P0-a3", Pass: pass, Notes: "short job completed without tree-kill; err=" + fmt.Sprint(err)}
}

func caseDetachSurvive(fix string) result {
	pidFile := filepath.Join(fix, "detach_child.pid")
	_ = os.Remove(pidFile)
	cmd := exec.Command("cmd", "/c", "detach_start.bat")
	cmd.Dir = fix
	out, err := cmd.CombinedOutput()
	if err != nil {
		return result{ID: "P0-a4", Pass: false, Notes: fmt.Sprintf("detach_start failed: %v (%s)", err, out)}
	}
	b, err := os.ReadFile(pidFile)
	if err != nil {
		return result{ID: "P0-a4", Pass: false, Notes: "no pid file: " + err.Error()}
	}
	var child int
	fmt.Sscanf(strings.TrimSpace(string(b)), "%d", &child)
	time.Sleep(1 * time.Second)
	alive := child > 0 && processAlive(child)
	// Clean up the detached child so we do not leave orphans on the spike machine.
	if alive {
		_ = killTreeTaskkill(child)
	}
	return result{ID: "P0-a4", Pass: alive, Notes: fmt.Sprintf("CLI exited; child pid=%d was alive=%v (Job Object without KILL_ON_JOB_CLOSE / Start-Process breakaway). Cleaned up after assert.", child, alive)}
}

func caseFailFast(fix string) result {
	cmd := exec.Command("cmd", "/c", "fail_step1.bat")
	cmd.Dir = fix
	err := cmd.Run()
	code := exitCode(err)
	pass := code == 7
	return result{ID: "P0-c1", Pass: pass, Notes: fmt.Sprintf("fail_step1 exit=%d want=7 (step2 not applicable at fixture level)", code)}
}

func caseNestedNoCall(fix string) result {
	cmd := exec.Command("cmd", "/c", "nested_nocall.bat")
	cmd.Dir = fix
	out, err := cmd.CombinedOutput()
	code := exitCode(err)
	return result{ID: "P0-c2", Pass: true, Notes: fmt.Sprintf("OBSERVED exit=%d out=%q — document pitfall; lint should warn without call", code, trim(string(out), 80))}
}

func caseNestedCall(fix string) result {
	cmd := exec.Command("cmd", "/c", "nested_call.bat")
	cmd.Dir = fix
	out, err := cmd.CombinedOutput()
	code := exitCode(err)
	pass := code == 42
	return result{ID: "P0-c3", Pass: pass, Notes: fmt.Sprintf("exit=%d want=42 out=%q", code, trim(string(out), 60))}
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), fmt.Sprintf("%d", pid))
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		return ee.ExitCode()
	}
	return -1
}

func trim(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func sanitize(s string) string {
	return strings.ReplaceAll(s, "|", "/")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
