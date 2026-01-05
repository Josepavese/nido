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

	// 3. Load existing catalog for comparison (to detect new versions)
	existingCatalog, _ := image.LoadCatalogFromFile(*outputFile)

	// 4. Fetch and Filter Images
	catalog := &image.Catalog{
		SchemaVersion: "1",
		UpdatedAt:     time.Now().UTC(),
		Images:        []image.Image{},
	}

	newImages := []string{}
	for _, source := range config.Sources {
		fmt.Printf("Processing %s...\n", source.Name)

		imgEntry := image.Image{
			Name:        source.Name,
			Registry:    "official",
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

			// Detect new versions
			for _, v := range vers {
				isNew := true
				if existingCatalog != nil {
					_, _, findErr := existingCatalog.FindImage(source.Name, v.Version)
					if findErr == nil {
						isNew = false
					}
				}
				if isNew {
					fmt.Printf("  ✨ NEW VERSION FOUND: %s:%s\n", source.Name, v.Version)
					newImages = append(newImages, fmt.Sprintf("%s:%s", source.Name, v.Version))
				}
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

	// 5. Semantic Check & Write Output
	// If the new catalog (excluding UpdatedAt) is the same as the old one,
	// we keep the old UpdatedAt to avoid "noisy" git diffs.
	if existingCatalog != nil {
		// Temporary match check (ignoring UpdatedAt)
		tempCatalog := *catalog
		tempCatalog.UpdatedAt = existingCatalog.UpdatedAt

		newData, _ := json.Marshal(tempCatalog)
		oldData, _ := json.Marshal(existingCatalog)

		if string(newData) == string(oldData) {
			fmt.Println("\n✅ No semantic changes detected. Keeping existing catalog timestamp.")
			catalog.UpdatedAt = existingCatalog.UpdatedAt
		}
	}

	outputData, err := json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		fmt.Printf("❌ Failed to marshal JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputFile, outputData, 0644); err != nil {
		fmt.Printf("❌ Failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✨ Registry sync complete at %s (%d images)\n", *outputFile, len(catalog.Images))

	// 6. Signal for Validation (used by CI/CD)
	if len(newImages) > 0 {
		fmt.Println("\n📋 Targeted validation required for new images:")
		for _, imgName := range newImages {
			fmt.Printf("VALIDATE: %s\n", imgName)
		}
	}
}
