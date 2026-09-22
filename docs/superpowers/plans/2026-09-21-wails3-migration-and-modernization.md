# HowBig: Fyne → Wails3 Migration and Go Modernization Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** Skeleton. Tasks are named and scoped with a check each; step-level detail gets filled in before a task is executed. Add items with `::`.

**Goal:** Replace the Fyne UI with a Wails3 desktop app that looks and behaves the same, then bring the Go code up to Go 1.27 idiom with tests behind it.

**Architecture:** Go keeps everything that isn't drawing: settings, country data, GeoJSON loading, Mercator projection, Pacific centering, the LRU cache, and the fit-scale/draw-order math. Those get exposed to the frontend as Wails3 services. The frontend (plain TypeScript, no framework) owns layout, the two searchable lists, the header, the area bars, and draws the countries as SVG from projected paths that Go hands over. `fogleman/gg` and every `fyne.io` import go away.

**Tech Stack:** Go 1.27, Wails v3 (installed CLI: `v3.0.0-beta.24`, pinned), WebView2 on Windows, TypeScript + Vite (Wails `vanilla-ts` template), NSIS (via the Wails3 Taskfile, as in Go-Strider).

**Spec:** None, by request. This plan and the current app's behavior are the reference.

## Global Constraints

- `go.mod` stays at `go 1.27`; modernization guidelines apply up to 1.27 (Engram `go`/`coding` memories; `modern-go-guidelines` CLI wins where installed).
- Pin the Wails3 module to the exact version matching the installed CLI. It's beta; no floating `@latest`.
- Behavior parity with v1.0.0 is the bar for Part 1. No new features.
- `settings.json` keeps its keys and defaults so existing files still load.
- The copyright string is `Copyright © Joel L. Caesar`, with no year and no "All Rights Reserved" (that phrase contradicts GPL-3.0). It goes everywhere copyright appears: `SPDX-FileCopyrightText: Copyright © Joel L. Caesar` on every Go and TS source file (existing headers get rewritten), the About dialog, `build/config.yml`, `build/windows/info.json` `LegalCopyright`, the NSIS `INFO_COPYRIGHT`, and `COPYRIGHT.md`. `SPDX-License-Identifier: GPL-3.0` stays.
- No Claude/Anthropic attribution trailers on commits.
- Work happens on an announced feature branch; tag the current `master` head as a rollback point before starting.

## Assumptions (flag any that are wrong)

1. Frontend is framework-free TypeScript. The UI is two lists, a header, two bars, a map and three buttons; React/Svelte would be dead weight.
2. Map rendering moves to SVG in the webview. Go sends projected, already-scaled paths; the browser fills and strokes them with `fill-rule="evenodd"`. This replaces `gg` rasterization.
3. `mapdata/` and `country_data.json` stay on disk, located via `settings.json`, the way they are now. Embedding them is a possible later change, not part of this plan.
4. Windows x64 is the only release target (NSIS installer). Linux/macOS builds should keep compiling but aren't packaged.
5. "Modernize" means idiom, structure and tests, not a rewrite of the geometry math.

---

## Part 0: Safety net

### Task 0.1: Rollback tag and branch
Tag `master` (annotated, pushed) as `pre-wails3`, create and announce branch `feature/wails3`.
**Check:** `git tag -l pre-wails3` exists on the remote; `git branch --show-current` is `feature/wails3`.

### Task 0.2: Characterization tests for logic that survives
There are no tests today. Before moving code, pin current behavior of the non-UI functions: `LatLonToMercator`, `NeedsPacificCentering`, `ApplyPacificCentering`, `UpdateBoundingBox`, `convertGeoJSONToDisplayFormat` (against a couple of real files, e.g. `FJI` and `USA`), `GeoCache` eviction order, `formatNumber`, `ParseHexColor`, `NewCountryCollection`.
**Check:** `go test ./...` passes on unmodified logic.

### Task 0.3: Pull the scale/order math out of the widget tree
`getFitScale` and `getScaleAndOrder` read sizes off Fyne containers and hunt for the footer by button text (three copies of that loop). Rewrite them as pure functions taking the drawable width and height, with tests. This is the one piece of logic that has to move before Fyne can go.
**Check:** New tests cover both-selected, one-selected, and the "smaller by area but wider in pixels" swap; Fyne app still runs identically using the new functions.

