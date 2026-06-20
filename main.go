package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CountryData holds the global collection of country information.
// cCenter is the central container for the application.
// cMap is the custom widget that renders the map.
// headerContainer holds the header elements for area display.
// leftBar and rightBar are containers for the side information panels.
// leftSelectedCountry and rightSelectedCountry manage the selection state.
// AppSettings stores global application configuration.
var (
	CountryData          *CountryCollection
	cCenter              *fyne.Container
	cMap                 *MapWidget
	headerContainer      *fyne.Container
	leftBar              *fyne.Container
	rightBar             *fyne.Container
	leftSelectedCountry  binding.String
	rightSelectedCountry binding.String
	AppSettings          Settings
)

// Settings define the UI configuration for the application.
type Settings struct {
	DebugShowBoundary   bool   `json:"debug_show_boundary"`   // DebugShowBoundary determines if the bounding box is rendered.
	LeftColor           string `json:"left_color"`            // LeftColor is the hex color for the left map.
	RightColor          string `json:"right_color"`           // RightColor is the hex color for the right map.
	EnablePacificCenter bool   `json:"enable_pacific_center"` // EnablePacificCenter determines if the map is centered on the Pacific Ocean for countries spanning the 180-degree meridian.
	SkipSmall           int    `json:"skip_small"`            // SkipSmall determines if polygons with few points are skipped.
}

// MapWidget is a custom widget that provides a map interface.
type MapWidget struct {
	widget.BaseWidget
	Container *fyne.Container // Container holds the map canvas objects.
}

// NewMapWidget creates and initializes a new MapWidget instance with a container.
func NewMapWidget() *MapWidget {
	zm := &MapWidget{
		Container: container.NewWithoutLayout(),
	}
	zm.ExtendBaseWidget(zm)
	return zm
}

// CreateRenderer creates and returns a renderer for the MapWidget.
func (zm *MapWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(zm.Container)
}

// Resize handles the resizing of the MapWidget and triggers a map redraw if necessary.
func (zm *MapWidget) Resize(s fyne.Size) {
	zm.BaseWidget.Resize(s)
	if cMap != nil && leftBar != nil && rightBar != nil && headerContainer != nil {
		updateMapDisplay()
	}
}

// init initializes the global country collection and loads application settings.
func init() {
	cc := NewCountryCollection()
	CountryData = cc
	loadSettings()
}

// loadSettings reads application configuration from settings.json, applying default values if the file is missing or invalid.
func loadSettings() {
	data, err := os.ReadFile("settings.json")
	if err != nil {
		log.Println("Error reading settings.json, using defaults:", err)
		AppSettings = Settings{DebugShowBoundary: false, LeftColor: "#00FF00", RightColor: "#FF0000", EnablePacificCenter: true, SkipSmall: 0}
		return
	}
	if err := json.Unmarshal(data, &AppSettings); err != nil {
		log.Println("Error unmarshaling settings.json:", err)
		AppSettings = Settings{DebugShowBoundary: false, LeftColor: "#00FF00", RightColor: "#FF0000", EnablePacificCenter: true, SkipSmall: 0}
	}
	if AppSettings.LeftColor == "" {
		AppSettings.LeftColor = "#00FF00"
	}
	if AppSettings.RightColor == "" {
		AppSettings.RightColor = "#FF0000"
	}
}

// ParseHexColor parses a hexadecimal color string (e.g., "#RRGGBB") and returns its color.NRGBA representation.
func ParseHexColor(s string) color.NRGBA {
	var r, g, b uint8
	if len(s) == 7 && s[0] == '#' {
		if _, err := fmt.Sscanf(s[1:], "%02x%02x%02x", &r, &g, &b); err == nil {
			return color.NRGBA{R: r, G: g, B: b, A: 255}
		}
	}
	return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
}

