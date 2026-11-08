package main

import (
	"flag"
	"fmt"
)

type config struct {
	Verbose   bool
	OutputDir string
}

func parseCommandLine() config {
	verbose := flag.Bool("verbose", false, "enable verbose output")
	outputDir := flag.String("output", "./.output", "output directory")
	flag.Parse()

	return config{
		Verbose:   *verbose,
		OutputDir: *outputDir,
	}
}

func main() {
	cfg := parseCommandLine()

	fmt.Println("Generating the website")
	fmt.Println("Output directory:", cfg.OutputDir)
}
