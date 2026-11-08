package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/flopp/gueterbahnhof/internal/site"
)

type config struct {
	Verbose            bool
	OutputDir          string
	ContentPath        string
	Boundary           string
	TemplateDir        string
	StaticDir          string
	BasePath           string
	TileURL            string
	TileAttribution    string
	TileAttributionURL string
}

func parseCommandLine(args []string, output io.Writer) (config, error) {
	cfg := config{}
	flags := flag.NewFlagSet("generator", flag.ContinueOnError)
	flags.SetOutput(output)
	flags.BoolVar(&cfg.Verbose, "verbose", false, "enable verbose output")
	flags.StringVar(&cfg.OutputDir, "output", "./.output", "output directory")
	flags.StringVar(&cfg.ContentPath, "content", "./content/places.json", "places JSON file")
	flags.StringVar(&cfg.Boundary, "boundary", "./data/gueterbahnhof.geojson", "neighborhood GeoJSON boundary")
	flags.StringVar(&cfg.TemplateDir, "templates", "./templates", "HTML template directory")
	flags.StringVar(&cfg.StaticDir, "static", "./static", "static asset directory")
	flags.StringVar(&cfg.BasePath, "base-path", "/", "URL base path for deployment")
	flags.StringVar(&cfg.TileURL, "tile-url", "https://tile.openstreetmap.org/{z}/{x}/{y}.png", "OpenStreetMap-compatible tile URL template")
	flags.StringVar(&cfg.TileAttribution, "tile-attribution", "OpenStreetMap-Kartenkacheln", "visible tile provider attribution")
	flags.StringVar(&cfg.TileAttributionURL, "tile-attribution-url", "https://www.openstreetmap.org/copyright", "tile provider attribution link")
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("unexpected positional arguments: %v", flags.Args())
	}
	return cfg, nil
}

func main() {
	cfg, err := parseCommandLine(os.Args[1:], os.Stderr)
	if errors.Is(err, flag.ErrHelp) {
		return
	}
	if err == nil {
		err = site.Generate(site.Config{
			OutputDir:          cfg.OutputDir,
			ContentPath:        cfg.ContentPath,
			Boundary:           cfg.Boundary,
			TemplateDir:        cfg.TemplateDir,
			StaticDir:          cfg.StaticDir,
			BasePath:           cfg.BasePath,
			TileURL:            cfg.TileURL,
			TileAttribution:    cfg.TileAttribution,
			TileAttributionURL: cfg.TileAttributionURL,
			Verbose:            cfg.Verbose,
			Log:                os.Stdout,
		})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "generator:", err)
		os.Exit(1)
	}
}
