package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"os"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/fogleman/gg"
)

const Version = "1.0.0"

// Settings define the UI configuration for the application.
type Settings struct {
	DebugShowBoundary   bool   `json:"debug_show_boundary"`   // DebugShowBoundary determines if the bounding box is rendered.
	LeftColor           string `json:"left_color"`            // LeftColor is the hex color for the left map.
	RightColor          string `json:"right_color"`           // RightColor is the hex color for the right map.
	LeftBorderColor     string `json:"left_border_color"`     // LeftBorderColor is the hex color for the left country border.
	RightBorderColor    string `json:"right_border_color"`    // RightBorderColor is the hex color for the right country border.
	BackgroundColor     string `json:"background_color"`      // BackgroundColor is the background color for the application.
	EnablePacificCenter bool   `json:"enable_pacific_center"` // EnablePacificCenter determines if the map is centered on the Pacific Ocean for countries spanning the 180-degree meridian.
	SkipSmall           int    `json:"skip_small"`            // SkipSmall determines if polygons with few points are skipped.
	MapDataPath         string `json:"map_data_path"`         // MapDataPath is the path to the directory containing GeoJSON files.
	CountryDataPath     string `json:"country_data_path"`     // CountryDataPath is the path to the country_data.json file.
}

// MapWidget is a custom widget that provides a map interface.
type MapWidget struct {
	widget.BaseWidget
	Container *fyne.Container // Container holds the map canvas objects.
}

// customTheme is a custom Fyne theme to override default styling.
type customTheme struct {
	base fyne.Theme // base is the default theme being overridden.
}

// App holds the application state and dependencies.
type App struct {
	fyneApp fyne.App
	window  fyne.Window

	Settings    *Settings
	CountryData *CountryCollection
	GeoCache    *GeoCache

	cCenter              *fyne.Container
	cMap                 *MapWidget
	headerContainer      *fyne.Container
	leftBar              *fyne.Container
	rightBar             *fyne.Container
	leftSelectedCountry  binding.String
	rightSelectedCountry binding.String
}

// selectionListener implements binding.DataListener to react to country selection changes.
type selectionListener struct {
	app *App
}

func (s *selectionListener) DataChanged() {
	s.app.updateHeader()
	s.app.updateMapDisplay()
}

