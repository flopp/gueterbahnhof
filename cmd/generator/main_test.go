package main

import (
	"bytes"
	"flag"
	"strings"
	"testing"
)

func TestParseCommandLine(t *testing.T) {
	var help bytes.Buffer
	cfg, err := parseCommandLine([]string{
		"-verbose", "-output", "build", "-base-path", "/quartier",
		"-tile-url", "https://tiles.example/{z}/{x}/{y}.png",
	}, &help)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Verbose || cfg.OutputDir != "build" || cfg.BasePath != "/quartier" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.TileURL != "https://tiles.example/{z}/{x}/{y}.png" {
		t.Fatalf("tile URL = %q", cfg.TileURL)
	}
}

func TestParseCommandLineHelp(t *testing.T) {
	var help bytes.Buffer
	_, err := parseCommandLine([]string{"-h"}, &help)
	if err != flag.ErrHelp {
		t.Fatalf("parseCommandLine(-h) error = %v, want flag.ErrHelp", err)
	}
	if !strings.Contains(help.String(), "-boundary") || !strings.Contains(help.String(), "-tile-url") {
		t.Fatalf("help output is missing expected flags: %s", help.String())
	}
}