---

## Part 1: Wails3 migration

### Task 1.1: Scaffold Wails3 alongside the existing code
Generate a `vanilla-ts` Wails3 app in place (`frontend/`, `build/`, `Taskfile.yml`), wire `main.go` to `application.New` with a single window 1280×768, min size 1280×768, starting fullscreen, black background. Fyne code is left in place but no longer called from `main`.
**Check:** `wails3 dev` opens an empty fullscreen window; `wails3 build` produces `HowBig.exe`.

### Task 1.2: Go services
Split into services bound to the frontend:
- `CountryService`: `List() []CountryInfo`, area lookups.
- `MapService`: `Layout(left, right string, width, height float64)` returns scale, draw order and projected pixel paths for each country (plus bounding box when `debug_show_boundary`).
- `SettingsService`: returns the loaded `Settings` (colors, font sizes, flags).
- `WindowService`: `ToggleFullscreen() bool`, `Quit()`, window-title updates on resize.
Generate TS bindings.
**Check:** Unit tests on the services' Go side; bindings generated under `frontend/bindings` and importable.

### Task 1.3: Layout shell and styling
HTML/CSS grid: left list | left bar | center (header, map, footer) | right bar | right list. White 2px panel borders, background color and font sizes driven from `SettingsService`, 20px scrollbars. List width fits the longest country name.
**Check:** Side-by-side screenshot against v1.0.0 at 1280×768 and fullscreen; proportions match.

### Task 1.4: Country lists with search
Two independent lists: search box with "X" clear, "Deselect All" button, click to select, click the selected row again to deselect. ESC in the search box quits (matches current behavior).
**Check:** Manual run-through of each interaction on both lists; filter is case-insensitive substring.

### Task 1.5: Header and area bars
Header shows `Name: N sq. mi. / N km.` per selection in the side's color at `header_font_size`. Bars show area relative to the larger selection, 50px wide, 2px padding, 50% alpha, minimum 1px when area > 0.
**Check:** Russia vs. Vatican shows a full bar and a 1px bar; header numbers match v1.0.0.

### Task 1.6: SVG map rendering
Render the larger country first, then the smaller, then both borders in their border colors; center within the map area; redraw on resize (debounced). Honor `skip_small`, `enable_pacific_center`, `debug_show_boundary`.
**Check:** Fiji, Russia, USA, Kiribati and a USA vs. Brazil comparison render the same shapes and placement as v1.0.0.

### Task 1.7: Keyboard shortcuts, buttons, About
ESC quits, F toggles fullscreen (button label flips between "Windowed (F)" and "Fullscreen (F)"), A opens About. About shows on launch as a modal with the current text. Shortcuts work without first clicking the map, which the Fyne version required.
**Check:** Each shortcut works from anywhere except inside a search box, where only ESC is global.

### Task 1.8: Error paths
Missing `country_data.json` or a missing GeoJSON file shows an in-app error dialog instead of crashing. (Today a missing `country_data.json` panics on nil `CountryData`.)
**Check:** Rename `mapdata/FJI.geojson` and `country_data.json` in turn; app reports each and keeps running.

### Task 1.9: Remove Fyne
Delete the Fyne widgets, themes and wrappers, drop `fyne.io/fyne/v2` and `github.com/fogleman/gg`, `go mod tidy`.
**Check:** `go list -m all | grep -E 'fyne|fogleman'` is empty; `go vet ./...` and `go test ./...` pass; `wails3 build` succeeds.

