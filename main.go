package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var (
	CountryData          *CountryCollection
	cCenter              *fyne.Container
	cMap                 *ZoomableMap
	headerContainer      *fyne.Container
	leftBar              *fyne.Container
	rightBar             *fyne.Container
	leftSelectedCountry  binding.String
	rightSelectedCountry binding.String
	AppSettings          Settings
	zoomSlider           *widget.Slider
)

type Settings struct {
	MinScale float32 `json:"minScale"`
	MaxScale float32 `json:"maxScale"`
}

type ZoomableMap struct {
	widget.BaseWidget
	Container *fyne.Container
	ZoomScale binding.Float
	ZoomLevel binding.Int // 0 to 19
	Slider    *widget.Slider
}

func NewZoomableMap() *ZoomableMap {
	zm := &ZoomableMap{
		Container: container.NewWithoutLayout(),
		ZoomScale: binding.NewFloat(),
		ZoomLevel: binding.NewInt(),
	}
	zm.ZoomScale.Set(float64(AppSettings.MinScale))
	zm.ZoomLevel.Set(0)
	zm.ExtendBaseWidget(zm)
	return zm
}

func (zm *ZoomableMap) Scrolled(e *fyne.ScrollEvent) {
	zoomLevel, _ := zm.ZoomLevel.Get()
	if e.Scrolled.DY > 0 {
		if zoomLevel < 19 {
			zoomLevel++
		}
	} else if e.Scrolled.DY < 0 {
		if zoomLevel > 0 {
			zoomLevel--
		}
	}
	zm.ZoomLevel.Set(zoomLevel)

	minScale := AppSettings.MinScale
	maxScale := AppSettings.MaxScale
	newScale := minScale * float32(math.Pow(float64(maxScale/minScale), float64(zoomLevel)/19.0))
	zm.ZoomScale.Set(float64(newScale))

	if zm.Slider != nil {
		zm.Slider.SetValue(float64(zoomLevel))
	}
	updateMapDisplay()
}

func createFooter() fyne.CanvasObject {
	minText := canvas.NewText("0", color.White)
	minText.TextSize = 26
	maxText := canvas.NewText("19", color.White)
	maxText.TextSize = 26
	zoomSlider = widget.NewSlider(0, 19)
	// Force slider height by adding a transparent rectangle to a stack
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.Resize(fyne.NewSize(1, 80))
	sliderWrapper := container.NewStack(spacer, zoomSlider)

	zoomSlider.OnChanged = func(val float64) {
		currentLevel, _ := cMap.ZoomLevel.Get()
		if int(val) == currentLevel {
			return
		}
		newLevel := int(val)
		cMap.ZoomLevel.Set(newLevel)
		minScale := AppSettings.MinScale
		maxScale := AppSettings.MaxScale
		newScale := minScale * float32(math.Pow(float64(maxScale/minScale), float64(newLevel)/19.0))
		cMap.ZoomScale.Set(float64(newScale))
		updateMapDisplay()
	}
	return container.NewBorder(nil, nil, container.NewPadded(minText), container.NewPadded(maxText), sliderWrapper)
}

func (zm *ZoomableMap) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(zm.Container)
}

func init() {
	cc := NewCountryCollection()
	path := filepath.Join("mapdata", "country_data.json")
	if err := cc.SaveToJSON(path); err != nil {
		log.Println("Error saving country_data.json:", err)
	}
	CountryData = cc
	loadSettings()
}

func loadSettings() {
	data, err := os.ReadFile("settings.json")
	if err != nil {
		log.Println("Error reading settings.json, using defaults:", err)
		AppSettings = Settings{MinScale: 0.1, MaxScale: 10.0}
		return
	}
	if err := json.Unmarshal(data, &AppSettings); err != nil {
		log.Println("Error unmarshaling settings.json:", err)
		AppSettings = Settings{MinScale: 0.1, MaxScale: 10.0}
	}
}
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
	return container.NewStack(bg, scroll)
}

func addBorder(obj fyne.CanvasObject) fyne.CanvasObject {
	border := canvas.NewRectangle(color.Transparent)
	border.StrokeColor = color.White
	border.StrokeWidth = 2
	return container.NewStack(obj, border)
}

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

func calculateZoomLimits() (float32, float32) {
	left, _ := leftSelectedCountry.Get()
	right, _ := rightSelectedCountry.Get()
	var selected []string
	if left != "" {
		selected = append(selected, left)
	}
	if right != "" {
		selected = append(selected, right)
	}

	if len(selected) == 0 {
		return 0.1, 10.0
	}

	var scales []float32
	for _, c := range selected {
		scales = append(scales, getFitScale(c))
	}

	minScale := scales[0]
	maxScale := scales[0]
	for _, s := range scales {
		if s < minScale {
			minScale = s
		}
		if s > maxScale {
			maxScale = s
		}
	}
	return minScale * 0.5, maxScale * 2.0
}

func getFitScale(country string) float32 {
	bbox, err := GetBoundingBox(country)
	if err != nil {
		return 1.0
	}
	width := bbox.MaxX - bbox.MinX
	height := bbox.MaxY - bbox.MinY
	if width == 0 || height == 0 {
		return 1.0
	}
	
	size := cMap.Container.Size()
	scaleX := size.Width / width
	scaleY := size.Height / height
	return min(scaleX, scaleY)
}