// createList creates a scrollable list of countries with search functionality and a selection callback.
// The width parameter sets the minimum width of the list.
func createList(width float32, onSelected func(string)) fyne.CanvasObject {
	filteredIndices := make([]int, len(CountryData.Countries))
	for i := range CountryData.Countries {
		filteredIndices[i] = i
	}

	list := widget.NewList(
		func() int { return len(filteredIndices) },
		func() fyne.CanvasObject {
			text := canvas.NewText("", color.White)
			text.TextSize = 18
			return container.NewPadded(text)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			padded := obj.(*fyne.Container)
			text := padded.Objects[0].(*canvas.Text)
			text.Text = CountryData.Countries[filteredIndices[id]].Name
			text.Refresh()
		},
	)
	var selectedID = -1
	list.OnSelected = func(id widget.ListItemID) {
		realID := filteredIndices[id]
		if realID == selectedID {
			list.Unselect(id)
			selectedID = -1
			onSelected("")
		} else {
			selectedID = realID
			onSelected(CountryData.Countries[realID].Name)
		}
	}

	bg := canvas.NewRectangle(color.Black)
	// Set a reasonable width for the lists
	scroll := container.NewScroll(list)
	scroll.SetMinSize(fyne.NewSize(width, 0))

	button := widget.NewButton("Deselect All", func() {
		list.UnselectAll()
		onSelected("")
	})

	entry := widget.NewEntry()
	entry.SetPlaceHolder("Search...")
	entry.OnChanged = func(s string) {
		filteredIndices = []int{}
		for i, c := range CountryData.Countries {
			if strings.Contains(strings.ToLower(c.Name), strings.ToLower(s)) {
				filteredIndices = append(filteredIndices, i)
			}
		}
		list.Refresh()
	}

	clearBtn := widget.NewButton("X", func() {
		entry.SetText("")
	})

	searchBar := container.NewBorder(nil, nil, nil, clearBtn, entry)

	return container.NewBorder(searchBar, button, nil, nil, container.NewStack(bg, scroll))
}

// addBorder applies a visual border to a Fyne canvas object using a stack container.
func addBorder(obj fyne.CanvasObject) fyne.CanvasObject {
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = color.White
	border.StrokeWidth = 2
	return container.NewStack(obj, border)
}

// formatNumber formats a float as a string with comma-separated thousands.
func formatNumber(n float64) string {
	s := fmt.Sprintf("%.0f", n)
	var res []byte
	for i, j := len(s)-1, 0; i >= 0; i, j = i-1, j+1 {
		if j > 0 && j%3 == 0 {
			res = append([]byte{','}, res...)
		}
		res = append([]byte{s[i]}, res...)
	}
	return string(res)
}

// getFitScale calculates the scale factor required to fit the bounding box of a country within the available display area.
func getFitScale(country string) float32 {
	paths, err := FetchAndCacheGeoJSON(country, true, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)
	if err != nil {
		return 1.0
	}
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, path := range paths {
		for _, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))
			minX = min(minX, mx)
			maxX = max(maxX, mx)
			minY = min(minY, my)
			maxY = max(maxY, my)
		}
	}
	mercWidth := maxX - minX
	mercHeight := maxY - minY
	if mercWidth == 0 || mercHeight == 0 {
		return 1.0
	}

	size := cMap.Container.Size()
	if headerContainer != nil {
		h := headerContainer.MinSize().Height
		if size.Height > h {
			size.Height -= h
		} else {
			size.Height = 0
		}
	}
	if size.Width < 4 || size.Height < 4 {
		return 1.0
	}

	scaleX := (size.Width - 4) / float32(mercWidth)
	scaleY := (size.Height - 4) / float32(mercHeight)
	return min(scaleX, scaleY)
}

