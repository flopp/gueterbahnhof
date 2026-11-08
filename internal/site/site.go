package site

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Config struct {
	OutputDir          string
	ContentPath        string
	Boundary           string
	TemplateDir        string
	StaticDir          string
	BasePath           string
	TileURL            string
	TileAttribution    string
	TileAttributionURL string
	Verbose            bool
	Log                io.Writer
}

type Dataset struct {
	Places []Place `json:"places"`
}

type Place struct {
	Slug         string      `json:"slug"`
	Name         string      `json:"name"`
	Category     string      `json:"category"`
	Description  string      `json:"description,omitempty"`
	Address      string      `json:"address"`
	OpeningHours []string    `json:"openingHours,omitempty"`
	Website      string      `json:"website,omitempty"`
	Phone        string      `json:"phone,omitempty"`
	Images       []Image     `json:"images,omitempty"`
	Coordinates  *Coordinate `json:"coordinates,omitempty"`
	Source       string      `json:"source,omitempty"`
	LastVerified string      `json:"lastVerified,omitempty"`
}

type Image struct {
	Src  string `json:"src"`
	Alt  string `json:"alt"`
	Path string `json:"-"`
}

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type featureCollection struct {
	Type     string    `json:"type"`
	Features []feature `json:"features"`
}

type feature struct {
	Type     string   `json:"type"`
	Geometry geometry `json:"geometry"`
}

type geometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type polygonCoordinates [][][2]float64

type pageData struct {
	BasePath           string
	Title              string
	Description        string
	MapPage            bool
	PlaceMap           bool
	TileURL            string
	TileAttribution    string
	TileAttributionURL string
	Content            template.HTML
	Places             []placeView
	Groups             []placeGroup
	Place              placeView
}

type placeView struct {
	Place
	URL                string
	Index              string
	HasCoordinates     bool
	Latitude           float64
	Longitude          float64
	PhoneHref          string
	LastVerifiedGerman string
}

type placeGroup struct {
	Name       string
	CountLabel string
	Places     []placeView
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var phonePattern = regexp.MustCompile(`^[0-9+() /-]+$`)

func Generate(cfg Config) error {
	if cfg.OutputDir == "" || cfg.ContentPath == "" || cfg.Boundary == "" || cfg.TemplateDir == "" || cfg.StaticDir == "" {
		return errors.New("output, content, boundary, template, and static paths are required")
	}
	basePath, err := normalizeBasePath(cfg.BasePath)
	if err != nil {
		return err
	}
	if err := validateTileURL(cfg.TileURL); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.TileAttribution) == "" || !validHTTPSURL(cfg.TileAttributionURL) {
		return errors.New("tile attribution text and an absolute HTTPS attribution URL are required")
	}
	for _, source := range []string{cfg.ContentPath, cfg.Boundary, cfg.TemplateDir, cfg.StaticDir} {
		overlaps, err := pathsOverlap(cfg.OutputDir, source)
		if err != nil {
			return fmt.Errorf("resolve output and source paths: %w", err)
		}
		if overlaps {
			return fmt.Errorf("output directory %q must not overlap source path %q", cfg.OutputDir, source)
		}
	}

	places, err := loadPlaces(cfg.ContentPath, cfg.StaticDir)
	if err != nil {
		return err
	}
	if err := validateBoundary(cfg.Boundary); err != nil {
		return err
	}

	tmpl, err := template.New("").ParseGlob(filepath.Join(cfg.TemplateDir, "*.tmpl"))
	if err != nil {
		return fmt.Errorf("load templates: %w", err)
	}
	if err := copyStatic(cfg.StaticDir, filepath.Join(cfg.OutputDir, "static")); err != nil {
		return err
	}
	if err := copyFile(cfg.Boundary, filepath.Join(cfg.OutputDir, "data", "gueterbahnhof.geojson")); err != nil {
		return fmt.Errorf("copy neighborhood boundary: %w", err)
	}

	views := make([]placeView, 0, len(places))
	for i, place := range places {
		view := makePlaceView(place, i+1, basePath)
		views = append(views, view)
		if err := renderPage(tmpl, "place-content", filepath.Join(cfg.OutputDir, "places", place.Slug, "index.html"), pageData{
			BasePath:           basePath,
			Title:              place.Name,
			Description:        place.Name + " im Quartier Güterbahnhof in Freiburg.",
			PlaceMap:           view.HasCoordinates,
			TileURL:            cfg.TileURL,
			TileAttribution:    cfg.TileAttribution,
			TileAttributionURL: cfg.TileAttributionURL,
			Place:              view,
		}); err != nil {
			return err
		}
	}

	groups := groupPlaces(views)
	pages := []struct {
		name     string
		template string
		file     string
		data     pageData
	}{
		{
			name: "Startseite", template: "home-content",
			file: "index.html",
			data: pageData{
				BasePath: basePath, Title: "Das Quartier entdecken",
				Description: "Orte, Adressen und Orientierung im Quartier Güterbahnhof in Freiburg.",
				Places:      firstPlaces(views, 3),
			},
		},
		{
			name: "Orte", template: "places-content",
			file: filepath.Join("places", "index.html"),
			data: pageData{
				BasePath: basePath, Title: "Orte im Quartier",
				Description: "Geschäfte, Gastronomie und Angebote im Quartier Güterbahnhof.",
				Places:      views, Groups: groups,
			},
		},
		{
			name: "Karte", template: "map-content",
			file: filepath.Join("map", "index.html"),
			data: pageData{
				BasePath: basePath, Title: "Quartierskarte",
				Description: "Interaktive Karte des Quartiers Güterbahnhof in Freiburg.",
				MapPage:     true, TileURL: cfg.TileURL,
				TileAttribution: cfg.TileAttribution, TileAttributionURL: cfg.TileAttributionURL,
				Places: views,
			},
		},
	}
	for _, page := range pages {
		if cfg.Verbose && cfg.Log != nil {
			fmt.Fprintf(cfg.Log, "Erzeuge %s\n", page.name)
		}
		if err := renderPage(tmpl, page.template, filepath.Join(cfg.OutputDir, page.file), page.data); err != nil {
			return err
		}
	}
	if cfg.Verbose && cfg.Log != nil {
		fmt.Fprintf(cfg.Log, "%d Orte verarbeitet; Ausgabe: %s\n", len(places), cfg.OutputDir)
	}
	return nil
}

