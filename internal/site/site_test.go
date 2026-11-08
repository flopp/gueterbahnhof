package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateBuildsStaticPages(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	temp := t.TempDir()
	contentPath := filepath.Join(temp, "places.json")
	content := `{
  "places": [{
    "slug": "test-ort",
    "name": "<script>alert(1)</script>",
    "category": "Kultur",
    "address": "Teststraße 1, Freiburg",
    "coordinates": {"latitude": 48.01, "longitude": 7.84},
    "lastVerified": "2026-11-08"
  }, {
    "slug": "ohne-koordinaten",
    "name": "Ohne Koordinaten",
    "category": "Kultur",
    "address": "Teststraße 2, Freiburg"
  }]
}`
	if err := os.WriteFile(contentPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(temp, "out")
	err = Generate(Config{
		OutputDir:          output,
		ContentPath:        contentPath,
		Boundary:           filepath.Join(root, "data", "gueterbahnhof.geojson"),
		TemplateDir:        filepath.Join(root, "templates"),
		StaticDir:          filepath.Join(root, "static"),
		BasePath:           "/quartier",
		TileURL:            "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
		TileAttribution:    "OpenStreetMap-Kartenkacheln",
		TileAttributionURL: "https://www.openstreetmap.org/copyright",
	})
	if err != nil {
		t.Fatal(err)
	}

	checks := map[string][]string{
		"index.html": {
			"lang=\"de\"", "Das Quartier", "/quartier/places/", "/quartier/static/css/site.css",
		},
		"places/index.html": {
			"Teststraße 1", "/quartier/places/test-ort/", "&lt;script&gt;alert(1)&lt;/script&gt;",
		},
		"places/test-ort/index.html": {
			"Angaben zuletzt geprüft: 08.11.2026", "&lt;script&gt;alert(1)&lt;/script&gt;",
			"Standort", "id=\"place-map\"", "data-lat=\"48.01\"", "data-lon=\"7.84\"",
			"/quartier/static/js/place-map.js",
			"OpenStreetMap-Kartenkacheln",
		},
		"places/ohne-koordinaten/index.html": {
			"Ohne Koordinaten",
		},
		"map/index.html": {
			"data-tile-url=\"https://tile.openstreetmap.org/%7bz%7d/%7bx%7d/%7by%7d.png\"",
			"data-lat=\"48.01\"", "data-lon=\"7.84\"",
			"/quartier/data/gueterbahnhof.geojson",
			"/quartier/static/vendor/leaflet/leaflet.js",
			"OpenStreetMap-Kartenkacheln",
		},
		"data/gueterbahnhof.geojson":       {"FeatureCollection"},
		"static/vendor/leaflet/leaflet.js": {"Leaflet"},
	}
	for name, expected := range checks {
		data, err := os.ReadFile(filepath.Join(output, name))
		if err != nil {
			t.Fatalf("read generated %s: %v", name, err)
		}
		for _, snippet := range expected {
			if !strings.Contains(string(data), snippet) {
				t.Errorf("%s does not contain %q", name, snippet)
			}
		}
	}
	unlocatedPage, err := os.ReadFile(filepath.Join(output, "places", "ohne-koordinaten", "index.html"))
	if err != nil {
		t.Fatalf("read generated unlocated place: %v", err)
	}
	for _, unexpected := range []string{"id=\"place-map\"", "leaflet.css", "place-map.js"} {
		if strings.Contains(string(unlocatedPage), unexpected) {
			t.Errorf("unlocated place page unexpectedly contains %q", unexpected)
		}
	}
}

func TestLoadPlacesRejectsInvalidData(t *testing.T) {
	temp := t.TempDir()
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "missing required field",
			content: `{"places":[{"slug":"place","name":"Ort","category":"Kultur"}]}`,
			want:    "address is required",
		},
		{
			name: "duplicate slug",
			content: `{"places":[
				{"slug":"place","name":"Ort","category":"Kultur","address":"A"},
				{"slug":"place","name":"Ort 2","category":"Kultur","address":"B"}
			]}`,
			want: "duplicate slug",
		},
		{
			name:    "invalid coordinates",
			content: `{"places":[{"slug":"place","name":"Ort","category":"Kultur","address":"A","coordinates":{"latitude":91,"longitude":7}}]}`,
			want:    "outside WGS 84",
		},
		{
			name:    "unknown field",
			content: `{"places":[{"slug":"place","name":"Ort","category":"Kultur","address":"A","unknown":true}]}`,
			want:    "unknown field",
		},
		{
			name:    "invalid URL",
			content: `{"places":[{"slug":"place","name":"Ort","category":"Kultur","address":"A","website":"javascript:alert(1)"}]}`,
			want:    "absolute HTTP(S) URL",
		},
		{
			name:    "missing image",
			content: `{"places":[{"slug":"place","name":"Ort","category":"Kultur","address":"A","images":[{"src":"images/missing.jpg","alt":"Fehlendes Bild"}]}]}`,
			want:    "no such file",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			contentPath := filepath.Join(temp, strings.ReplaceAll(test.name, " ", "-")+".json")
			if err := os.WriteFile(contentPath, []byte(test.content), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := loadPlaces(contentPath, temp)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("loadPlaces() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestValidateTileURL(t *testing.T) {
	if err := validateTileURL("http://tiles.example/{z}/{x}/{y}.png"); err == nil {
		t.Fatal("validateTileURL accepted a non-HTTPS URL")
	}
	if err := validateTileURL("https://tiles.example/{z}/{x}/{y}.png"); err != nil {
		t.Fatalf("validateTileURL rejected HTTPS URL template: %v", err)
	}
}

func TestPathsOverlap(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "output")
	source := filepath.Join(root, "static")
	inside := filepath.Join(output, "static")
	overlaps, err := pathsOverlap(output, inside)
	if err != nil || !overlaps {
		t.Fatalf("pathsOverlap(output, inside) = %v, %v; want true", overlaps, err)
	}
	overlaps, err = pathsOverlap(output, source)
	if err != nil || overlaps {
		t.Fatalf("pathsOverlap(output, source) = %v, %v; want false", overlaps, err)
	}
}

func TestValidateBoundaryRejectsUnclosedRing(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "boundary.geojson")
	content := `{"type":"FeatureCollection","features":[{"type":"Feature","geometry":{"type":"Polygon","coordinates":[[[0,0],[1,0],[1,1],[0,1]]]}}]}`
	if err := os.WriteFile(filename, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateBoundary(filename); err == nil || !strings.Contains(err.Error(), "must have at least four positions and be closed") {
		t.Fatalf("validateBoundary() error = %v, want unclosed ring error", err)
	}
}

func TestNormalizeBasePath(t *testing.T) {
	tests := []struct {
		input string
		want  string
		fail  bool
	}{
		{input: "/", want: ""},
		{input: "/site", want: "/site"},
		{input: "site", fail: true},
		{input: "/site/", fail: true},
		{input: "/../other", fail: true},
	}
	for _, test := range tests {
		got, err := normalizeBasePath(test.input)
		if test.fail {
			if err == nil {
				t.Errorf("normalizeBasePath(%q) succeeded, want error", test.input)
			}
			continue
		}
		if err != nil || got != test.want {
			t.Errorf("normalizeBasePath(%q) = %q, %v; want %q", test.input, got, err, test.want)
		}
	}
}