// getScaleAndOrder determines the appropriate scale and drawing order for selected countries.
// It uses the larger of the two bounding box areas to ensure both are visible and fit the screen.
func getScaleAndOrder(active, other string) (float32, string, string) {
	if active == "" && other == "" {
		return 1.0, "", ""
	}
	if active == "" {
		return getFitScale(other), other, ""
	}
	if other == "" {
		return getFitScale(active), active, ""
	}

	pathsActive, errActive := FetchAndCacheGeoJSON(active, true, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)
	pathsOther, errOther := FetchAndCacheGeoJSON(other, true, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)

	if errActive != nil && errOther != nil {
		return 1.0, active, other
	}
	if errActive != nil {
		return getFitScale(other), other, active
	}
	if errOther != nil {
		return getFitScale(active), active, other
	}

	minXActive, minYActive := math.MaxFloat64, math.MaxFloat64
	maxXActive, maxYActive := -math.MaxFloat64, -math.MaxFloat64
	for _, path := range pathsActive {
		for _, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))
			minXActive = min(minXActive, mx)
			maxXActive = max(maxXActive, mx)
			minYActive = min(minYActive, my)
			maxYActive = max(maxYActive, my)
		}
	}

	minXOther, minYOther := math.MaxFloat64, math.MaxFloat64
	maxXOther, maxYOther := -math.MaxFloat64, -math.MaxFloat64
	for _, path := range pathsOther {
		for _, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))
			minXOther = min(minXOther, mx)
			maxXOther = max(maxXOther, mx)
			minYOther = min(minYOther, my)
			maxYOther = max(maxYOther, my)
		}
	}

	if cMap == nil {
		return 1.0, active, other
	}
	size := cMap.Container.Size()
	if size.Width < 4 || size.Height < 4 {
		return 1.0, active, other
	}

	areaActive := (maxXActive - minXActive) * (maxYActive - minYActive)
	areaOther := (maxXOther - minXOther) * (maxYOther - minYOther)

	if areaActive > areaOther {
		return getFitScale(active), active, other
	}
	return getFitScale(other), other, active
}

// updateHeader updates the header to display the surface area information for the selected countries.
func updateHeader() {
	const sqMiToSqKm = 2.58998811
	left, _ := leftSelectedCountry.Get()
	right, _ := rightSelectedCountry.Get()

	formatPart := func(name string) string {
		areaMi := getArea(name)
		areaKm := areaMi * sqMiToSqKm
		return fmt.Sprintf("%s: %s sq. mi. / %s km.", name, formatNumber(areaMi), formatNumber(areaKm))
	}

	leftPart := ""
	if left != "" {
		leftPart = formatPart(left)
	}

	rightPart := ""
	if right != "" {
		rightPart = formatPart(right)
	}

	sep := ""
	if leftPart != "" && rightPart != "" {
		sep = " || "
	}

	headerContainer.Objects = nil
	if leftPart != "" {
		t := canvas.NewText(leftPart, ParseHexColor(AppSettings.LeftColor))
		t.TextSize = 36
		headerContainer.Add(t)
	}
	if sep != "" {
		t := canvas.NewText(sep, color.White)
		t.TextSize = 36
		headerContainer.Add(t)
	}
	if rightPart != "" {
		t := canvas.NewText(rightPart, ParseHexColor(AppSettings.RightColor))
		t.TextSize = 36
		headerContainer.Add(t)
	}
	headerContainer.Refresh()
}

// updateMapDisplay clears the map canvas and redraws the selected countries based on the calculated target scale.
func updateMapDisplay() {
	clearAll()
	left, _ := leftSelectedCountry.Get()
	right, _ := rightSelectedCountry.Get()

	scale, large, small := getScaleAndOrder(left, right)
	leftColor := ParseHexColor(AppSettings.LeftColor)
	rightColor := ParseHexColor(AppSettings.RightColor)

	if left != "" {
		drawBar(leftBar, getArea(left), leftColor)
	}
	if right != "" {
		drawBar(rightBar, getArea(right), rightColor)
	}

	var largeColor, smallColor color.Color
	if large == left {
		largeColor = leftColor
	} else {
		largeColor = rightColor
	}

	if small == right {
		smallColor = rightColor
	} else {
		smallColor = leftColor
	}

	if large != "" {
		drawCountry(cMap, large, scale, false, largeColor)
	}
	if small != "" {
		drawCountry(cMap, small, scale, false, smallColor)
	}
}

