package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CountryData holds the global collection of country information.
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

// Settings define the scaling limits for the application.
type Settings struct {
	MinScale          float32 `json:"minScale"`
	MaxScale          float32 `json:"maxScale"`
	DebugShowBoundary bool    `json:"debug_show_boundary"`
}

// MapWidget is a custom widget that provides a map interface.
type MapWidget struct {
	widget.BaseWidget
	Container *fyne.Container
}

// NewMapWidget creates and initializes a new MapWidget.
func NewMapWidget() *MapWidget {
	zm := &MapWidget{
		Container: container.NewWithoutLayout(),
	}
	zm.ExtendBaseWidget(zm)
	return zm
}

// CreateRenderer creates a renderer for the MapWidget.
func (zm *MapWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(zm.Container)
}

// Resize handles resizing of the MapWidget and redraws the map.
func (zm *MapWidget) Resize(s fyne.Size) {
	zm.BaseWidget.Resize(s)
	if cMap != nil && leftBar != nil && rightBar != nil && headerContainer != nil {
		updateMapDisplay()
	}
}

// init initializes the country collection and application settings.
func init() {
	cc := NewCountryCollection()
	//path := filepath.Join("mapdata", "country_data.json")
	//if err := cc.SaveToJSON(path); err != nil {
	//	log.Println("Error saving country_data.json:", err)
	//}
	CountryData = cc
	loadSettings()
}

// loadSettings reads application settings from settings.json.
func loadSettings() {
	data, err := os.ReadFile("settings.json")
	if err != nil {
		log.Println("Error reading settings.json, using defaults:", err)
		AppSettings = Settings{MinScale: 0.1, MaxScale: 10.0, DebugShowBoundary: false}
		return
	}
	if err := json.Unmarshal(data, &AppSettings); err != nil {
		log.Println("Error unmarshaling settings.json:", err)
		AppSettings = Settings{MinScale: 0.1, MaxScale: 10.0, DebugShowBoundary: false}
	}
}

