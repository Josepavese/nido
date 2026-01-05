package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/registry/builder"
)

func main() {
	sourcesFile := flag.String("sources", "registry/sources.yaml", "Path to sources configuration")
	outputFile := flag.String("output", "registry/images.json", "Path to output JSON")
	flag.Parse()

	// 1. Read Sources
	data, err := os.ReadFile(*sourcesFile)
	if err != nil {
		fmt.Printf("❌ Failed to read sources: %v\n", err)
		os.Exit(1)
	}

	var config builder.SourcesConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("❌ Failed to parse YAML: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("🔍 Loaded %d sources from %s\n", len(config.Sources), *sourcesFile)

	// 2. Fetch Images
	catalog := &image.Catalog{
		SchemaVersion: "1",
		UpdatedAt:     time.Now().UTC(),
		Images:        []image.Image{},
	}

	for _, source := range config.Sources {
		fmt.Printf("Processing %s...\n", source.Name)

		imgEntry := image.Image{
			Name:        source.Name,
			Registry:    "official", // TODO: make configurable?
			Description: source.Description,
			Homepage:    source.Homepage,
			Versions:    []image.Version{},
		}

		for _, strat := range source.Strategies {
			vers, err := builder.Fetch(source, strat)
			if err != nil {
				fmt.Printf("  ⚠️ Strategy %s failed: %v\n", strat.Type, err)
				continue
			}
			imgEntry.Versions = append(imgEntry.Versions, vers...)
			fmt.Printf("  ✅ Added %d versions via %s\n", len(vers), strat.Type)
		}

		if len(imgEntry.Versions) > 0 {
			catalog.Images = append(catalog.Images, imgEntry)
		} else {
			fmt.Printf("  ⚠️ Skipping %s (no versions found)\n", source.Name)
		}
	}

	// 3. Write Output
	outputData, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFile, outputData, 0644); err != nil {
		fmt.Printf("❌ Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✨ Registry generated at %s (%d images)\n", *outputFile, len(catalog.Images))
}