// clearAll removes all canvas objects from the map container and refreshes it.
func clearAll() {
	leftBar.Objects = nil
	leftBar.Refresh()
	rightBar.Objects = nil
	rightBar.Refresh()
	cMap.Container.Objects = []fyne.CanvasObject{canvas.NewRectangle(color.Black)}
	cMap.Container.Refresh()
}

// customTheme is a custom Fyne theme to override default styling.
type customTheme struct {
	base fyne.Theme // base is the default theme being overridden.
}

// Color returns the color for a given theme color name and variant.
func (c *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return c.base.Color(name, variant)
}

// Font returns the font resource for a given style.
func (c *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return c.base.Font(style)
}

// Icon returns the icon resource for a given icon name.
func (c *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return c.base.Icon(name)
}

// Size returns the size for a given theme size name, with customizations for scrollbars.
func (c *customTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameScrollBar || name == theme.SizeNameScrollBarSmall {
		return 20
	}
	return c.base.Size(name)
}

// main is the application entry point, setting up the GUI and initializing components.
func main() {
	a := app.New()
	leftSelectedCountry = binding.NewString()
	rightSelectedCountry = binding.NewString()
	a.Settings().SetTheme(&customTheme{base: theme.DefaultTheme()})
	w := a.NewWindow("Fullscreen App")
	w.Resize(fyne.NewSize(1280, 1024))
	w.SetFullScreen(true)

	isFullScreen := true
	w.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		if key.Name == fyne.KeyEscape {
			a.Quit()
		} else if key.Name == fyne.KeyF {
			isFullScreen = !isFullScreen
			w.SetFullScreen(isFullScreen)
		}
	})

	var maxWidth float32
	for _, country := range CountryData.Countries {
		size := fyne.MeasureText(country.Name, 18, fyne.TextStyle{})
		if size.Width > maxWidth {
			maxWidth = size.Width
		}
	}
	maxWidth += 2

	cMap = NewMapWidget()
	headerContainer = container.NewHBox()
	cCenter = container.NewBorder(container.NewCenter(headerContainer), nil, nil, nil, cMap)

	leftBar = container.NewWithoutLayout()
	leftBarWrapper := container.NewScroll(leftBar)
	leftBarWrapper.SetMinSize(fyne.NewSize(50, 0))
	rightBar = container.NewWithoutLayout()
	rightBarWrapper := container.NewScroll(rightBar)
	rightBarWrapper.SetMinSize(fyne.NewSize(50, 0))

	listener := &selectionListener{}
	leftSelectedCountry.AddListener(listener)
	rightSelectedCountry.AddListener(listener)

	innerBorder := container.NewBorder(nil, nil, leftBarWrapper, rightBarWrapper, cCenter)

	left := addBorder(createList(maxWidth, func(c string) {
		if err := leftSelectedCountry.Set(c); err != nil {
			log.Printf("Error setting left country: %v", err)
		}
	}))
	right := addBorder(createList(maxWidth, func(c string) {
		if err := rightSelectedCountry.Set(c); err != nil {
			log.Printf("Error setting right country: %v", err)
		}
	}))
	center := addBorder(innerBorder)

	w.SetContent(container.NewBorder(nil, nil, left, right, center))

	w.ShowAndRun()
}

// selectionListener implements binding.DataListener to react to country selection changes.
type selectionListener struct{}

// DataChanged reacts to country selection changes and updates the UI.
func (s *selectionListener) DataChanged() {
	updateHeader()
	updateMapDisplay()
}

