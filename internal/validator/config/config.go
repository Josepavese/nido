package config

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"time"

	nidocfg "github.com/Josepavese/nido/internal/config"
)

// Config holds all configurable knobs for the validation suite.
type Config struct {
	RunID            string
	NidoBin          string
	BaseTemplate     string
	BaseImage        string
	PoolImage        string
	UserDataPath     string
	SSHPortBase      int
	GUIPortBase      int
	FWPortBase       int
	BootTimeout      time.Duration
	GUITimeout       time.Duration
	DownloadTimeout  time.Duration
	PortWaitTimeout  time.Duration
	SkipGUI          bool
	SkipUpdate       bool
	FailFast         bool
	KeepArtifacts    bool
	LogDir           string
	LogFile          string
	SummaryFile      string
	WorkingDir       string
	UpdateURL        string
	UpdateReleaseAPI string
	WorkflowPath     string
	CheckForward     bool
	CheckCloudInit   bool
	Scenario         string
	SSHUser          string
	SSHPassword      string
}

// Parse builds the configuration from flags and environment variables.
func Parse(runID string) Config {
	var cfg Config
	cfg.RunID = runID

	defaultBin := getenvOrDefault("NIDO_BIN", "nido")
	defaultTpl := os.Getenv("NIDO_TPL")
	defaultImg := os.Getenv("NIDO_IMAGE")
	defaultPool := os.Getenv("POOL_IMAGE")
	defaultUserData := os.Getenv("NIDO_USER_DATA")
	defaultSSHUser := os.Getenv("NIDO_SSH_USER")
	defaultSSHPwd := os.Getenv("NIDO_SSH_PASSWORD")

	defaultSSHBase := getenvInt("SSH_HOST_PORT_BASE", 2222)
	defaultGUIBase := getenvInt("GUI_PORT_BASE", 5900)
	defaultFWBase := getenvInt("FW_PORT_BASE", 30080)

	defaultBoot := getenvDuration("BOOT_TIMEOUT", 3*time.Minute)
	defaultGUI := getenvDuration("GUI_TIMEOUT", 30*time.Second)
	defaultDownload := getenvDuration("DOWNLOAD_TIMEOUT", 10*time.Minute)
	defaultPortWait := getenvDuration("PORT_WAIT_TIMEOUT", 30*time.Second)

	// Load Nido's main config for SSOT defaults
	home, _ := os.UserHomeDir()
	nidoConfigPath := filepath.Join(home, ".nido", "config.env")
	nidoCfg, _ := nidocfg.LoadConfig(nidoConfigPath)

	if defaultSSHUser == "" && nidoCfg != nil {
		defaultSSHUser = nidoCfg.SSHUser
	}
	// Fallback to hardcoded if still empty
	if defaultSSHUser == "" {
		defaultSSHUser = "vmuser"
	}

	wd, _ := os.Getwd()

	flag.StringVar(&cfg.NidoBin, "nido-bin", defaultBin, "Path to nido binary")
	flag.StringVar(&cfg.BaseTemplate, "template", defaultTpl, "Base template to use (NIDO_TPL)")
	flag.StringVar(&cfg.BaseImage, "image", defaultImg, "Base image to use for --image spawn (NIDO_IMAGE)")
	flag.StringVar(&cfg.PoolImage, "pool-image", defaultPool, "Image to pull/test from image pool (POOL_IMAGE)")
	flag.StringVar(&cfg.UserDataPath, "user-data", defaultUserData, "User-data file to inject (NIDO_USER_DATA)")
	flag.IntVar(&cfg.SSHPortBase, "ssh-port-base", defaultSSHBase, "Base host port for SSH (SSH_HOST_PORT_BASE)")
	flag.IntVar(&cfg.GUIPortBase, "gui-port-base", defaultGUIBase, "Base host port for GUI (GUI_PORT_BASE)")
	flag.IntVar(&cfg.FWPortBase, "fw-port-base", defaultFWBase, "Base host port for forwarded ports (FW_PORT_BASE)")
	flag.DurationVar(&cfg.BootTimeout, "boot-timeout", defaultBoot, "Max time to wait for VM boot/SSH (BOOT_TIMEOUT)")
	flag.DurationVar(&cfg.GUITimeout, "gui-timeout", defaultGUI, "Max time to wait for GUI/VNC port (GUI_TIMEOUT)")
	flag.DurationVar(&cfg.DownloadTimeout, "download-timeout", defaultDownload, "Max time to wait for image downloads (DOWNLOAD_TIMEOUT)")
	flag.DurationVar(&cfg.PortWaitTimeout, "port-wait-timeout", defaultPortWait, "Max time to wait for forwarded ports (PORT_WAIT_TIMEOUT)")
	flag.BoolVar(&cfg.SkipGUI, "skip-gui", getenvBool("SKIP_GUI", true), "Skip GUI-related checks (SKIP_GUI)")
	flag.BoolVar(&cfg.SkipUpdate, "skip-update", getenvBool("SKIP_UPDATE", true), "Skip update command checks (SKIP_UPDATE)")
	flag.BoolVar(&cfg.FailFast, "fail-fast", getenvBool("FAIL_FAST", true), "Stop on first critical failure (FAIL_FAST)")
	flag.BoolVar(&cfg.KeepArtifacts, "keep-artifacts", getenvBool("KEEP_ARTIFACTS", false), "Do not cleanup VMs/templates after run (KEEP_ARTIFACTS)")
	flag.BoolVar(&cfg.CheckForward, "check-forwarding", getenvBool("CHECK_FORWARDING", false), "Start dummy server and dial forwarded port to verify connectivity (CHECK_FORWARDING)")
	flag.BoolVar(&cfg.CheckCloudInit, "check-cloud-init", getenvBool("CHECK_CLOUD_INIT", false), "Verify cloud-init marker in guest via SSH (CHECK_CLOUD_INIT)")
	flag.StringVar(&cfg.SSHUser, "ssh-user", defaultSSHUser, "Default SSH user for guest (NIDO_SSH_USER)")
	flag.StringVar(&cfg.SSHPassword, "ssh-password", defaultSSHPwd, "Default SSH password for guest (NIDO_SSH_PASSWORD)")

	// Logging paths
	flag.StringVar(&cfg.LogDir, "log-dir", filepath.Join("logs"), "Directory to store validation logs")
	flag.StringVar(&cfg.WorkingDir, "workdir", wd, "Working directory for the suite")
	flag.StringVar(&cfg.WorkflowPath, "workflow", getenvOrDefault("NIDO_WORKFLOW", filepath.Join("internal", "validator", "workflows", "default.yaml")), "Path to workflow definition YAML (NIDO_WORKFLOW)")
	flag.StringVar(&cfg.Scenario, "scenario", "", "Specific scenario to run (case-insensitive)")
	flag.StringVar(&cfg.UpdateURL, "update-url", os.Getenv("NIDO_UPDATE_URL"), "Override update download URL (NIDO_UPDATE_URL)")
	flag.StringVar(&cfg.UpdateReleaseAPI, "update-release-api", os.Getenv("NIDO_RELEASE_API"), "Override release API endpoint (NIDO_RELEASE_API)")

	flag.Parse()

	timestamp := time.Now().UTC().Format("20060102-150405")
	cfg.LogFile = filepath.Join(cfg.LogDir, "cli-validate-"+timestamp+".ndjson")
	cfg.SummaryFile = filepath.Join(cfg.LogDir, "cli-validate-"+timestamp+".summary.txt")

	if cfg.PoolImage == "" {
		cfg.PoolImage = cfg.BaseImage
	}

	return cfg
}

func getenvOrDefault(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func getenvInt(key string, def int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return def
}

func getenvDuration(key string, def time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return def
}

func getenvBool(key string, def bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return def
}