func getTargetScale() (float32, int) {
	left, _ := leftSelectedCountry.Get()
	right, _ := rightSelectedCountry.Get()

	areaLeft := 0.0
	if left != "" {
		areaLeft = getArea(left)
	}
	areaRight := 0.0
	if right != "" {
		areaRight = getArea(right)
	}

	larger := ""
	if areaLeft >= areaRight {
		larger = left
	} else {
		larger = right
	}

	if larger == "" {
		return AppSettings.MinScale, 0
	}
	
	targetScale := getFitScale(larger) / 20.0
	
	minScale := AppSettings.MinScale
	maxScale := AppSettings.MaxScale
	
	level := int(19.0 * math.Log(float64(targetScale/minScale)) / math.Log(float64(maxScale/minScale)))
	if level < 0 {
		level = 0
	}
	if level > 19 {
		level = 19
	}
	
	// Recalculate scale from level to ensure consistency
	actualScale := minScale * float32(math.Pow(float64(maxScale/minScale), float64(level)/19.0))
	
	return actualScale, level
}

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

func updateMapDisplay() {
	clearAll()
	left, _ := leftSelectedCountry.Get()
	right, _ := rightSelectedCountry.Get()
	if left != "" {
		drawBar(leftBar, getArea(left), color.NRGBA{G: 255, A: 255})
		drawCountry(cMap, left, false, color.NRGBA{G: 255, A: 255})
	}
	if right != "" {
		drawBar(rightBar, getArea(right), color.NRGBA{G: 255, B: 255, A: 255})
		drawCountry(cMap, right, false, color.NRGBA{G: 255, B: 255, A: 255})
	}
}

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

func (c *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return c.base.Color(name, variant)
}
func (c *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return c.base.Font(style)
}
func (c *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return c.base.Icon(name)
}
func (c *customTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNameScrollBar || name == theme.SizeNameScrollBarSmall {
		return 20
	}
	return c.base.Size(name)
}

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

	cMap = NewZoomableMap()
	zoomControl := createFooter()
	cMap.Slider = zoomSlider
	headerContainer = container.NewHBox()
	cCenter = container.NewBorder(container.NewCenter(headerContainer), zoomControl, nil, nil, cMap)

	leftBar = container.NewWithoutLayout()
	leftBarWrapper := container.NewScroll(leftBar)
	leftBarWrapper.SetMinSize(fyne.NewSize(50, 0))
	rightBar = container.NewWithoutLayout()
	rightBarWrapper := container.NewScroll(rightBar)
	rightBarWrapper.SetMinSize(fyne.NewSize(50, 0))

	innerBorder := container.NewBorder(nil, nil, leftBarWrapper, rightBarWrapper, cCenter)

	left := addBorder(createList(maxWidth, func(c string) {
		leftSelectedCountry.Set(c)
		newScale, newLevel := getTargetScale()
		cMap.ZoomScale.Set(float64(newScale))
		cMap.ZoomLevel.Set(newLevel)
		if cMap.Slider != nil {
			cMap.Slider.SetValue(float64(newLevel))
		}
		clearAll()
		updateHeader()
		if c != "" {
			drawBar(leftBar, getArea(c), color.NRGBA{G: 255, A: 255})
			drawCountry(cMap, c, true, color.NRGBA{G: 255, A: 255})
		}
		right, _ := rightSelectedCountry.Get()
		if right != "" {
			drawBar(rightBar, getArea(right), color.NRGBA{G: 255, B: 255, A: 255})
			drawCountry(cMap, right, false, color.NRGBA{G: 255, B: 255, A: 255})
		}
	}))
	right := addBorder(createList(maxWidth, func(c string) {
		rightSelectedCountry.Set(c)
		newScale, newLevel := getTargetScale()
		cMap.ZoomScale.Set(float64(newScale))
		cMap.ZoomLevel.Set(newLevel)
		if cMap.Slider != nil {
			cMap.Slider.SetValue(float64(newLevel))
		}
		clearAll()
		updateHeader()
		var clear = true
		left, _ := leftSelectedCountry.Get()
		if left != "" {
			drawBar(leftBar, getArea(left), color.NRGBA{G: 255, A: 255})
			drawCountry(cMap, left, clear, color.NRGBA{G: 255, A: 255})
			clear = false
		}
		if c != "" {
			drawBar(rightBar, getArea(c), color.NRGBA{G: 255, B: 255, A: 255})
			drawCountry(cMap, c, clear, color.NRGBA{G: 255, B: 255, A: 255})
		}
	}))
	center := addBorder(innerBorder)

	w.SetContent(container.NewBorder(nil, nil, left, right, center))

	w.ShowAndRun()
}

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

func getArea(name string) float64 {
	for _, country := range CountryData.Countries {
		if country.Name == name {
			return country.Area
		}
	}
	return 0
}

func drawCountry(zm *ZoomableMap, country string, clear bool, lineColor color.Color) {
	paths, err := getCachedGeoJSON(country, true, true)
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

	zScale, _ := zm.ZoomScale.Get()
	fixedScale := float32(20.0) * float32(zScale)

	var totalLat, totalLon float32
	var numPoints int
	for _, path := range paths {
		for _, pos := range path {
			totalLat += pos.X
			totalLon += pos.Y
			numPoints++
		}
	}
	centroidLat := totalLat / float32(numPoints)
	centroidLon := totalLon / float32(numPoints)

	canvasCenterX := size.Width / 2
	canvasCenterY := size.Height / 2

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

	for _, path := range paths {
		var points []fyne.Position
		for _, pos := range path {
			points = append(points, fyne.NewPos(
				(pos.Y-centroidLon)*fixedScale+canvasCenterX,
				(centroidLat-pos.X)*fixedScale+canvasCenterY,
			))
		}
		for i := 0; i < len(points)-1; i++ {
			l := &canvas.Line{Position1: points[i], Position2: points[i+1], StrokeColor: lineColor, StrokeWidth: 1}
			objects = append(objects, l)
		}
	}
	zm.Container.Objects = objects
	zm.Container.Refresh()
}
