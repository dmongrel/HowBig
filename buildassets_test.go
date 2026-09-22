// SPDX-FileCopyrightText: Copyright © Joel L. Caesar
// SPDX-License-Identifier: GPL-3.0

package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// TestBuildAssetVersions checks that build/windows/info.json and every other
// build file that carries a version agree with Version, and that info.json
// still uses the 0409 (US English) string table that Explorer shows rather
// than the neutral 0000 one a build-asset regeneration writes.
func TestBuildAssetVersions(t *testing.T) {
	raw, err := os.ReadFile("build/windows/info.json")
	if err != nil {
		t.Fatal(err)
	}
	var info struct {
		Fixed map[string]string            `json:"fixed"`
		Info  map[string]map[string]string `json:"info"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		t.Fatalf("parsing info.json: %v", err)
	}

	for key := range info.Info {
		if key != "0409" {
			t.Errorf("info.json has string table %q; want only 0409", key)
		}
	}
	strs, ok := info.Info["0409"]
	if !ok {
		t.Fatal("info.json has no 0409 string table")
	}
	for _, field := range []string{"ProductVersion", "FileVersion"} {
		if got := strs[field]; got != Version {
			t.Errorf("info.json 0409 %s = %q, want %q", field, got, Version)
		}
	}
	for _, field := range []string{"file_version", "product_version"} {
		if got := info.Fixed[field]; got != Version {
			t.Errorf("info.json fixed %s = %q, want %q", field, got, Version)
		}
	}

	// The other files that carry the version, each with the text it must contain.
	for _, c := range []struct{ path, want string }{
		{"build/config.yml", `version: "` + Version + `"`},
		{"build/windows/nsis/wails_tools.nsh", `!define INFO_PRODUCTVERSION "` + Version + `"`},
		{"build/windows/wails.exe.manifest", `name="com.dmongrel.howbig" version="` + Version + `"`},
		{"build/windows/msix/app_manifest.xml", `Version="` + Version + `.0"`},
		{"build/darwin/Info.plist", `<string>` + Version + `</string>`},
		{"build/darwin/Info.dev.plist", `<string>` + Version + `</string>`},
		{"build/linux/nfpm/nfpm.yaml", `version: "` + Version + `"`},
	} {
		data, err := os.ReadFile(c.path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), c.want) {
			t.Errorf("%s does not contain %s", c.path, c.want)
		}
	}
}
