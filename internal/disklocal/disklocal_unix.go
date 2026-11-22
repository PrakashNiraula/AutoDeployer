//go:build unix

package disklocal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var remoteFS = map[string]bool{
	"nfs": true, "nfs4": true, "cifs": true, "smb": true, "smbfs": true,
	"fuse.sshfs": true, "afs": true, "glusterfs": true,
}

// Check reports whether path lives on a local disk.
func Check(path string) (local bool, fstype string, err error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false, "", err
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return false, "", err
	}
	var st syscall.Statfs_t
	if err := syscall.Statfs(abs, &st); err != nil {
		return false, "", err
	}
	fstype = mountType(abs)
	if fstype == "" {
		fstype = "unknown"
	}
	if remoteFS[strings.ToLower(fstype)] {
		return false, fstype, nil
	}
	// UNC-style paths should not appear on Unix; treat as remote if they do.
	if strings.HasPrefix(abs, "//") || strings.HasPrefix(abs, `\\`) {
		return false, "unc", nil
	}
	return true, fstype, nil
}

func mountType(path string) string {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return ""
	}
	defer f.Close()
	bestLen := -1
	best := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		mp, typ := fields[1], fields[2]
		mp = decodeMount(mp)
		if strings.HasPrefix(path, mp) && len(mp) > bestLen {
			bestLen = len(mp)
			best = typ
		}
	}
	return best
}

func decodeMount(s string) string {
	out := s
	out = strings.ReplaceAll(out, `\040`, " ")
	out = strings.ReplaceAll(out, `\011`, "\t")
	return out
}

// EnsureLocal returns an error if path is not local.
func EnsureLocal(path string) error {
	ok, fstype, err := Check(path)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("data/ appears to be on remote filesystem %q; must be local disk", fstype)
	}
	return nil
}