func loadPlaces(contentPath, staticDir string) ([]Place, error) {
	file, err := os.Open(contentPath)
	if err != nil {
		return nil, fmt.Errorf("open places file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var dataset Dataset
	if err := decoder.Decode(&dataset); err != nil {
		return nil, fmt.Errorf("decode places file: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, fmt.Errorf("decode places file: %w", err)
	}

	seen := make(map[string]struct{}, len(dataset.Places))
	for i := range dataset.Places {
		place := &dataset.Places[i]
		label := fmt.Sprintf("place %d", i+1)
		if place.Slug != "" {
			label = fmt.Sprintf("place %q", place.Slug)
		}
		if !slugPattern.MatchString(place.Slug) {
			return nil, fmt.Errorf("%s: slug must contain lowercase letters, digits, and single hyphens", label)
		}
		if _, ok := seen[place.Slug]; ok {
			return nil, fmt.Errorf("%s: duplicate slug", label)
		}
		seen[place.Slug] = struct{}{}
		for field, value := range map[string]string{
			"name": place.Name, "category": place.Category, "address": place.Address,
		} {
			if strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("%s: %s is required", label, field)
			}
		}
		if place.Website != "" && !validHTTPURL(place.Website) {
			return nil, fmt.Errorf("%s: website must be an absolute HTTP(S) URL", label)
		}
		if place.Phone != "" && !phonePattern.MatchString(place.Phone) {
			return nil, fmt.Errorf("%s: phone contains unsupported characters", label)
		}
		if place.Coordinates != nil && (place.Coordinates.Latitude < -90 || place.Coordinates.Latitude > 90 ||
			place.Coordinates.Longitude < -180 || place.Coordinates.Longitude > 180) {
			return nil, fmt.Errorf("%s: coordinates are outside WGS 84 latitude/longitude ranges", label)
		}
		if place.LastVerified != "" {
			if _, err := time.Parse("2006-01-02", place.LastVerified); err != nil {
				return nil, fmt.Errorf("%s: lastVerified must use YYYY-MM-DD format", label)
			}
		}
		for j := range place.Images {
			image := &place.Images[j]
			imagePath, err := validateImagePath(staticDir, image.Src)
			if err != nil {
				return nil, fmt.Errorf("%s image %d: %w", label, j+1, err)
			}
			if strings.TrimSpace(image.Alt) == "" {
				return nil, fmt.Errorf("%s image %d: alt text is required", label, j+1)
			}
			image.Path = imagePath
		}
	}
	sort.Slice(dataset.Places, func(i, j int) bool {
		left, right := dataset.Places[i], dataset.Places[j]
		if left.Category != right.Category {
			return left.Category < right.Category
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.Slug < right.Slug
	})
	return dataset.Places, nil
}

func validateBoundary(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open neighborhood boundary: %w", err)
	}
	defer file.Close()
	var collection featureCollection
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&collection); err != nil {
		return fmt.Errorf("decode neighborhood boundary: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return fmt.Errorf("decode neighborhood boundary: %w", err)
	}
	if collection.Type != "FeatureCollection" || len(collection.Features) == 0 {
		return errors.New("neighborhood boundary must be a non-empty GeoJSON FeatureCollection")
	}
	for i, item := range collection.Features {
		if item.Type != "Feature" || item.Geometry.Type != "Polygon" {
			return fmt.Errorf("neighborhood boundary feature %d must be a Polygon", i+1)
		}
		var polygon polygonCoordinates
		if err := json.Unmarshal(item.Geometry.Coordinates, &polygon); err != nil {
			return fmt.Errorf("neighborhood boundary feature %d: decode polygon: %w", i+1, err)
		}
		if len(polygon) == 0 {
			return fmt.Errorf("neighborhood boundary feature %d has no rings", i+1)
		}
		for ringIndex, ring := range polygon {
			if len(ring) < 4 || ring[0] != ring[len(ring)-1] {
				return fmt.Errorf("neighborhood boundary feature %d ring %d must have at least four positions and be closed", i+1, ringIndex+1)
			}
			for _, position := range ring {
				lon, lat := position[0], position[1]
				if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
					return fmt.Errorf("neighborhood boundary feature %d contains out-of-range WGS 84 coordinates", i+1)
				}
			}
		}
	}
	return nil
}

func makePlaceView(place Place, index int, basePath string) placeView {
	view := placeView{
		Place: place,
		URL:   fmt.Sprintf("%s/places/%s/", basePath, place.Slug),
		Index: fmt.Sprintf("%02d", index),
		PhoneHref: strings.NewReplacer(" ", "", "-", "", "(", "", ")", "", "/", "").
			Replace(place.Phone),
	}
	if place.Coordinates != nil {
		view.HasCoordinates = true
		view.Latitude = place.Coordinates.Latitude
		view.Longitude = place.Coordinates.Longitude
	}
	if place.LastVerified != "" {
		date, _ := time.Parse("2006-01-02", place.LastVerified)
		view.LastVerifiedGerman = date.Format("02.01.2006")
	}
	return view
}

func groupPlaces(places []placeView) []placeGroup {
	groups := make([]placeGroup, 0)
	byCategory := make(map[string]int)
	for _, place := range places {
		index, ok := byCategory[place.Category]
		if !ok {
			index = len(groups)
			byCategory[place.Category] = index
			groups = append(groups, placeGroup{Name: place.Category})
		}
		groups[index].Places = append(groups[index].Places, place)
		groups[index].CountLabel = fmt.Sprintf("%02d ORTE", len(groups[index].Places))
	}
	return groups
}

func firstPlaces(places []placeView, count int) []placeView {
	if len(places) > count {
		return places[:count]
	}
	return places
}

func renderPage(tmpl *template.Template, contentTemplate, filename string, data pageData) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	var content bytes.Buffer
	if err := tmpl.ExecuteTemplate(&content, contentTemplate, data); err != nil {
		return fmt.Errorf("render %s: %w", filename, err)
	}
	data.Content = template.HTML(content.String())
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout", data); err != nil {
		return fmt.Errorf("render %s: %w", filename, err)
	}
	if err := os.WriteFile(filename, rendered.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", filename, err)
	}
	return nil
}