// createList creates a scrollable list of countries with a selection callback.
func createList(width float32, onSelected func(string)) fyne.CanvasObject {
	list := widget.NewList(
		func() int { return len(CountryData.Countries) },
		func() fyne.CanvasObject {
			text := canvas.NewText("", color.White)
			text.TextSize = 28
			return container.NewPadded(text)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			padded := obj.(*fyne.Container)
			text := padded.Objects[0].(*canvas.Text)
			text.Text = CountryData.Countries[id].Name
			text.Refresh()
		},
	)
	var selectedID widget.ListItemID = -1
	list.OnSelected = func(id widget.ListItemID) {
		if id == selectedID {
			list.Unselect(id)
			selectedID = -1
			onSelected("")
		} else {
			selectedID = id
			onSelected(CountryData.Countries[id].Name)
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

	return container.NewBorder(nil, button, nil, nil, container.NewStack(bg, scroll))
}

// addBorder adds a border to a Fyne canvas object.
func addBorder(obj fyne.CanvasObject) fyne.CanvasObject {
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = color.White
	border.StrokeWidth = 2
	return container.NewStack(obj, border)
}

// formatNumber formats a float as a string with thousands separators.
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

// getFitScale returns the scale needed to fit the bounding box of a country.
func getFitScale(country string) float32 {
	minX, minY, maxX, maxY, err := GetMercatorBoundingBox(country)
	if err != nil {
		return 1.0
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

// getTargetScale calculates the appropriate scale for displaying selected countries.
func getTargetScale(active, other string) float32 {
	if active == "" && other == "" {
		return 1.0
	}
	if active == "" {
		return getFitScale(other)
	}
	if other == "" {
		return getFitScale(active)
	}

	minXActive, minYActive, maxXActive, maxYActive, errActive := GetMercatorBoundingBox(active)
	minXOther, minYOther, maxXOther, maxYOther, errOther := GetMercatorBoundingBox(other)

	if errActive != nil && errOther != nil {
		return 1.0
	}
	if errActive != nil {
		return getFitScale(other)
	}
	if errOther != nil {
		return getFitScale(active)
	}

	if cMap == nil {
		return 1.0
	}
	size := cMap.Container.Size()
	width := float64(size.Width)
	height := float64(size.Height)

	areaActive := (maxXActive - minXActive) * width * (maxYActive - minYActive) * height
	areaOther := (maxXOther - minXOther) * width * (maxYOther - minYOther) * height

	if areaActive > areaOther {
		return getFitScale(active)
	}
	return getFitScale(other)
}

// updateHeader updates the header displaying area information for selected countries.
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
		t := canvas.NewText(leftPart, color.NRGBA{G: 255, A: 255})
		t.TextSize = 36
		headerContainer.Add(t)
	}
	if sep != "" {
		t := canvas.NewText(sep, color.White)
		t.TextSize = 36
		headerContainer.Add(t)
	}
	if rightPart != "" {
		t := canvas.NewText(rightPart, color.NRGBA{G: 255, B: 255, A: 255})
		t.TextSize = 36
		headerContainer.Add(t)
	}
	headerContainer.Refresh()
}

// updateMapDisplay clears and redraws the map based on current selections.
func updateMapDisplay() {
	clearAll()
	left, _ := leftSelectedCountry.Get()
	right, _ := rightSelectedCountry.Get()
	scale := getTargetScale(left, right)
	if left != "" {
		drawBar(leftBar, getArea(left), color.NRGBA{G: 255, A: 255})
		drawCountry(cMap, left, scale, false, color.NRGBA{G: 255, A: 255})
	}
	if right != "" {
		drawBar(rightBar, getArea(right), color.NRGBA{G: 255, B: 255, A: 255})
		drawCountry(cMap, right, scale, false, color.NRGBA{G: 255, B: 255, A: 255})
	}
}

// clearAll clears all containers on the map.
func clearAll() {
	leftBar.Objects = nil
	leftBar.Refresh()
	rightBar.Objects = nil
	rightBar.Refresh()
	cMap.Container.Objects = []fyne.CanvasObject{canvas.NewRectangle(color.Black)}
	cMap.Container.Refresh()
}

type customTheme struct {
	base fyne.Theme
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

// main is the application entry point.
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
		size := fyne.MeasureText(country.Name, 28, fyne.TextStyle{})
		if size.Width > maxWidth {
			maxWidth = size.Width
		}
	}
	maxWidth += 6

	cMap = NewMapWidget()
	headerContainer = container.NewHBox()
	cCenter = container.NewBorder(container.NewCenter(headerContainer), nil, nil, nil, cMap)

	leftBar = container.NewWithoutLayout()
	leftBarWrapper := container.NewScroll(leftBar)
	leftBarWrapper.SetMinSize(fyne.NewSize(50, 0))
	rightBar = container.NewWithoutLayout()
	rightBarWrapper := container.NewScroll(rightBar)
	rightBarWrapper.SetMinSize(fyne.NewSize(50, 0))

	innerBorder := container.NewBorder(nil, nil, leftBarWrapper, rightBarWrapper, cCenter)

	left := addBorder(createList(maxWidth, func(c string) {
		err := leftSelectedCountry.Set(c)
		if err != nil {
			return
		}
		clearAll()
		updateHeader()

		rightStr, _ := rightSelectedCountry.Get()
		scale := getTargetScale(c, rightStr)

		if c != "" {
			drawBar(leftBar, getArea(c), color.NRGBA{G: 255, A: 255})
			drawCountry(cMap, c, scale, true, color.NRGBA{G: 255, A: 255})
		}
		if rightStr != "" {
			drawBar(rightBar, getArea(rightStr), color.NRGBA{G: 255, B: 255, A: 255})
			drawCountry(cMap, rightStr, scale, false, color.NRGBA{G: 255, B: 255, A: 255})
		}
	}))
	right := addBorder(createList(maxWidth, func(c string) {
		err := rightSelectedCountry.Set(c)
		if err != nil {
			return
		}
		clearAll()
		updateHeader()

		leftStr, _ := leftSelectedCountry.Get()
		scale := getTargetScale(c, leftStr)
		var doClear = true

		if leftStr != "" {
			drawBar(leftBar, getArea(leftStr), color.NRGBA{G: 255, A: 255})
			drawCountry(cMap, leftStr, scale, doClear, color.NRGBA{G: 255, A: 255})
			doClear = false
		}
		if c != "" {
			drawBar(rightBar, getArea(c), color.NRGBA{G: 255, B: 255, A: 255})
			drawCountry(cMap, c, scale, doClear, color.NRGBA{G: 255, B: 255, A: 255})
		}
	}))
	center := addBorder(innerBorder)

	w.SetContent(container.NewBorder(nil, nil, left, right, center))

	w.ShowAndRun()
}

// drawBar draws a bar representing the area of a country.
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

// getArea retrieves the area of a country by its name.
func getArea(name string) float64 {
	for _, country := range CountryData.Countries {
		if country.Name == name {
			return country.Area
		}
	}
	return 0
}

func getCountryInfo(name string) *CountryInfo {
	for i := range CountryData.Countries {
		if CountryData.Countries[i].Name == name {
			return &CountryData.Countries[i]
		}
	}
	return nil
}

// drawCountry draws the geoJSON paths of a country on the map.
func drawCountry(zm *MapWidget, country string, scale float32, clear bool, lineColor color.Color) {
	paths, err := GetCachedGeoJSON(country, true)
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

	minX, minY, maxX, maxY, err := GetMercatorBoundingBox(country)
	if err != nil {
		log.Printf("Error getting bbox for %s: %v", country, err)
		return
	}

	info := getCountryInfo(country)
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2

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
	transformedMinX, transformedMinY := math.MaxFloat64, math.MaxFloat64
	transformedMaxX, transformedMaxY := -math.MaxFloat64, -math.MaxFloat64

	for i, path := range paths {
		transformedPaths[i] = make([]fyne.Position, len(path))
		for j, pos := range path {
			mx, my := LatLonToMercator(float64(pos.X), float64(pos.Y))

			if info != nil {
				if info.Rotate != 0 {
					angle := float64(info.Rotate) * math.Pi / 180.0
					cosTheta := math.Cos(angle)
					sinTheta := math.Sin(angle)

					// Shift to origin
					tx := mx - centerX
					ty := my - centerY

					// Rotate
					nx := tx*cosTheta - ty*sinTheta
					ny := tx*sinTheta + ty*cosTheta

					// Shift back
					mx = nx + centerX
					my = ny + centerY
				}

				if info.Flip_Y {
					my = (minY + maxY) - my
				}
			}

			if mx < transformedMinX {
				transformedMinX = mx
			}
			if mx > transformedMaxX {
				transformedMaxX = mx
			}
			if my < transformedMinY {
				transformedMinY = my
			}
			if my > transformedMaxY {
				transformedMaxY = my
			}
			transformedPaths[i][j] = fyne.NewPos(float32(mx), float32(my))
		}
	}

	transformedWidth := transformedMaxX - transformedMinX
	transformedHeight := transformedMaxY - transformedMinY
	offsetX := (size.Width - float32(transformedWidth)*scale) / 2
	offsetY := (size.Height - float32(transformedHeight)*scale) / 2

	pixelMinX := offsetX
	pixelMinY := offsetY
	pixelMaxX := offsetX + float32(transformedMaxX-transformedMinX)*scale
	pixelMaxY := offsetY + float32(transformedMaxY-transformedMinY)*scale
	log.Printf("Logging info for %s: Screen size: %+v, Bounding box in pixels: minX=%f, minY=%f, maxX=%f, maxY=%f", country, size, pixelMinX, pixelMinY, pixelMaxX, pixelMaxY)

	// Draw bounding box
	if AppSettings.DebugShowBoundary {
		rect := canvas.NewRectangle(color.Transparent)
		rect.StrokeColor = color.NRGBA{R: 255, A: 255}
		rect.StrokeWidth = 1
		rect.Resize(fyne.NewSize(pixelMaxX-pixelMinX, pixelMaxY-pixelMinY))
		rect.Move(fyne.NewPos(pixelMinX, pixelMinY))
		objects = append(objects, rect)
	}

	// Pass 2: Draw the transformed paths
	for _, path := range transformedPaths {
		var prevPoint fyne.Position
		for i, p := range path {
			// Screen space
			screenX := float32(float64(p.X)-transformedMinX) * scale
			screenY := float32(float64(p.Y)-transformedMinY) * scale

			// Apply centering
			pPos := fyne.NewPos(screenX+offsetX, screenY+offsetY)

			if i > 0 {
				line := canvas.NewLine(lineColor)
				line.StrokeWidth = 1
				line.Position1 = prevPoint
				line.Position2 = pPos
				objects = append(objects, line)
			}
			prevPoint = pPos
		}
	}
	zm.Container.Objects = objects
	zm.Container.Refresh()
}
