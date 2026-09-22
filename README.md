
# HowBig

HowBig is a desktop app for comparing the sizes of countries. Pick one country on each side and it draws both at the same scale, one over the other, so you can see how they stack up. It's written in Go, with a [Wails v3](https://v3.wails.io/) window and a TypeScript frontend.

<img width="1200" height="502" alt="image" src="https://github.com/user-attachments/assets/bf6f87b2-0f8e-499d-9927-5bb6055e54b5" />

## Features

- **Side-by-side comparison:** select two countries to compare their shapes and relative sizes.
- **Accurate rendering:** country boundaries are drawn from GeoJSON data in a Mercator projection.
- **Area statistics:** shows each selected country's area in square miles and square kilometres, with a bar for each.
- **Responsive UI:** both countries are scaled to fit the window and redrawn when it resizes.
- **Caching:** parsed GeoJSON is cached, so switching between recent countries is quick.

Keyboard shortcuts: `ESC` exits, `F` toggles fullscreen, `A` shows the About box.

## Install

Download `howbig-amd64-installer.exe` from the [latest release](https://github.com/dmongrel/HowBig/releases/latest) and run it. It installs HowBig to `C:\Program Files\HowBig` and adds Start-menu and desktop shortcuts. Windows x64 only.

A screen of at least 1280 x 768 is required.

## Building from source

### Prerequisites

- **Go 1.27**
- **Node.js** with npm
- **Wails v3 CLI** `v3.0.0-beta.24`, the version `go.mod` pins:
  ```
  go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.24
  ```
- **NSIS 3**, only for building the installer. `makensis` must be on your PATH; the NSIS installer puts it in `C:\Program Files (x86)\NSIS`.

### Commands

Run these from the repository root.

```
wails3 dev            # run with hot reload
wails3 build          # build bin/HowBig.exe
wails3 task package   # build bin/howbig-amd64-installer.exe
go test ./...         # run the Go tests
```

`wails3 build` installs the frontend's npm packages, generates the TypeScript bindings, builds the frontend and then the exe. `wails3 task package` does the same build and then wraps the exe, `mapdata\`, `country_data.json`, `settings.json`, `ATTRIBUTION.md` and `LICENSE.md` into an NSIS installer. The version, product name and copyright shown in the exe's and the installer's Properties come from `build/config.yml` and `build/windows/info.json`.

`bin/HowBig.exe` finds its data in the repository root, so it runs straight from the build without being installed.

## Settings

HowBig reads `settings.json` at startup. Relative paths, for `settings.json` itself and for `map_data_path` and `country_data_path`, are looked up next to the exe first, then in the working directory, then in the exe folder's parent. An installed copy uses the `settings.json` in its install folder, and a build in `bin/` uses the one in the repository root.

If the file is missing or isn't valid JSON, every setting takes the default below. A setting left out of an otherwise valid file also takes its default, except `enable_pacific_center`, which is then off.

| Setting                   | Description                                                                          | Default               |
|:--------------------------|:-------------------------------------------------------------------------------------|:----------------------|
| `debug_show_boundary`     | If `true`, draws a red bounding box around each rendered country for debugging.      | `false`               |
| `left_color`              | Hex color (`#RRGGBB`) for the left country's fill, header line and bar.              | `#00FF00`             |
| `right_color`             | Hex color (`#RRGGBB`) for the right country's fill, header line and bar.             | `#FF0000`             |
| `left_border_color`       | Hex color (`#RRGGBB`) for the left country's outline.                                | `#00FFFF`             |
| `right_border_color`      | Hex color (`#RRGGBB`) for the right country's outline.                               | `#FFCC00`             |
| `background_color`        | Hex color (`#RRGGBB`) for the map, bar and list backgrounds.                         | `#000000`             |
| `enable_pacific_center`   | If `true`, recenters countries that cross the antimeridian so they aren't split.    | `true`                |
| `skip_small`              | Polygon rings with this many points or fewer aren't drawn. `0` draws every ring.     | `0`                   |
| `button_font_size`        | Font size for the buttons, in pixels.                                                | `14`                  |
| `search_font_size`        | Font size for the search fields, in pixels.                                          | `14`                  |
| `country_list_font_size`  | Font size for the country lists, in pixels.                                          | `18`                  |
| `header_font_size`        | Font size for the header above the map, in pixels.                                   | `36`                  |
| `map_data_path`           | Directory that holds the GeoJSON files.                                              | `mapdata`             |
| `country_data_path`       | Path to the JSON file of country names, ISO codes and areas.                         | `country_data.json`   |

The `settings.json` that ships with HowBig sets `skip_small` to `25` and `header_font_size` to `18`:
```json
{
  "debug_show_boundary": false,
  "background_color": "#000000",
  "left_color": "#00FF00",
  "right_color": "#FF0000",
  "left_border_color": "#00FFFF",
  "right_border_color": "#FFCC00",
  "enable_pacific_center": true,
  "skip_small": 25,
  "button_font_size": 14,
  "search_font_size": 14,
  "country_list_font_size": 18,
  "header_font_size": 18,
  "map_data_path": "mapdata",
  "country_data_path": "country_data.json"
}
```

## Data Attribution

This project uses geographic data from [geoBoundaries](https://www.geoboundaries.org/), provided under the [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/) license. See [ATTRIBUTION.md](ATTRIBUTION.md).

## License

Copyright © Joel L. Caesar. This software is licensed under the GNU General Public License v3.0 (GPL-3.0). See [LICENSE.md](LICENSE.md) for the full text.

## Scripts

The `scripts/` directory contains utility scripts for data management.

### `download_geojson.go`

This script automates the process of downloading and optimizing GeoJSON data from the geoBoundaries API. It performs the following actions:
- Reads the list of countries from `country_data.json`.
- Fetches the simplified ADM0 (national level) boundaries for each country.
- Optimizes the data by truncating coordinates to 4 decimal places and removing duplicate points.
- Saves the resulting GeoJSON files into the `mapdata/` directory.

To run the script:
```bash
go run scripts/download_geojson.go
```
