package builder

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Josepavese/nido/internal/image"
)

// BlueprintInfo is the public, serializable summary used by CLI, MCP, and TUI.
type BlueprintInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Path        string `json:"path"`
	Source      string `json:"source"`
	OutputImage string `json:"output_image"`
	OutputTag   string `json:"output_tag"`
	OutputPath  string `json:"output_path"`
	OutputSize  string `json:"output_size"`
	Built       bool   `json:"built"`
	SSHUser     string `json:"ssh_user,omitempty"`
	HasPassword bool   `json:"has_initial_password"`
	CPU         int    `json:"cpu"`
	Memory      string `json:"memory"`
	Timeout     string `json:"timeout"`
}

type blueprintSearchDir struct {
	dir    string
	source string
}

// BlueprintSearchDirs returns the registry directories, in precedence order.
func BlueprintSearchDirs(cwd, nidoDir string) []string {
	dirs := blueprintSearchDirs(cwd, nidoDir)
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, d.dir)
	}
	return out
}

func blueprintSearchDirs(cwd, nidoDir string) []blueprintSearchDir {
	var dirs []blueprintSearchDir
	if cwd != "" {
		dirs = append(dirs, blueprintSearchDir{
			dir:    filepath.Join(cwd, "registry", "blueprints"),
			source: "project",
		})
	}
	if nidoDir != "" {
		dirs = append(dirs,
			blueprintSearchDir{dir: filepath.Join(nidoDir, "blueprints"), source: "user"},
			blueprintSearchDir{dir: filepath.Join(nidoDir, "registry", "blueprints"), source: "user-registry"},
		)
	}
	return dirs
}

// FindBlueprintPath resolves a blueprint name or path.
func FindBlueprintPath(cwd, nidoDir, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("blueprint name cannot be empty")
	}

	for _, candidate := range explicitBlueprintCandidates(cwd, name) {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	base := NormalizeBlueprintName(name)
	for _, dir := range blueprintSearchDirs(cwd, nidoDir) {
		for _, ext := range []string{".yaml", ".yml"} {
			candidate := filepath.Join(dir.dir, base+ext)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	return "", fmt.Errorf("blueprint %q not found", name)
}

// LoadBlueprintRef resolves and loads a blueprint, returning its summary.
func LoadBlueprintRef(cwd, nidoDir, imageDir, name string) (*image.Blueprint, BlueprintInfo, error) {
	path, err := FindBlueprintPath(cwd, nidoDir, name)
	if err != nil {
		return nil, BlueprintInfo{}, err
	}
	bp, err := LoadBlueprint(path)
	if err != nil {
		return nil, BlueprintInfo{}, err
	}
	return bp, NewBlueprintInfo(path, sourceForBlueprintPath(cwd, nidoDir, path), imageDir, bp), nil
}

// ListBlueprints scans all configured blueprint registries.
func ListBlueprints(cwd, nidoDir, imageDir string) ([]BlueprintInfo, error) {
	seen := map[string]bool{}
	var out []BlueprintInfo

	for _, dir := range blueprintSearchDirs(cwd, nidoDir) {
		entries, err := os.ReadDir(dir.dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read blueprint directory %s: %w", dir.dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !isBlueprintFile(entry.Name()) {
				continue
			}
			base := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
			if seen[base] {
				continue
			}
			path := filepath.Join(dir.dir, entry.Name())
			bp, err := LoadBlueprint(path)
			if err != nil {
				continue
			}
			if bp.Hidden {
				seen[base] = true
				continue
			}
			key := bp.Name
			if key == "" {
				key = base
			}
			if seen[key] {
				continue
			}
			seen[base] = true
			seen[key] = true
			out = append(out, NewBlueprintInfo(path, dir.source, imageDir, bp))
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return sourceRank(out[i].Source) < sourceRank(out[j].Source)
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// NewBlueprintInfo summarizes a loaded blueprint.
func NewBlueprintInfo(path, source, imageDir string, bp *image.Blueprint) BlueprintInfo {
	tag := BlueprintOutputTag(bp)
	outputPath := ""
	built := false
	if imageDir != "" && bp.OutputImage != "" {
		outputPath = filepath.Join(imageDir, bp.OutputImage)
		_, err := os.Stat(outputPath)
		built = err == nil
	}

	return BlueprintInfo{
		Name:        bp.Name,
		DisplayName: bp.DisplayName,
		Description: bp.Description,
		Version:     bp.Version,
		Path:        path,
		Source:      source,
		OutputImage: bp.OutputImage,
		OutputTag:   tag,
		OutputPath:  outputPath,
		OutputSize:  bp.OutputSize,
		Built:       built,
		SSHUser:     bp.SSHUser,
		HasPassword: bp.SSHPassword != "",
		CPU:         bp.BuildSpecs.CPU,
		Memory:      bp.BuildSpecs.Memory,
		Timeout:     bp.BuildSpecs.Timeout,
	}
}

// NormalizeBlueprintName strips a blueprint file extension if present.
func NormalizeBlueprintName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.TrimSuffix(name, ".yaml")
	name = strings.TrimSuffix(name, ".yml")
	return name
}

// BlueprintOutputTag is the tag accepted by nido spawn --image once built.
func BlueprintOutputTag(bp *image.Blueprint) string {
	return strings.TrimSuffix(bp.OutputImage, ".qcow2")
}

func explicitBlueprintCandidates(cwd, name string) []string {
	var names []string
	if isExplicitBlueprintPath(name) {
		names = append(names, name)
	} else if cwd != "" {
		names = append(names, filepath.Join(cwd, name))
	}
	var out []string
	for _, candidate := range names {
		out = append(out, candidate)
		if !isBlueprintFile(candidate) {
			out = append(out, candidate+".yaml", candidate+".yml")
		}
	}
	return out
}

func isExplicitBlueprintPath(name string) bool {
	return filepath.IsAbs(name) || strings.ContainsAny(name, `/\`)
}

func isBlueprintFile(name string) bool {
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

func sourceForBlueprintPath(cwd, nidoDir, path string) string {
	if cwd != "" {
		projectDir := filepath.Join(cwd, "registry", "blueprints")
		if pathWithin(projectDir, path) {
			return "project"
		}
	}
	if nidoDir != "" {
		userDir := filepath.Join(nidoDir, "blueprints")
		if pathWithin(userDir, path) {
			return "user"
		}
		userRegistryDir := filepath.Join(nidoDir, "registry", "blueprints")
		if pathWithin(userRegistryDir, path) {
			return "user-registry"
		}
	}
	return "path"
}

func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func sourceRank(source string) int {
	switch source {
	case "project":
		return 0
	case "path":
		return 1
	case "user":
		return 2
	case "user-registry":
		return 3
	default:
		return 4
	}
}