### Task 1.10: NSIS installer, built the way Go-Strider builds it
Use the same packaging Go-Strider uses (`G:\_GoProjects\Go-Strider\build\windows\`): the Wails3-generated `build/windows/Taskfile.yml` `package` task, which runs `create:nsis:installer`. That task uses `wails3 generate webview2bootstrapper` and then `makensis build/windows/nsis/project.nsi`. Start from Go-Strider's files and make these changes:
- `build/config.yml`: `companyName: "dmongrel"`, `productName: "HowBig"`, `copyright: "Copyright © Joel L. Caesar"`, and version `1.1.0` (or whatever the release number turns out to be).
- `build/windows/info.json`: set ProductName, FileDescription, versions and `LegalCopyright: "Copyright © Joel L. Caesar"` under the `0409` language key. Go-Strider needed a fix (ac8c858) because the version block was tagged with a language code Windows skips, which left Properties > Details blank.
- `build/windows/icon.ico`, `build/appicon.png`: a HowBig icon instead of the Wails default.
- `project.nsi`: machine scope, installing to `$PROGRAMFILES64\HowBig` like Go-Strider's 392e711. It also has to install the data files the Wails template doesn't know about: `mapdata\*.geojson`, `country_data.json`, `settings.json`, `ATTRIBUTION.md`, `LICENSE.md`. The uninstaller removes them too.
- The installer is written to `bin/howbig-amd64-installer.exe`.
- Relative `map_data_path` and `country_data_path` are resolved against the exe's directory, not the working directory, so the app still finds its data when launched from a shortcut or from anywhere else.
- Delete `innosetup.iss`. Retire the `Makefile` in favour of `wails3 task …`, which is Go-Strider's single entry point.

**Check:** On a clean profile, `wails3 task package` produces `bin/howbig-amd64-installer.exe`. Installing it puts HowBig under Program Files with Start-menu and desktop shortcuts, and the app launches from both and shows maps. Properties > Details on both the exe and the installer shows `Copyright © Joel L. Caesar` and the version. Uninstalling leaves nothing behind.

### Task 1.11: Docs and copyright sweep
In the README, replace the Fyne and Inno Setup references with the Wails3 prerequisites (Node, `wails3` CLI, NSIS) and the build and package commands, and add an Install section pointing at the latest release, as Go-Strider's does. Fix the stale `LICENSE.txt` link and the `header_font_size` default mismatch (the README says 18, the code says 36). Rewrite every existing `SPDX-FileCopyrightText` line and `COPYRIGHT.md` to `Copyright © Joel L. Caesar`, and update the About text to match.
**Check:** A fresh clone builds and packages by following the README alone. `git grep -n "2026 Joel\|All Rights Reserved"` returns nothing.

---

## Part 2: Go modernization

### Task 2.1: Package layout
Move out of one `main` package: `internal/geo` (projection, centering, bounding box, GeoJSON parsing), `internal/country`, `internal/settings`, `internal/mapcache`, services in `internal/app` or next to `main`. `main.go` shrinks to wiring.
**Check:** `go build ./...` and all tests pass; no package imports `main`-level state.

### Task 2.2: Settings loading
Collapse the three copies of the defaults into one `DefaultSettings()` and apply zero-value fallbacks once. Return errors instead of only logging them.
**Check:** Tests for missing file, malformed JSON, and partial JSON each yield the documented defaults.

### Task 2.3: GeoJSON loading API
Replace the seven-argument `FetchAndCacheGeoJSON` with a loader type that owns the map path, options and cache. Drop the unused `singlePolyline=false` path if nothing needs it. Replace `os.IsNotExist` with `errors.Is(err, fs.ErrNotExist)`, `fmt.Sprint(int)` with `strconv.Itoa`.
**Check:** Existing Part 0 tests pass against the new API.

### Task 2.4: Idiom sweep
Run `modernize -fix ./...` on a clean tree, then hand-apply what it misses from the 1.20–1.27 guideline bands: remove hand-written iterator closures that only feed `slices.Collect`/`maps.Collect` over a plain range, `log` → `log/slog`, `any` over `interface{}`, typed cache (drop `container/list` type assertions or make the LRU generic only if it stays simpler). `encoding/json/v2` only in newly written code, not as an import swap.
**Check:** `modernize` reports nothing; `go vet`, `staticcheck` clean; tests pass.

### Task 2.5: `scripts/download_geojson.go`
Apply the same sweep; give it a `//go:build ignore` tag (or move to `cmd/`) so it doesn't collide with the main package's `CountryInfo`.
**Check:** `go run scripts/download_geojson.go` still fetches and writes one country correctly.

### Task 2.6: Final verification
Run the full test suite and the race detector on the cache, do a manual parity pass against the Task 1.3/1.6 screenshots, and build the installer with `wails3 task package`.
**Check:** `go test -race ./...` passes, the installer installs and launches, and the screenshots match.

---

## Resolved decisions

- Build entry point: the Wails3 Taskfile, following Go-Strider. The `Makefile` and `innosetup.iss` are removed.
- WebView2: the NSIS installer embeds the bootstrapper through `wails3 generate webview2bootstrapper`.