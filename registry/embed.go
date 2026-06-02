package registry

import "embed"

// FS contains the bundled registry assets shipped with the Nido binary.
//
//go:embed images.json sources.yaml blueprints/*.yaml blueprints/shared/*
var FS embed.FS
