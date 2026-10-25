package image

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Josepavese/nido/v5/internal/machine"
	"github.com/Josepavese/nido/v5/internal/pkg/sysutil"
)

const artifactSchema = 1
const artifactSuffix = ".nido.json"
const artifactMode = 0600

type artifactMetadata struct {
	Schema int    `json:"schema"`
	Arch   string `json:"arch"`
}

func MetadataPath(disk string) string  { return disk + artifactSuffix }
func VariablesPath(disk string) string { return disk + ".uefi.fd" }

// ReadArchitecture reads the declared ISA; disk formats do not identify it.
func ReadArchitecture(disk string) (string, error) {
	data, err := os.ReadFile(MetadataPath(disk))
	if err != nil {
		return "", err
	}
	var meta artifactMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", err
	}
	if meta.Schema != artifactSchema {
		return "", fmt.Errorf("unsupported guest artifact schema %d", meta.Schema)
	}
	return machine.Normalize(meta.Arch)
}

func WriteArchitecture(disk, arch string) error {
	canonical, err := machine.Normalize(arch)
	if err != nil {
		return err
	}
	data, err := json.Marshal(artifactMetadata{Schema: artifactSchema, Arch: canonical})
	if err != nil {
		return err
	}
	return sysutil.WriteFile(MetadataPath(disk), data, artifactMode)
}
