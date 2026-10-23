package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"os"
	urlpath "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/image"
	"github.com/Josepavese/nido/v4/internal/pkg/seediso"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"gopkg.in/yaml.v3"
)

const defaultBuildTimeout = 2 * time.Hour

// Engine orchestrates the image building process.
type Engine struct {
	CacheDir      string // Where ISOs and Drivers are cached
	WorkDir       string // Where temporary build files go
	ImageDir      string // Where the final output qcow2 goes
	Reporter      Reporter
	CommandOutput io.Writer
	Acceleration  hardware.Virtualization
}

// NewEngine creates a new builder engine.
func NewEngine(cacheDir, workDir, imageDir string, acceleration hardware.Virtualization, opts ...EngineOption) *Engine {
	e := &Engine{
		CacheDir:     cacheDir,
		WorkDir:      workDir,
		ImageDir:     imageDir,
		Reporter:     NopReporter{},
		Acceleration: acceleration,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(e)
		}
	}
	return e
}

// LoadBlueprint reads a blueprint from a YAML file.
func LoadBlueprint(path string) (*image.Blueprint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read blueprint: %w", err)
	}

	var bp image.Blueprint
	if err := yaml.Unmarshal(data, &bp); err != nil {
		return nil, fmt.Errorf("failed to parse blueprint definition: %w", err)
	}

	// Resolve external scripts
	baseDir := filepath.Dir(path)
	for name, content := range bp.Scripts {
		if len(content) > 0 && content[0] == '@' {
			relPath := content[1:]
			scriptPath, err := safeJoin(baseDir, relPath)
			if err != nil {
				return nil, fmt.Errorf("invalid external script path '%s': %w", relPath, err)
			}
			scriptData, err := os.ReadFile(scriptPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load script '%s' from '%s': %w", name, scriptPath, err)
			}
			bp.Scripts[name] = string(scriptData)
		}
	}
	for name, content := range bp.Scripts {
		content = applyBlueprintVariables(content, bp.Variables)
		if err := validateScript(name, content); err != nil {
			return nil, err
		}
		bp.Scripts[name] = content
	}

	return &bp, nil
}

