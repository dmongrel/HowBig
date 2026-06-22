# HowBig

HowBig is a Go-based application designed to visualize and compare the geographical sizes of different countries. Built with the [Fyne](https://fyne.io/) GUI toolkit, it provides an interactive way to see how countries stack up against each other in terms of their physical footprint.

<img width="1200" height="502" alt="image" src="https://github.com/user-attachments/assets/bf6f87b2-0f8e-499d-9927-5bb6055e54b5" />

## Features

- **Side-by-Side Comparison:** Select two countries to compare their shapes and relative sizes.
- **Accurate Rendering:** Uses GeoJSON data to draw precise country boundaries.
- **Area Statistics:** Displays the surface area (in square miles) for the selected countries.
- **Responsive UI:** Dynamic scaling ensures countries fit within the application window.
- **Caching:** Efficient GeoJSON processing with internal caching for smooth performance.

## Build and Install

### Prerequisites

- **Go:** Version 1.26 or higher.
- **Fyne Dependencies:** You will need the dependencies required by the Fyne toolkit for your operating system (e.g., C compiler, graphics drivers). See the [Fyne Installation Guide](https://developer.fyne.io/started/) for details.
- **Inno Setup 7:** Required for packaging releases on Windows.
- **Bash/Git Bash:** Required for using the `Makefile` (especially on Windows).
- **Screen Size:** A minimum resolution of 1280 x 768 is required for the application to display correctly.

### Building from Source

1. Clone the repository to your local machine.
2. Navigate to the project directory.
3. Fetch the dependencies:
   ```bash
   go mod tidy
   ```
4. Build the application:
   ```bash
   go build .
   ```
5. Run the executable:
   - Windows: `HowBig.exe`
   - Linux/macOS: `./HowBig`

Alternatively, you can run it directly without building:
```bash
go run .
```

### Using the Makefile

For convenience, a `Makefile` is provided to automate common tasks. **Note:** This requires a Bash-compatible environment (e.g., Git Bash on Windows).

- **Build the application:**
  ```bash
  make build
  ```
- **Run the application:**
  ```bash
  make run
  ```
- **Clean build artifacts:**
  ```bash
  make clean
  ```
- **Package a release (requires Inno Setup 7):**
  ```bash
  make release
  ```

## Settings

The application can be configured via the `settings.json` file located in the project root.

| Setting                   | Description                                                                          | Default               |
|:--------------------------|:-------------------------------------------------------------------------------------|:----------------------|
| `debug_show_boundary`     | If `true`, draws a red bounding box around the rendered country for debugging.       | `false`               |
| `left_color`              | Hex color code for the left selected country's fill.                                 | `#00FF00`             |
| `right_color`             | Hex color code for the right selected country's fill.                                | `#FF0000`             |
| `left_border_color`       | Hex color code for the left country's outline.                                      | `#00FFFF`             |
| `right_border_color`      | Hex color code for the right country's outline.                                     | `#FFCC00`             |
| `background_color`        | Hex color code for the application background.                                       | `#000000`             |
| `enable_pacific_center`   | If `true`, applies special centering for countries that cross the anti-meridian.     | `true`                |
| `skip_small`              | Minimum number of points required for a polygon to be rendered.                      | `25`                  |
| `button_font_size`        | Font size for buttons and general widgets.                                           | `14`                  |
| `search_font_size`        | Font size for search input fields.                                                   | `14`                  |
| `country_list_font_size`  | Font size for the country selection lists.                                           | `18`                  |
| `header_font_size`        | Font size for the top header text.                                                   | `18`                  |
| `map_data_path`           | Directory path where GeoJSON files are stored.                                       | `mapdata`             |
| `country_data_path`       | Path to the JSON file containing country metadata.                                   | `country_data.json`   |

Example `settings.json`:
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

This project uses geographic data from [geoBoundaries](https://www.geoboundaries.org/), provided under the [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/) license.

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
