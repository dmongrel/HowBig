# HowBig

HowBig is a Go-based application designed to visualize and compare the geographical sizes of different countries. Built with the [Fyne](https://fyne.io/) GUI toolkit, it provides an interactive way to see how countries stack up against each other in terms of their physical footprint.

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

## Settings

The application can be configured via the `settings.json` file located in the project root.

| Setting               | Description                                                                          | Default   |
|:----------------------|:-------------------------------------------------------------------------------------|:----------|
| `LineColor`           | Hex color code for the primary selected country's outline and fill.                  | `#ff0000` |
| `LineColor2`          | Hex color code for the second selected country's outline and fill.                   | `#00ff00` |
| `DebugShowBoundary`   | If `true`, draws a red bounding box around the rendered country for debugging.       | `false`   |
| `SkipSmall`           | If `true`, ignores small polygons (islands) during rendering to improve performance. | `true`    |
| `EnablePacificCenter` | If `true`, applies special centering for countries that cross the anti-meridian.     | `true`    |

Example `settings.json`:
```json
{
  "LineColor": "#ff0000",
  "LineColor2": "#00ff00",
  "DebugShowBoundary": false,
  "SkipSmall": true,
  "EnablePacificCenter": true
}
```

## Data Attribution

This project uses geographic data from [geoBoundaries](https://www.geoboundaries.org/), provided under the [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/) license.