// drawBar draws a vertical bar representing the relative area of a country on a given container.
func drawBar(c *fyne.Container, area float64, barColor color.Color) {
	size := c.Size()
	if size.Width == 0 || size.Height == 0 {
		size = fyne.NewSize(50, 500)
	}

	const maxArea = 6601667.0
	proportion := area / maxArea
	if proportion > 1 {
		proportion = 1
	}

	barHeight := float32(proportion) * size.Height

	rect := canvas.NewRectangle(barColor)

	// Apply padding: 2px on all sides
	padding := float32(2.0)
	rectWidth := size.Width - (padding * 2)
	rectHeight := barHeight - (padding * 2)

	if rectWidth < 0 {
		rectWidth = 0
	}
	if rectHeight < 0 {
		rectHeight = 0
	}

	rect.Resize(fyne.NewSize(rectWidth, rectHeight))
	rect.Move(fyne.NewPos(padding, size.Height-barHeight+padding))

	c.Objects = []fyne.CanvasObject{rect}
	c.Refresh()
}

// getArea retrieves the surface area of a country by its name from the global CountryData.
func getArea(name string) float64 {
	for _, country := range CountryData.Countries {
		if country.Name == name {
			return country.Area
		}
	}
	return 0
}

// fillPolygonIntoImage implements a scanline fill algorithm to color the polygon.
func fillPolygonIntoImage(img *image.RGBA, polyPoints []fyne.Position, fillColor color.Color) {
	if len(polyPoints) < 3 {
		return
	}
	minY, maxY := polyPoints[0].Y, polyPoints[0].Y
	for _, p := range polyPoints {
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	for y := int(math.Floor(float64(minY))); y <= int(math.Ceil(float64(maxY))); y++ {
		var intersections []float32
		for i := 0; i < len(polyPoints); i++ {
			p1 := polyPoints[i]
			p2 := polyPoints[(i+1)%len(polyPoints)]

			if (p1.Y <= float32(y) && p2.Y > float32(y)) || (p2.Y <= float32(y) && p1.Y > float32(y)) {
				x := p1.X + (float32(y)-p1.Y)*(p2.X-p1.X)/(p2.Y-p1.Y)
				intersections = append(intersections, x)
			}
		}
		sort.Slice(intersections, func(i, j int) bool { return intersections[i] < intersections[j] })

		for i := 0; i < len(intersections)-1; i += 2 {
			xStart := intersections[i]
			xEnd := intersections[i+1]
			// Use exact range for scanline to avoid 1-pixel bloat
			for x := int(math.Round(float64(xStart))); x <= int(math.Round(float64(xEnd))); x++ {
				img.Set(x, y, fillColor)
			}
		}
	}
}

// drawFilledPolygon creates a Raster canvas object representing the filled polygon.
func drawFilledPolygon(polyPoints []fyne.Position, fillColor color.Color) fyne.CanvasObject {
	minX, maxX := polyPoints[0].X, polyPoints[0].X
	minY, maxY := polyPoints[0].Y, polyPoints[0].Y
	for _, p := range polyPoints {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	w := int(math.Round(float64(maxX))) - int(math.Round(float64(minX))) + 1
	h := int(math.Round(float64(maxY))) - int(math.Round(float64(minY))) + 1

	raster := canvas.NewRaster(func(width, height int) image.Image {
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		relativePoints := make([]fyne.Position, len(polyPoints))
		for i, p := range polyPoints {
			relativePoints[i] = fyne.NewPos(p.X-float32(math.Round(float64(minX))), p.Y-float32(math.Round(float64(minY))))
		}
		fillPolygonIntoImage(img, relativePoints, fillColor)
		return img
	})
	raster.Resize(fyne.NewSize(float32(w), float32(h)))
	raster.Move(fyne.NewPos(float32(math.Round(float64(minX))), float32(math.Round(float64(minY)))))
	return raster
}

// drawCountry draws the GeoJSON paths of a country on the provided MapWidget.
// It applies scaling, centering, and optionally renders the bounding box for debugging purposes.
func drawCountry(zm *MapWidget, country string, scale float32, clear bool, lineColor color.Color) {
	paths, err := FetchAndCacheGeoJSON(country, true, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)
	if err != nil {
		log.Printf("Error loading %s: %v", country, err)
		return
	}

	if len(paths) == 0 {
		if clear {
			zm.Container.Objects = []fyne.CanvasObject{canvas.NewRectangle(color.Black)}
			zm.Container.Refresh()
		}
		return
	}

	size := zm.Container.Size()
	if size.Width == 0 || size.Height == 0 {
		size = fyne.NewSize(500, 500)
	}
	if headerContainer != nil {
		h := headerContainer.MinSize().Height
		if size.Height > h {
			size.Height -= h
		} else {
			size.Height = 0
		}
	}

	var objects []fyne.CanvasObject
	if clear {
		objects = []fyne.CanvasObject{canvas.NewRectangle(color.Black)}
	} else {
		objects = make([]fyne.CanvasObject, len(zm.Container.Objects))
		copy(objects, zm.Container.Objects)
		if len(objects) == 0 {
			objects = []fyne.CanvasObject{canvas.NewRectangle(color.Black)}
		}
	}

	// Pass 1: Transform and find bounds
	transformedPaths := make([][]fyne.Position, len(paths))
	minXLocal, minYLocal := math.MaxFloat64, math.MaxFloat64
	maxXLocal, maxYLocal := -math.MaxFloat64, -math.MaxFloat64
	for i, path := range paths {
		transformedPaths[i] = make([]fyne.Position, len(path))
		for j, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))
			transformedPaths[i][j] = fyne.NewPos(float32(mx), float32(my))
			minXLocal = min(minXLocal, mx)
			maxXLocal = max(maxXLocal, mx)
			minYLocal = min(minYLocal, my)
			maxYLocal = max(maxYLocal, my)
		}
	}

	transformedMinX, transformedMinY := minXLocal, minYLocal
	transformedMaxX, transformedMaxY := maxXLocal, maxYLocal

	transformedWidth := transformedMaxX - transformedMinX
	transformedHeight := transformedMaxY - transformedMinY
	offsetX := (size.Width - float32(transformedWidth)*scale) / 2
	offsetY := (size.Height - float32(transformedHeight)*scale) / 2

	pixelMinX, pixelMinY, pixelMaxX, pixelMaxY, err := GetBoundingBox(country, scale, offsetX, offsetY, AppSettings.SkipSmall, AppSettings.EnablePacificCenter)
	if err != nil {
		log.Printf("Error getting pixel bbox for %s: %v", country, err)
		return
	}

	LogSouthernmostPixels(country, paths, scale, offsetX, offsetY, transformedMinX, transformedMaxX, transformedMinY, transformedMaxY)

	// Draw bounding box
	if AppSettings.DebugShowBoundary {
		rect := canvas.NewRectangle(color.Transparent)
		rect.StrokeColor = color.NRGBA{R: 255, A: 255}
		rect.StrokeWidth = 1
		rect.Resize(fyne.NewSize(float32(pixelMaxX-pixelMinX), float32(pixelMaxY-pixelMinY)))
		rect.Move(fyne.NewPos(float32(pixelMinX), float32(pixelMinY)))
		objects = append(objects, rect)
	}

	// Pass 2: Draw the transformed paths
	for _, path := range transformedPaths {
		var polyPoints []fyne.Position
		for _, p := range path {
			// Screen space
			screenX := (float64(p.X) - transformedMinX) * float64(scale)
			screenY := (float64(p.Y) - transformedMinY) * float64(scale)

			// Apply centering
			pPos := fyne.NewPos(float32(screenX)+offsetX, float32(screenY)+offsetY)
			polyPoints = append(polyPoints, pPos)
		}

		if len(polyPoints) < 3 {
			continue
		}

		objects = append(objects, drawFilledPolygon(polyPoints, lineColor))
	}
	zm.Container.Objects = objects
	zm.Container.Refresh()
}