func copyStatic(source, destination string) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create static output directory: %w", err)
	}
	return filepath.WalkDir(source, func(filename string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("read static assets: %w", walkErr)
		}
		relative, err := filepath.Rel(source, filename)
		if err != nil {
			return fmt.Errorf("resolve static asset path: %w", err)
		}
		if relative == "." {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("static asset %q must not be a symbolic link", relative)
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(filename, target)
	})
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", source)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func validateImagePath(staticDir, src string) (string, error) {
	normalized := strings.TrimPrefix(src, "/")
	if normalized == "" || strings.HasPrefix(src, "//") || strings.ContainsAny(src, `\?#:%`) ||
		path.IsAbs(normalized) || path.Clean(normalized) != normalized || normalized == "." {
		return "", errors.New("src must be a safe path within the static directory")
	}
	filename := filepath.Join(staticDir, filepath.FromSlash(normalized))
	info, err := os.Stat(filename)
	if err != nil {
		return "", fmt.Errorf("image %q: %w", src, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("image %q is not a regular file", src)
	}
	return filepath.ToSlash(normalized), nil
}

func validHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func validHTTPSURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme == "https" && parsed.Host != ""
}

func validateTileURL(value string) error {
	if value == "" {
		return errors.New("tile URL must not be empty")
	}
	probe := strings.NewReplacer("{z}", "1", "{x}", "1", "{y}", "1", "{s}", "a").Replace(value)
	parsed, err := url.ParseRequestURI(probe)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return errors.New("tile URL must be an absolute HTTPS URL template")
	}
	return nil
}

func normalizeBasePath(value string) (string, error) {
	if value == "" || value == "/" {
		return "", nil
	}
	if !strings.HasPrefix(value, "/") || strings.HasSuffix(value, "/") || strings.Contains(value, "//") {
		return "", errors.New("base path must begin with / and must not end with /")
	}
	clean := path.Clean(value)
	if clean != value || clean == "." || strings.Contains(value, "..") || strings.ContainsAny(value, `?#\`) {
		return "", errors.New("base path must be a clean URL path")
	}
	return value, nil
}

func pathsOverlap(first, second string) (bool, error) {
	firstAbs, err := filepath.Abs(first)
	if err != nil {
		return false, err
	}
	secondAbs, err := filepath.Abs(second)
	if err != nil {
		return false, err
	}
	return pathContains(firstAbs, secondAbs) || pathContains(secondAbs, firstAbs), nil
}

func pathContains(parent, child string) bool {
	relative, err := filepath.Rel(parent, child)
	return err == nil && (relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))))
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values are not allowed")
		}
		return err
	}
	return nil
}
