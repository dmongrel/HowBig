// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathFrom(t *testing.T) {
	root := t.TempDir()
	exeDir := filepath.Join(root, "bin")
	cwd := t.TempDir()
	for _, dir := range []string{exeDir, cwd} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(path string) {
		t.Helper()
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(exeDir, "both.json"))
	write(filepath.Join(cwd, "both.json"))
	write(filepath.Join(cwd, "cwd.json"))
	write(filepath.Join(root, "parent.json"))
	t.Chdir(cwd)

	tests := []struct {
		name, in, want string
	}{
		{"exe dir wins over cwd", "both.json", filepath.Join(exeDir, "both.json")},
		{"cwd", "cwd.json", "cwd.json"},
		{"exe dir's parent", "parent.json", filepath.Join(root, "parent.json")},
		{"not found anywhere", "none.json", "none.json"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvePathFrom(tt.in, exeDir); got != tt.want {
				t.Errorf("resolvePathFrom(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
	abs := filepath.Join(root, "parent.json")
	if got := ResolvePath(abs); got != abs {
		t.Errorf("ResolvePath(abs) = %q, want it unchanged", got)
	}
}
