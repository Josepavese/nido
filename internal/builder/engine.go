package builder

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/image"
	"gopkg.in/yaml.v3"
)

// Engine orchestrates the image building process.
type Engine struct {
	CacheDir      string // Where ISOs and Drivers are cached
	WorkDir       string // Where temporary build files go
	ImageDir      string // Where the final output qcow2 goes
	Reporter      Reporter
	CommandOutput io.Writer
}

// NewEngine creates a new builder engine.
func NewEngine(cacheDir, workDir, imageDir string, opts ...EngineOption) *Engine {
	e := &Engine{
		CacheDir: cacheDir,
		WorkDir:  workDir,
		ImageDir: imageDir,
		Reporter: NopReporter{},
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

	// 2. Download Assets
	isoName := filepath.Base(bp.ISOURL)
	isoPath := filepath.Join(e.CacheDir, isoName)
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
	seedDir, err := safeJoin(e.WorkDir, bp.Name+"-seed")
	if err != nil {
		return fmt.Errorf("invalid blueprint name: %w", err)
	}
	os.RemoveAll(seedDir) // Clean start
	if err := os.MkdirAll(seedDir, 0755); err != nil {
		return err
	}

	for name, content := range bp.Scripts {
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

	seedISO, err := safeJoin(e.WorkDir, bp.Name+"-seed.iso")
	if err != nil {
		return fmt.Errorf("invalid seed iso path: %w", err)
	}
	// Use mkisofs/genisoimage to create the seed ISO
	// -J (Joliet) -R (Rock Ridge) -V (Label)
	cmd := exec.Command("mkisofs", "-J", "-R", "-V", "OEMDRV", "-o", seedISO, seedDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create seed ISO: %v (%s)", err, string(out))
	}

	// 4. Create Target Disk
	e.Reporter.Info("Creating disk %s (%s)...", bp.OutputImage, bp.OutputSize)
	// qemu-img create -f qcow2 path size
	cmd = exec.Command("qemu-img", "create", "-f", "qcow2", outputPath, bp.OutputSize)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create disk: %v (%s)", err, string(out))
	}

	// 5. Run QEMU Installer
	e.Reporter.Header("Starting Unattended Installation")
	e.Reporter.Info("This process may take a while (%s timeout).", bp.BuildSpecs.Timeout)

	qemuArgs := []string{
		"-m", bp.BuildSpecs.Memory,
		"-smp", fmt.Sprintf("%d", bp.BuildSpecs.CPU),
		"-drive", fmt.Sprintf("file=%s,if=virtio", outputPath),
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

	// KVM if available
	if _, err := os.Stat("/dev/kvm"); err == nil {
		qemuArgs = append(qemuArgs, "-enable-kvm", "-cpu", "host")
	} else {
		e.Reporter.Warn("KVM not available. Build will be slower.")
	}

	// Add loopback-only VNC for debugging visibility.
	qemuArgs = append(qemuArgs, "-vnc", "127.0.0.1:99") // Port 5999
	e.Reporter.Info("VNC available on 127.0.0.1:99 (port 5999).")

	e.Reporter.Info("Starting QEMU...")
	qemuCmd := exec.Command("qemu-system-x86_64", qemuArgs...)
	if e.CommandOutput != nil {
		qemuCmd.Stdout = e.CommandOutput
		qemuCmd.Stderr = e.CommandOutput
	}

	start := time.Now()
	if err := qemuCmd.Start(); err != nil {
		return fmt.Errorf("failed to start QEMU: %w", err)
	}

	// Wait with timeout
	done := make(chan error, 1)
	go func() {
		done <- qemuCmd.Wait()
	}()

	// Parse timeout
	timeout, _ := time.ParseDuration(bp.BuildSpecs.Timeout)
	if timeout == 0 {
		timeout = 2 * time.Hour
	}

	select {
	case <-time.After(timeout):
		if qemuCmd.Process != nil {
			qemuCmd.Process.Kill()
		}
		return fmt.Errorf("build timed out after %s", timeout)
	case err := <-done:
		if err != nil {
			return fmt.Errorf("QEMU exited with error: %w", err)
		}
	}

	e.Reporter.Success("Build completed in %s.", time.Since(start))

	// Validate Output
	if info, err := os.Stat(outputPath); err != nil || info.Size() < 100*1024*1024 { // < 100MB is suspicious
		return fmt.Errorf("build finished but output image seems invalid (too small or missing)")
	}

	// Clean up seed
	os.RemoveAll(seedDir)
	os.Remove(seedISO)

	return nil
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
	downloader := image.Downloader{Quiet: true}
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
