# HowBig Project Outline

HowBig is a Go-based application built with the Fyne toolkit to visualize and compare the geographical sizes of different countries using GeoJSON data and Mercator projection.

## 1. Core Source Files

### `main.go`
The entry point and primary UI logic of the application.
- **UI Components:**
    - `MapWidget`: A custom Fyne widget for displaying country maps.
    - `createList`: Generates searchable side-by-side country selection lists.
    - `updateHeader` & `updateMapDisplay`: Manage dynamic UI updates based on selection.
- **Rendering Logic:**
    - `drawCountry`: Orchestrates the drawing of a country, including scaling and centering.
    - `drawFilledPolygon`: Uses a scanline algorithm to render filled geographic shapes.
    - `fillPolygonIntoImage`: Core implementation of the scanline fill algorithm.
- **Scale & Transformation:**
    - `getFitScale`: Calculates the necessary scale to fit a country within the window.
    - `getScaleAndOrder`: Determines the common scale for side-by-side comparisons.

### `mapdata.go`
Handles geographic data processing and coordinate transformations.
- **Projections:**
    - `LatLonToMercator`: Converts geographic coordinates to Mercator space.
- **Data Loading:**
    - `FetchAndCacheGeoJSON`: Loads GeoJSON from disk with an internal thread-safe cache.
    - `convertGeoJSONToDisplayFormat`: Parses raw GeoJSON into internal structures.
- **Handling Edge Cases:**
    - `NeedsPacificCentering` & `ApplyPacificCentering`: Manage countries crossing the anti-meridian.

### `country_data.go`
Manages the metadata for available countries.
- `CountryInfo`: Stores name, ISO code, area, and associated GeoJSON filename.
- `NewCountryCollection`: Loads the master country list from `mapdata/country_data.json`.

## 2. Key Data Structures

### `mapdata.go`
- `Point`: 2D coordinates with `float64` precision.
- `BoundingBox`: Stores `MinX`, `MaxX`, `MinY`, `MaxY` and dimensions (`Width`, `Height`).
- `GeoData`: Encapsulates paths (as `[][]Point`) and the `BoundingBox`.

### `main.go`
- `Settings`: User-configurable options (colors, debug flags).
- `MapWidget`: Extends `widget.BaseWidget` to hold a map container.

### `country_data.go`
- `CountryInfo`: Metadata (Name, CompactName, ISOCode, Area).

## 3. Data & Configuration

- **`mapdata/`**:
    - `country_data.json`: Master index of countries and their metadata.
    - `*.geojson`: Individual country boundary data files.
- **`settings.json`**:
    - User settings for line colors, debug boundaries, and performance filters (`SkipSmall`).

## 4. Utility Scripts (`scripts/`)
- `get_size.go`: Analyzes GeoJSON file sizes.
- `recenter_geojson.go`: Recenters GeoJSON data for specific regions.
- `filter_russia.go`: Specialized script for handling large geographic datasets.

## 5. Documentation
- `README.md`: Project description, build instructions, and settings documentation.
- `ATTRIBUTION.md`: Data source and license information.
- `COMMENTS.md`: Internal evaluation and recommendations.
