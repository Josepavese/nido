package builder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/ui"
	"gopkg.in/yaml.v3"
)

// Engine orchestrates the image building process.
type Engine struct {
	CacheDir string // Where ISOs and Drivers are cached
	WorkDir  string // Where temporary build files go
	ImageDir string // Where the final output qcow2 goes
}

// NewEngine creates a new builder engine.
func NewEngine(cacheDir, workDir, imageDir string) *Engine {
	return &Engine{
		CacheDir: cacheDir,
		WorkDir:  workDir,
		ImageDir: imageDir,
	}
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
			scriptPath := filepath.Join(baseDir, relPath)
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

	outputPath := filepath.Join(e.ImageDir, bp.OutputImage)
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
		drvPath := filepath.Join(e.CacheDir, drv.Name)
		if err := e.ensureAsset(drv.URL, drvPath, drv.Checksum); err != nil {
			return err
		}
		driverPaths = append(driverPaths, drvPath)
	}

	// 3. Create Seed Media (Autounattend)
	seedDir := filepath.Join(e.WorkDir, bp.Name+"-seed")
	os.RemoveAll(seedDir) // Clean start
	if err := os.MkdirAll(seedDir, 0755); err != nil {
		return err
	}

	for name, content := range bp.Scripts {
		if err := os.WriteFile(filepath.Join(seedDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write script %s: %w", name, err)
		}
	}

	seedISO := filepath.Join(e.WorkDir, bp.Name+"-seed.iso")
	// Use mkisofs/genisoimage to create the seed ISO
	// -J (Joliet) -R (Rock Ridge) -V (Label)
	cmd := exec.Command("mkisofs", "-J", "-R", "-V", "OEMDRV", "-o", seedISO, seedDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create seed ISO: %v (%s)", err, string(out))
	}

	// 4. Create Target Disk
	ui.Info("Creating empty disk %s (%s)...", bp.OutputImage, bp.OutputSize)
	// qemu-img create -f qcow2 path size
	cmd = exec.Command("qemu-img", "create", "-f", "qcow2", outputPath, bp.OutputSize)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create disk: %v (%s)", err, string(out))
	}

	// 5. Run QEMU Installer
	ui.Header("Starting Unattended Installation")
	ui.Info("This process will take a while (%s timeout). Please do not close this window.", bp.BuildSpecs.Timeout)

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
		ui.Warn("KVM not available! Building will be extremely slow.")
	}

	// Add VNC for debugging visibility
	qemuArgs = append(qemuArgs, "-vnc", ":99") // Port 5999
	ui.Info("VNC server running on :99 (Port 5999) for observation.")

	ui.Info("Running QEMU...")
	qemuCmd := exec.Command("qemu-system-x86_64", qemuArgs...)

	// We want to capture output or show it?
	// QEMU usually prints to stderr.
	qemuCmd.Stdout = os.Stdout
	qemuCmd.Stderr = os.Stderr

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

	ui.Success("Build complete in %s! 🐣", time.Since(start))

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
	if _, err := os.Stat(dest); err == nil {
		// Exists, assume okay for now (TODO: Verify checksum if provided)
		ui.Info("Using cached asset: %s", filepath.Base(dest))
		return nil
	}

	ui.Info("Downloading asset: %s", filepath.Base(dest))
	downloader := image.Downloader{Quiet: false}
	if err := downloader.Download(url, dest, 0); err != nil {
		return err
	}
	return nil
}
