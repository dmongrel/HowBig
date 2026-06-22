# HowBig Project Outline

HowBig is a Go-based application built with the Fyne toolkit to visualize and compare the geographical sizes of different countries using GeoJSON data and Mercator projection.

## 1. Core Source Files

### `main.go`
The entry point and primary UI logic of the application.
- **UI Components:**
    - `MapWidget`: A custom Fyne widget for displaying country maps.
    - `createList`: Generates searchable side-by-side country selection lists.
    - `updateHeader` & `updateMapDisplay`: Manage dynamic UI updates based on selection.
    - `showMessage`: Displays modal information or error dialogs.
    - `About Button`: Provides access to version and attribution information.
- **Rendering Logic:**
    - `drawCountry`: Orchestrates the drawing of a country, including coordinate transformation.
    - `drawFilledPolygon`: Creates a Fyne Raster object for high-DPI anti-aliased rendering.
    - `fillPolygonIntoImage`: Uses the `gg` library to perform optimized scanline filling.

### `geometry.go`
Centralizes geometry structures, scaling, and fitting logic.
- **Projections & Transformations:**
    - `LatLonToMercator`: Converts geographic coordinates to Mercator space (0-1).
    - `NeedsPacificCentering` & `ApplyPacificCentering`: Manage countries crossing the anti-meridian.
- **Scale & Fitting:**
    - `getFitScale`: Calculates the necessary scale to fit a country within the window.
    - `getScaleAndOrder`: Determines the common scale for side-by-side comparisons.

### `mapdata.go`
Handles geographic data loading and coordinate transformations.
- **Data Loading:**
    - `FetchAndCacheGeoJSON`: Loads GeoJSON from disk with an internal 5-item LRU cache.
    - `convertGeoJSONToDisplayFormat`: Parses raw GeoJSON and pre-calculates Mercator points.

### `country_data.go`
Manages the metadata for available countries.
- `CountryInfo`: Stores name, ISO code, area, and associated GeoJSON filename.
- `NewCountryCollection`: Loads the master country list from the root `country_data.json`.

### `utils.go`
Provides general utility functions.
- `ParseHexColor`: Converts hexadecimal color strings to `color.NRGBA`.

## 2. Key Data Structures

### `geometry.go`
- `Point`: 2D coordinates with `float64` precision.
- `BoundingBox`: Stores `MinX`, `MaxX`, `MinY`, `MaxY` and dimensions in Mercator space.
- `GeoData`: Encapsulates pre-calculated paths (as `[][]Point`) and the `BoundingBox`.

### `main.go`
- `Settings`: User-configurable options (colors, debug flags, Pacific centering).
- `MapWidget`: Extends `widget.BaseWidget` to hold a map container.

## 3. Data & Configuration

- **`country_data.json`**: Master index of countries and their metadata (located in root).
- **`mapdata/`**:
    - `*.geojson`: Individual country boundary data files (simplified and optimized).
- **`settings.json`**:
    - User settings for line colors, debug boundaries, and performance filters.

## 4. Utility Scripts (`scripts/`)
- `download_geojson.go`: Consolidated script that downloads, simplifies (truncates precision), and removes duplicate points from GeoJSON data.
- `get_size.go`: Analyzes GeoJSON file sizes.

## 5. Documentation
- `README.md`: Project description, build instructions, and settings documentation.
- `ATTRIBUTION.md`: Data source and license information.
- `COMMENTS.md`: Internal evaluation and recommendations.