func NewApp(fyneApp fyne.App) *App {
	settings := loadSettings("settings.json")
	return &App{
		fyneApp:              fyneApp,
		Settings:             settings,
		GeoCache:             NewGeoCache(CacheLimit),
		leftSelectedCountry:  binding.NewString(),
		rightSelectedCountry: binding.NewString(),
	}
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

// Resize handles the resizing of the MapWidget and triggers necessary redraws.
func (zm *MapWidget) Resize(s fyne.Size) {
	zm.BaseWidget.Resize(s)
}

// loadSettings reads application configuration from settings.json, applying default values if the file is missing or invalid.
func loadSettings(path string) *Settings {
	var s Settings
	data, err := os.ReadFile(path)
	if err != nil {
		log.Println("Error reading settings.json, using defaults:", err)
		s = Settings{
			DebugShowBoundary:   false,
			LeftColor:           "#00FF00",
			RightColor:          "#FF0000",
			LeftBorderColor:     "#00FFFF",
			RightBorderColor:    "#FFCC00",
			BackgroundColor:     "#000000",
			EnablePacificCenter: true,
			SkipSmall:           0,
			MapDataPath:         "mapdata",
			CountryDataPath:     "country_data.json",
		}
	} else if err := json.Unmarshal(data, &s); err != nil {
		log.Println("Error unmarshaling settings.json:", err)
		s = Settings{
			DebugShowBoundary:   false,
			LeftColor:           "#00FF00",
			RightColor:          "#FF0000",
			LeftBorderColor:     "#00FFFF",
			RightBorderColor:    "#FFCC00",
			BackgroundColor:     "#000000",
			EnablePacificCenter: true,
			SkipSmall:           0,
			MapDataPath:         "mapdata",
			CountryDataPath:     "country_data.json",
		}
	}
	if s.LeftColor == "" {
		s.LeftColor = "#00FF00"
	}
	if s.RightColor == "" {
		s.RightColor = "#FF0000"
	}
	if s.LeftBorderColor == "" {
		s.LeftBorderColor = "#00FFFF"
	}
	if s.RightBorderColor == "" {
		s.RightBorderColor = "#FFCC00"
	}
	if s.BackgroundColor == "" {
		s.BackgroundColor = "#000000"
	}
	if s.MapDataPath == "" {
		s.MapDataPath = "mapdata"
	}
	if s.CountryDataPath == "" {
		s.CountryDataPath = "country_data.json"
	}
	return &s
}

// createList creates a scrollable list of countries with search functionality and a selection callback.
// The width parameter sets the minimum width of the list.
func (a *App) createList(width float64, onSelected func(string)) fyne.CanvasObject {
	filteredIndices := slices.Collect(func(yield func(int) bool) {
		for i := range a.CountryData.Countries {
			if !yield(i) {
				return
			}
		}
	})

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
			text.Text = a.CountryData.Countries[filteredIndices[id]].Name
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
			onSelected(a.CountryData.Countries[realID].Name)
		}
	}

	bg := canvas.NewRectangle(ParseHexColor(a.Settings.BackgroundColor))
	// Set a reasonable width for the lists
	scroll := container.NewScroll(list)
	scroll.SetMinSize(fyne.NewSize(float32(width), 0))

	button := widget.NewButton("Deselect All", func() {
		list.UnselectAll()
		onSelected("")
	})

	entry := widget.NewEntry()
	entry.SetPlaceHolder("Search...")
	entry.OnChanged = func(s string) {
		searchTerm := strings.ToLower(s)
		filter := func(yield func(int) bool) {
			for i, c := range a.CountryData.Countries {
				if strings.Contains(strings.ToLower(c.Name), searchTerm) {
					if !yield(i) {
						return
					}
				}
			}
		}
		filteredIndices = slices.Collect(filter)
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

// updateHeader updates the header to display the surface area information for the selected countries.
func (a *App) updateHeader() {
	const sqMiToSqKm = 2.58998811
	left, _ := a.leftSelectedCountry.Get()
	right, _ := a.rightSelectedCountry.Get()

	formatPart := func(name string) string {
		areaMi := a.getArea(name)
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

	a.headerContainer.Objects = nil
	if leftPart != "" {
		t := canvas.NewText(leftPart, ParseHexColor(a.Settings.LeftColor))
		t.TextSize = 36
		a.headerContainer.Add(t)
	}
	if sep != "" {
		t := canvas.NewText(sep, color.White)
		t.TextSize = 36
		a.headerContainer.Add(t)
	}
	if rightPart != "" {
		t := canvas.NewText(rightPart, ParseHexColor(a.Settings.RightColor))
		t.TextSize = 36
		a.headerContainer.Add(t)
	}
	a.headerContainer.Refresh()
}

// updateMapDisplay clears the map canvas and redraws the selected countries based on the calculated target scale.
func (a *App) updateMapDisplay() {
	a.clearAll()
	left, _ := a.leftSelectedCountry.Get()
	right, _ := a.rightSelectedCountry.Get()

	scale, large, small := a.getScaleAndOrder(left, right)
	leftColor := ParseHexColor(a.Settings.LeftColor)
	rightColor := ParseHexColor(a.Settings.RightColor)

	bgColor := ParseHexColor(a.Settings.BackgroundColor)
	if left != "" {
		leftColor.A = 127
		drawBar(a.leftBar, a.getArea(left), leftColor, bgColor)
	}
	if right != "" {
		rightColor.A = 127
		drawBar(a.rightBar, a.getArea(right), rightColor, bgColor)
	}

	var largeColor, smallColor color.Color
	if large == left {
		largeColor = leftColor
		smallColor = rightColor
	} else {
		largeColor = rightColor
		smallColor = leftColor
	}

	if large != "" {
		a.drawCountry(a.cMap, large, scale, false, largeColor, nil)
	}
	if small != "" {
		a.drawCountry(a.cMap, small, scale, false, smallColor, nil)
	}

	// Redraw borders in configured border colors
	if left != "" {
		borderColor := ParseHexColor(a.Settings.LeftBorderColor)
		a.drawCountry(a.cMap, left, scale, false, nil, borderColor)
	}
	if right != "" {
		borderColor := ParseHexColor(a.Settings.RightBorderColor)
		a.drawCountry(a.cMap, right, scale, false, nil, borderColor)
	}
}

// clearAll removes all canvas objects from the map container and refreshes it.
func (a *App) clearAll() {
	a.leftBar.Objects = nil
	a.leftBar.Refresh()
	a.rightBar.Objects = nil
	a.rightBar.Refresh()
	a.cMap.Container.Objects = []fyne.CanvasObject{canvas.NewRectangle(ParseHexColor(a.Settings.BackgroundColor))}
	a.cMap.Container.Refresh()
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

func (a *App) showAbout() {
	attribution := "geoBoundaries data is used under CC-BY 4.0 license.\nFor more information refer to ATTRIBUTION.md"
	msg := fmt.Sprintf("HowBig %s Copyright © 2026 Joel L. Caesar. All Rights Reserved.\n\n%s", Version, attribution)
	a.showMessage("About", msg)
}

// showMessage displays a 500x300 message window centered on the parent window with a given title.
func (a *App) showMessage(title string, message string) {

	// Use canvas.Text for guaranteed color and size
	text := widget.NewRichText(&widget.TextSegment{
		Text:  message,
		Style: widget.RichTextStyleParagraph,
	})
	// Override style for the text segment
	if len(text.Segments) > 0 {
		if ts, ok := text.Segments[0].(*widget.TextSegment); ok {
			ts.Style.ColorName = theme.ColorNameForeground
			// Size is not directly in RichTextStyle in older Fyne,
			// it uses the theme size.
		}
	}
	text.Wrapping = fyne.TextWrapWord

	bgColor := ParseHexColor(a.Settings.BackgroundColor)
	bg := canvas.NewRectangle(bgColor)
	content := container.NewStack(bg, container.NewScroll(text))

	if a.window != nil {
		d := dialog.NewCustom(title, "OK", content, a.window)
		d.Resize(fyne.NewSize(500, 300))
		// Apply custom theme to the dialog window if possible
		// Since we can't easily, the RichText within content should still use the app theme for foreground.
		d.Show()
	} else {
		w := a.fyneApp.NewWindow(title)
		w.Resize(fyne.NewSize(500, 300))
		okBtn := widget.NewButton("OK", func() {
			w.Close()
		})
		buttonBar := container.NewHBox(layout.NewSpacer(), okBtn, layout.NewSpacer())
		w.SetContent(container.NewBorder(nil, buttonBar, nil, nil, content))
		w.CenterOnScreen()
		w.Show()
	}
}

// main is the application's entry point, setting up the GUI and initializing components.
func main() {
	fyneApp := app.New()
	fyneApp.Settings().SetTheme(&customTheme{base: theme.DefaultTheme()})

	a := NewApp(fyneApp)

	cc, err := NewCountryCollection(a.Settings.CountryDataPath)
	if err != nil {
		a.showMessage("Error", fmt.Sprintf("Fatal error: failed to load country data: %v", err))
		log.Printf("Fatal error: failed to load country data: %v", err)
	}
	a.CountryData = cc

	a.window = fyneApp.NewWindow("Fullscreen App")
	a.window.Resize(fyne.NewSize(1280, 1024))
	a.window.SetFullScreen(true)

	isFullScreen := true
	a.window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		if key.Name == fyne.KeyEscape {
			fyneApp.Quit()
		} else if key.Name == fyne.KeyF {
			isFullScreen = !isFullScreen
			a.window.SetFullScreen(isFullScreen)
		}
	})

	var maxWidth float64
	for _, country := range a.CountryData.Countries {
		size := fyne.MeasureText(country.Name, 18, fyne.TextStyle{})
		if float64(size.Width) > maxWidth {
			maxWidth = float64(size.Width)
		}
	}
	maxWidth += 2

	a.cMap = NewMapWidget()
	a.headerContainer = container.NewHBox()
	aboutBtn := widget.NewButton("About", func() {
		a.showAbout()
	})
	a.cCenter = container.NewBorder(container.NewCenter(a.headerContainer), aboutBtn, nil, nil, a.cMap)

	a.leftBar = container.NewWithoutLayout()
	leftBarWrapper := container.NewScroll(a.leftBar)
	leftBarWrapper.SetMinSize(fyne.NewSize(50, 0))
	a.rightBar = container.NewWithoutLayout()
	rightBarWrapper := container.NewScroll(a.rightBar)
	rightBarWrapper.SetMinSize(fyne.NewSize(50, 0))

	listener := &selectionListener{app: a}
	a.leftSelectedCountry.AddListener(listener)
	a.rightSelectedCountry.AddListener(listener)

	innerBorder := container.NewBorder(nil, nil, leftBarWrapper, rightBarWrapper, a.cCenter)
	centerBg := canvas.NewRectangle(ParseHexColor(a.Settings.BackgroundColor))
	center := addBorder(container.NewStack(centerBg, innerBorder))

	left := addBorder(a.createList(maxWidth, func(c string) {
		if err := a.leftSelectedCountry.Set(c); err != nil {
			log.Printf("Error setting left country: %v", err)
		}
	}))
	right := addBorder(a.createList(maxWidth, func(c string) {
		if err := a.rightSelectedCountry.Set(c); err != nil {
			log.Printf("Error setting right country: %v", err)
		}
	}))

	a.window.SetContent(container.NewBorder(nil, nil, left, right, center))

	a.showAbout()

	a.window.ShowAndRun()
}

// drawBar draws a vertical bar representing the relative area of a country on a given container.
func drawBar(c *fyne.Container, area float64, barColor color.Color, bgColor color.Color) {
	size := c.Size()
	if size.Width == 0 || size.Height == 0 {
		size = fyne.NewSize(50, 500)
	}

	const maxArea = 6601667.0
	proportion := area / maxArea
	if proportion > 1 {
		proportion = 1
	}

	barHeight := proportion * float64(size.Height)

	// Create background border rectangle
	bgRect := canvas.NewRectangle(bgColor)
	bgRect.Resize(size)
	bgRect.Move(fyne.NewPos(0, 0))

	// Create bar color rectangle
	rect := canvas.NewRectangle(barColor)

	// Apply padding: 2px on all sides for the bar relative to the background
	padding := 2.0
	rectWidth := float64(size.Width) - (padding * 2)
	rectHeight := barHeight - (padding * 2)

	if rectWidth < 0 {
		rectWidth = 0
	}
	if rectHeight < 0 {
		rectHeight = 0
	}

	rect.Resize(fyne.NewSize(float32(rectWidth), float32(rectHeight)))
	rect.Move(fyne.NewPos(float32(padding), float32(float64(size.Height)-barHeight+padding)))

	c.Objects = []fyne.CanvasObject{bgRect, rect}
	c.Refresh()
}

// getArea retrieves the surface area of a country by its name from the collection.
func (a *App) getArea(name string) float64 {
	return a.CountryData.Areas[name]
}

// drawFilledPolygon creates a Raster canvas object representing the filled polygon.
func drawFilledPolygon(polyPoints []Point, fillColor color.Color, strokeColor color.Color, strokeWidth float64) fyne.CanvasObject {
	minX, maxX := polyPoints[0].X, polyPoints[0].X
	minY, maxY := polyPoints[0].Y, polyPoints[0].Y
	for _, p := range polyPoints {
		minX = min(minX, p.X)
		maxX = max(maxX, p.X)
		minY = min(minY, p.Y)
		maxY = max(maxY, p.Y)
	}

	w := int(math.Round(maxX)) - int(math.Round(minX)) + 1
	h := int(math.Round(maxY)) - int(math.Round(minY)) + 1

	raster := canvas.NewRaster(func(width, height int) image.Image {
		relativePoints := make([]Point, len(polyPoints))
		offsetX := math.Round(minX)
		offsetY := math.Round(minY)

		// Calculate scaling factor between Fyne's requested size and our calculated size
		// This handles high-DPI screens correctly.
		scaleX := float64(width) / float64(w)
		scaleY := float64(height) / float64(h)

		for i, p := range polyPoints {
			relativePoints[i] = Point{
				X: (p.X - offsetX) * scaleX,
				Y: (p.Y - offsetY) * scaleY,
			}
		}

		if len(relativePoints) < 3 {
			return image.NewRGBA(image.Rect(0, 0, width, height))
		}

		dc := gg.NewContext(width, height)
		dc.SetFillRule(gg.FillRuleEvenOdd)

		// Draw the path
		dc.MoveTo(relativePoints[0].X, relativePoints[0].Y)
		for i := 1; i < len(relativePoints); i++ {
			dc.LineTo(relativePoints[i].X, relativePoints[i].Y)
		}
		dc.ClosePath()

		if fillColor != nil {
			dc.SetColor(fillColor)
			dc.FillPreserve()
		}

		if strokeColor != nil && strokeWidth > 0 {
			dc.SetColor(strokeColor)
			dc.SetLineWidth(strokeWidth * ((scaleX + scaleY) / 2))
			dc.Stroke()
		}

		return dc.Image()
	})
	raster.Resize(fyne.NewSize(float32(w), float32(h)))
	raster.Move(fyne.NewPos(float32(math.Round(minX)), float32(math.Round(minY))))
	return raster
}

// drawCountry draws the GeoJSON paths of a country on the provided MapWidget.
// It applies scaling, centering, and optionally renders the bounding box for debugging purposes.
func (a *App) drawCountry(zm *MapWidget, country string, scale float64, clear bool, fillColor color.Color, strokeColor color.Color) {
	data, err := FetchAndCacheGeoJSON(country, true, a.Settings.SkipSmall, a.Settings.EnablePacificCenter, a.Settings.MapDataPath, a.GeoCache, a.CountryData)
	if err != nil {
		a.showMessage("Error", fmt.Sprintf("Error loading %s: %v", country, err))
		return
	}

	if len(data.Paths) == 0 {
		if clear {
			zm.Container.Objects = []fyne.CanvasObject{canvas.NewRectangle(ParseHexColor(a.Settings.BackgroundColor))}
			zm.Container.Refresh()
		}
		return
	}

	// Ensure bounding box is updated
	data.UpdateBoundingBox()

	size := zm.Container.Size()
	if size.Width == 0 || size.Height == 0 {
		size = fyne.NewSize(500, 500)
	}
	if a.headerContainer != nil {
		h := a.headerContainer.MinSize().Height
		if size.Height > h {
			size.Height -= h
		} else {
			size.Height = 0
		}
	}

	// Subtract About button height
	if a.cCenter != nil && len(a.cCenter.Objects) > 0 {
		for _, obj := range a.cCenter.Objects {
			if btn, ok := obj.(*widget.Button); ok && btn.Text == "About" {
				h := btn.MinSize().Height
				if size.Height > h {
					size.Height -= h
				} else {
					size.Height = 0
				}
				break
			}
		}
	}

	var objects []fyne.CanvasObject
	if clear {
		objects = []fyne.CanvasObject{canvas.NewRectangle(ParseHexColor(a.Settings.BackgroundColor))}
	} else {
		objects = make([]fyne.CanvasObject, len(zm.Container.Objects))
		copy(objects, zm.Container.Objects)
		if len(objects) == 0 {
			objects = []fyne.CanvasObject{canvas.NewRectangle(ParseHexColor(a.Settings.BackgroundColor))}
		}
	}

	// Use the pre-calculated Mercator bounding box scaled to pixels
	pixelWidth := data.BoundingBox.Width * scale
	pixelHeight := data.BoundingBox.Height * scale

	offsetX := (float64(size.Width) - pixelWidth) / 2
	offsetY := (float64(size.Height) - pixelHeight) / 2

	// Draw bounding box
	if a.Settings.DebugShowBoundary {
		rect := canvas.NewRectangle(color.Transparent)
		rect.StrokeColor = color.NRGBA{R: 255, A: 255}
		rect.StrokeWidth = 1
		rect.Resize(fyne.NewSize(float32(pixelWidth), float32(pixelHeight)))
		rect.Move(fyne.NewPos(float32(offsetX), float32(offsetY)))
		objects = append(objects, rect)
	}

	// Pass 2: Draw the transformed paths
	minXScaled := data.BoundingBox.MinX * scale
	minYScaled := data.BoundingBox.MinY * scale

	for _, path := range data.Paths { // Use original paths for drawing
		var polyPoints []Point
		for _, p := range path {
			// Screen space: (Mercator * scale) - minXScaled
			polyPoints = append(polyPoints, Point{
				X: p.X*scale - minXScaled + offsetX,
				Y: p.Y*scale - minYScaled + offsetY,
			})
		}

		if len(polyPoints) < 3 {
			continue
		}

		var sw float64
		if strokeColor != nil {
			sw = 1.0
		}

		poly := drawFilledPolygon(polyPoints, fillColor, strokeColor, sw)
		objects = append(objects, poly)
	}
	zm.Container.Objects = objects
	zm.Container.Refresh()
}