// Build executes the blueprint instructions.
func (e *Engine) Build(bp *image.Blueprint) error {
	// 1. Prepare Environments
	if err := os.MkdirAll(e.CacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache dir: %w", err)
	}
	if err := os.MkdirAll(e.WorkDir, 0755); err != nil {
		return fmt.Errorf("failed to create work dir: %w", err)
	}
	if err := os.MkdirAll(e.ImageDir, 0755); err != nil {
		return fmt.Errorf("failed to create image dir: %w", err)
	}

	outputPath, err := safeJoin(e.ImageDir, bp.OutputImage)
	if err != nil {
		return fmt.Errorf("invalid output image path: %w", err)
	}
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("output image already exists: %s", outputPath)
	}
	stage, err := os.CreateTemp(e.ImageDir, ".nido-output-*.building")
	if err != nil {
		return err
	}
	tmpOutputPath := stage.Name()
	if err := stage.Close(); err != nil {
		os.Remove(tmpOutputPath)
		return err
	}
	defer os.Remove(tmpOutputPath)

	// 2. Download Assets
	isoName, err := cacheAssetName(bp.ISOName, bp.ISOURL, "installer.iso")
	if err != nil {
		return fmt.Errorf("invalid iso asset name: %w", err)
	}
	isoPath, err := safeJoin(e.CacheDir, isoName)
	if err != nil {
		return fmt.Errorf("invalid iso asset path: %w", err)
	}
	if err := e.ensureAsset(bp.ISOURL, isoPath, bp.ISOChecksum); err != nil {
		return err
	}

	driverPaths := []string{}
	for _, drv := range bp.Drivers {
		drvPath, err := safeJoin(e.CacheDir, drv.Name)
		if err != nil {
			return fmt.Errorf("invalid driver path %q: %w", drv.Name, err)
		}
		if err := e.ensureAsset(drv.URL, drvPath, drv.Checksum); err != nil {
			return err
		}
		driverPaths = append(driverPaths, drvPath)
	}

	// 3. Create Seed Media (Autounattend)
	work, err := os.MkdirTemp(e.WorkDir, "nido-build-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	seedDir := filepath.Join(work, "seed")
	if err := os.Mkdir(seedDir, 0755); err != nil {
		return err
	}

	seedScripts, err := canonicalSeedScripts(bp.Scripts)
	if err != nil {
		return err
	}
	for name, content := range seedScripts {
		scriptPath, err := safeJoin(seedDir, name)
		if err != nil {
			return fmt.Errorf("invalid script path %q: %w", name, err)
		}
		if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
			return fmt.Errorf("failed to create script dir %s: %w", filepath.Dir(scriptPath), err)
		}
		if err := os.WriteFile(scriptPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write script %s: %w", name, err)
		}
	}

	seedISO := filepath.Join(work, "seed.iso")
	if err := createSeedISO(seedISO, seedDir, "OEMDRV"); err != nil {
		return err
	}

	// 4. Create Target Disk
	e.Reporter.Info("Creating disk %s (%s)...", bp.OutputImage, bp.OutputSize)
	// qemu-img create -f qcow2 path size
	qemuImg, err := sysutil.QemuImgBinary()
	if err != nil {
		return fmt.Errorf("failed to find qemu-img: %w", err)
	}
	cmd := sysutil.NewCommand(qemuImg, "create", "-f", "qcow2", tmpOutputPath, bp.OutputSize)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create disk: %v (%s)", err, string(out))
	}

	// 5. Run QEMU Installer
	e.Reporter.Header("Starting Unattended Installation")
	e.Reporter.Info("This process may take a while (%s timeout).", bp.BuildSpecs.Timeout)

	qemuArgs := []string{
		"-m", bp.BuildSpecs.Memory,
		"-smp", fmt.Sprintf("%d", bp.BuildSpecs.CPU),
		"-drive", fmt.Sprintf("file=%s,format=qcow2,if=virtio", tmpOutputPath),
		"-cdrom", isoPath,
		"-netdev", "user,id=net0",
		"-device", "virtio-net-pci,netdev=net0",
		"-vga", "std", // Standard VGA is usually safer for installers
		"-no-reboot",       // Exit when install finishes (if supported by guest/script)
		"-display", "none", // Headless by default? or gtk?
		// Helper drive for Seed
		"-drive", fmt.Sprintf("file=%s,media=cdrom", seedISO),
	}

	// Attach drivers
	for _, drv := range driverPaths {
		qemuArgs = append(qemuArgs, "-drive", fmt.Sprintf("file=%s,media=cdrom", drv))
	}

	qemuArgs = append(qemuArgs, e.Acceleration.QEMUArgs()...)
	qemuArgs = append(qemuArgs, "-cpu", e.Acceleration.CPU)
	if e.Acceleration.Accelerator == "" {
		e.Reporter.Warn("Hardware acceleration not available. Build will be slower.")
	}

	// Add loopback-only VNC for debugging visibility.
	qemuArgs = append(qemuArgs, "-vnc", "127.0.0.1:99") // Port 5999
	e.Reporter.Info("VNC available on 127.0.0.1:99 (port 5999).")

	e.Reporter.Info("Starting QEMU...")
	qemuSystem, err := sysutil.QemuSystemBinary()
	if err != nil {
		return fmt.Errorf("failed to find QEMU: %w", err)
	}
	timeout, _ := time.ParseDuration(bp.BuildSpecs.Timeout)
	if timeout <= 0 {
		timeout = defaultBuildTimeout
	}
	elapsed, err := sysutil.RunTimedCommand(qemuSystem, qemuArgs, timeout, e.CommandOutput)
	if err != nil {
		return fmt.Errorf("QEMU installer failed: %w", err)
	}
	e.Reporter.Success("Build completed in %s.", elapsed)

	// Validate Output
	if info, err := os.Stat(tmpOutputPath); err != nil || info.Size() < 100*1024*1024 { // < 100MB is suspicious
		return fmt.Errorf("build finished but output image seems invalid (too small or missing)")
	}
	if err := sysutil.PublishFile(tmpOutputPath, outputPath); err != nil {
		return fmt.Errorf("failed to finalize output image: %w", err)
	}

	return nil
}

func applyBlueprintVariables(content string, vars map[string]string) string {
	for key, value := range vars {
		content = strings.ReplaceAll(content, "{{"+key+"}}", value)
	}
	return content
}

