package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/registry/builder"
	"gopkg.in/yaml.v3"
)

func main() {
	sources := flag.String("sources", "registry/sources.yaml", "Path to sources configuration")
	output := flag.String("output", "registry/images.json", "Path to output JSON")
	flag.Parse()
	if err := run(*sources, *output, builder.NewFetcher(nil), time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(sources, output string, fetcher builder.Fetcher, now time.Time) error {
	data, err := os.ReadFile(sources)
	if err != nil {
		return err
	}
	var config builder.SourcesConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return err
	}
	var previous *image.Catalog
	if _, err := os.Stat(output); err == nil {
		previous, err = image.LoadCatalogFromFile(output)
		if err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	catalog, changed, warnings, err := builder.Build(config, previous, fetcher, now)
	if err != nil {
		return fmt.Errorf("catalog unchanged: %w", err)
	}
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, warning)
	}
	data, err = json.MarshalIndent(catalog, "", "  ")
	if err != nil {
		return err
	}
	// Write in the destination directory, then replace only after complete generation.
	temp, err := os.CreateTemp(filepath.Dir(output), ".catalog-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err = temp.Write(append(data, '\n')); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Chmod(0644); err != nil {
		temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err = os.Rename(temp.Name(), output); err != nil {
		return err
	}
	fmt.Printf("Registry sync complete: %s (%d sources)\n", output, len(catalog.Images))
	for _, tag := range changed {
		fmt.Printf("VALIDATE: %s\n", tag)
	}
	return nil
}