func validateScript(name, content string) error {
	if !strings.EqualFold(filepath.Ext(name), ".xml") {
		return nil
	}
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		if _, err := decoder.Token(); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("invalid XML script %s: %w", name, err)
		}
	}
}

func cacheAssetName(preferredName, rawURL, fallbackName string) (string, error) {
	name := strings.TrimSpace(preferredName)
	if name == "" {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
		}
		name = urlpath.Base(parsed.EscapedPath())
		if name == "." || name == "/" || name == "" {
			name = fallbackName
		}
		name = sanitizeFilename(name)
		if name == "" {
			name = fallbackName
		}
		if filepath.Ext(name) == "" {
			sum := sha256.Sum256([]byte(rawURL))
			name = fmt.Sprintf("%s-%s%s", name, hex.EncodeToString(sum[:])[:8], filepath.Ext(fallbackName))
		}
	} else {
		name = sanitizeFilename(name)
	}
	clean := filepath.Clean(name)
	if clean == "." || filepath.IsAbs(clean) || clean != filepath.Base(clean) {
		return "", fmt.Errorf("asset name must be a filename")
	}
	return clean, nil
}

func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 32:
			b.WriteByte('_')
		case strings.ContainsRune(`<>:"/\|?*`, r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	name = strings.Trim(b.String(), ". ")
	if name == "" {
		return ""
	}
	return name
}

func createSeedISO(outputPath, sourceDir, label string) error {
	return seediso.Create(outputPath, sourceDir, label)
}

func canonicalSeedScripts(scripts map[string]string) (map[string]string, error) {
	canonical := make(map[string]string, len(scripts))
	for name, content := range scripts {
		dest := canonicalSeedScriptName(name)
		if existing, ok := canonical[dest]; ok && existing != content {
			return nil, fmt.Errorf("multiple seed scripts map to %q", dest)
		}
		canonical[dest] = content
	}
	return canonical, nil
}

func canonicalSeedScriptName(name string) string {
	clean := filepath.ToSlash(filepath.Clean(name))
	if strings.EqualFold(clean, "autounattend.xml") {
		return "Autounattend.xml"
	}
	return name
}

func BlueprintSpawnSeedFiles(bp *image.Blueprint) map[string]string {
	files := make(map[string]string, len(bp.Scripts))
	for name, content := range bp.Scripts {
		if strings.EqualFold(filepath.ToSlash(filepath.Clean(name)), "autounattend.xml") {
			continue
		}
		files[name] = content
	}
	if len(files) == 0 {
		return nil
	}
	return files
}

func (e *Engine) ensureAsset(url, dest, checksum string) error {
	if err := validateDownloadURL(url); err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		if err := verifyOptionalChecksum(dest, checksum); err != nil {
			return fmt.Errorf("cached asset checksum failed for %s: %w", filepath.Base(dest), err)
		}
		e.Reporter.Info("Using cached asset: %s", filepath.Base(dest))
		return nil
	}

	e.Reporter.Info("Downloading asset: %s", filepath.Base(dest))
	downloader := image.NewTransfer(image.TransferOptions{OnProgress: nil})
	if err := downloader.Download(url, dest, 0); err != nil {
		return err
	}
	if err := verifyOptionalChecksum(dest, checksum); err != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("downloaded asset checksum failed for %s: %w", filepath.Base(dest), err)
	}
	return nil
}

func safeJoin(root, rel string) (string, error) {
	if strings.TrimSpace(rel) == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute paths are not allowed")
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes target directory")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	target := filepath.Join(rootAbs, clean)
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes target directory")
	}
	return targetAbs, nil
}

func validateDownloadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("unsupported URL scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("URL host cannot be empty")
	}
	if u.Scheme == "http" {
		host := strings.ToLower(u.Hostname())
		if host != "localhost" && host != "127.0.0.1" && host != "::1" {
			return fmt.Errorf("plain HTTP is allowed only for loopback hosts")
		}
	}
	return nil
}

func verifyOptionalChecksum(path, checksum string) error {
	checksum = strings.TrimSpace(checksum)
	if checksum == "" {
		return nil
	}
	switch len(checksum) {
	case 64:
		return image.VerifyChecksum(path, checksum, "sha256")
	case 128:
		return image.VerifyChecksum(path, checksum, "sha512")
	default:
		return fmt.Errorf("unsupported checksum length %d", len(checksum))
	}
}
