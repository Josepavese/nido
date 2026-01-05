package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/mcp"
	"github.com/Josepavese/nido/internal/provider"
	"github.com/Josepavese/nido/internal/ui"
)

// Version is injected at build time
var Version = "v3.0.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// Home Dir for Nido State
	home, _ := os.UserHomeDir()
	nidoDir := filepath.Join(home, ".nido")

	cwd, _ := os.Getwd()

	// Load Config
	cfgPath := filepath.Join(nidoDir, "config.env")
	// For dev, check local project config first if nidoDir doesn't have it
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = filepath.Join(cwd, "config", "config.env")
	}

	cfg, _ := config.LoadConfig(cfgPath)
	prov := provider.NewQemuProvider(nidoDir, cfg)

	var err error

	switch cmd {
	case "version":
		cmdVersion()
	case "mcp-json-list":
		cmdMcpJsonList(prov)
	case "ls", "list":
		vms, _ := prov.List()
		if len(vms) == 0 {
			ui.Info("The nest is quiet. No VMs are currently making noise.")
		} else {
			fmt.Printf("\n %s%-20s %-12s %-10s %s%s\n", ui.Bold, "NAME", "STATE", "PID", "PORT", ui.Reset)
			fmt.Printf(" %s%s%s\n", ui.Dim, strings.Repeat("-", 60), ui.Reset)
			for _, vm := range vms {
				stateColor := ui.Yellow
				if vm.State == "running" {
					stateColor = ui.Green
				}
				fmt.Printf(" %-20s %s%-12s%s %-10d %d\n", vm.Name, stateColor, vm.State, ui.Reset, vm.PID, vm.SSHPort)
			}
			fmt.Println("")
		}
	case "ssh":
		if len(args) < 1 {
			ui.Error("Usage: nido ssh <name>")
			os.Exit(1)
		}
		cmdSsh(prov, args[0], args[1:])
	case "doctor":
		cmdDoctor(prov)
	case "info":
		if len(args) < 1 {
			fmt.Println("Usage: nido info <name>")
			os.Exit(1)
		}
		info, err := prov.Info(args[0])
		if err != nil {
			ui.Error("Failed to probe %s: %v", args[0], err)
			os.Exit(1)
		}
		ui.Header("VM Profile: " + info.Name)
		ui.FancyLabel("State", info.State)
		ui.FancyLabel("IP Address", info.IP)
		ui.FancyLabel("SSH Command", fmt.Sprintf("ssh -p %d %s@%s", info.SSHPort, info.SSHUser, info.IP))
		fmt.Println("")
	case "spawn":
		if len(args) < 1 {
			fmt.Println("Usage: nido spawn <name> [--image <name:tag> | <template>]")
			os.Exit(1)
		}
		name := args[0]
		tpl := ""
		imageTag := ""

		if len(args) > 1 {
			if args[1] == "--image" && len(args) > 2 {
				imageTag = args[2]
			} else {
				tpl = args[1]
			}
		}

		if imageTag != "" {
			// Resolve image
			imgDir := filepath.Join(nidoDir, "images")
			catalog, err := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
			if err != nil {
				ui.Error("Failed to load catalog: %v", err)
				os.Exit(1)
			}

			pName, pVer := imageTag, ""
			if strings.Contains(imageTag, ":") {
				parts := strings.Split(imageTag, ":")
				pName, pVer = parts[0], parts[1]
			}

			img, ver, err := catalog.FindImage(pName, pVer)
			if err != nil {
				ui.Error("Image %s not found in catalog.", imageTag)
				os.Exit(1)
			}

			imgPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))

			// Auto-pull if missing
			if _, err := os.Stat(imgPath); os.IsNotExist(err) {
				ui.Info("Image not found locally. Pulling %s:%s...", img.Name, ver.Version)
				downloader := image.Downloader{}
				if err := downloader.Download(ver.URL, imgPath, ver.SizeBytes); err != nil {
					ui.Error("Download failed: %v", err)
					os.Exit(1)
				}
				// Verify
				ui.Ironic("Verifying genetic integrity...")
				if err := image.VerifyChecksum(imgPath, ver.Checksum, ver.ChecksumType); err != nil {
					ui.Error("Verification failed: %v", err)
					os.Remove(imgPath)
					os.Exit(1)
				}
				ui.Success("Image downloaded successfully.")
			}

			// Use absolute path as template
			tpl = imgPath
		}

		ui.Ironic("Initiating hypervisor handshake...")
		if err := prov.Spawn(name, provider.VMOptions{DiskPath: tpl}); err != nil {
			ui.Error("Hatch failure for %s: %v", name, err)
			os.Exit(1)
		}

		source := "template"
		if imageTag != "" {
			source = "image " + imageTag
		}
		ui.Success("VM %s hatched successfully from %s.", name, source)
	case "start":
		if len(args) < 1 {
			fmt.Println("Usage: nido start <name>")
			os.Exit(1)
		}
		ui.Ironic("Reviving digital consciousness...")
		if err := prov.Start(args[0]); err != nil {
			ui.Error("Neural jumpstart failure for %s: %v", args[0], err)
			os.Exit(1)
		}
		ui.Success("VM %s is now active.", args[0])
	case "stop":
		if len(args) < 1 {
			fmt.Println("Usage: nido stop <name>")
			os.Exit(1)
		}
		ui.Ironic("Sending stop signal to nerves...")
		if err := prov.Stop(args[0], true); err != nil {
			ui.Error("Deep sleep rejection for %s: %v", args[0], err)
			os.Exit(1)
		}
		ui.Success("VM %s is now resting.", args[0])
	case "template":
		if len(args) == 0 || args[0] == "list" {
			ui.Header("Nido Templates (Grave)")
			templates, err := prov.ListTemplates()
			if err != nil {
				ui.Error("Failed to access graveyard: %v", err)
				os.Exit(1)
			}
			if len(templates) == 0 {
				ui.Info("The graveyard is empty. No templates found.")
			} else {
				for _, name := range templates {
					fmt.Printf("  %s%-20s%s %s(Cold storage)%s\n", ui.Cyan, name, ui.Reset, ui.Dim, ui.Reset)
				}
			}
			fmt.Println("")
			return
		}
		subCmd := args[0]
		if subCmd == "create" {
			if len(args) < 3 {
				ui.Error("Missing parameters. Usage: nido template create <vm-name> <template-name>")
				os.Exit(1)
			}
			vmName := args[1]
			tplName := args[2]
			ui.Ironic("Encrypting genetic data into template...")
			path, err := prov.CreateTemplate(vmName, tplName)
			if err != nil {
				ui.Error("Cloning evolution failed: %v", err)
				os.Exit(1)
			}
			ui.Success("New species '"+tplName+"' archived at: %s", path)
		} else {
			ui.Error("Unknown template action: %s", subCmd)
		}
	case "config":
		cmdConfig(cfg, cfgPath)
	case "register":
		cmdRegister()
	case "delete", "destroy":
		if len(args) < 1 {
			fmt.Println("Usage: nido delete <name>")
			os.Exit(1)
		}
		ui.Ironic("Vaporizing digital footprint...")
		if err := prov.Delete(args[0]); err != nil {
			ui.Error("Eviction failed: %v", err)
			os.Exit(1)
		}
		ui.Success("VM %s has left the nest (permanently).", args[0])
	case "prune":
		ui.Ironic("Cleaning up the graveyard...")
		count, err := prov.Prune()
		if err != nil {
			ui.Error("Pruning failed: %v", err)
			os.Exit(1)
		}
		ui.Success("Vaporized %d stopped life forms.", count)
	case "image":
		cmdImage(nidoDir, args)
	case "mcp":
		server := mcp.NewServer(prov)
		server.Serve()
	case "completion":
		if len(args) < 1 {
			fmt.Println("Usage: nido completion <bash|zsh>")
			os.Exit(1)
		}
		sub := args[0]
		if sub == "list-vms" {
			vms, _ := prov.List()
			for _, vm := range vms {
				fmt.Printf("%s ", vm.Name)
			}
			return
		}
		if sub == "list-templates" {
			templates, _ := prov.ListTemplates()
			for _, name := range templates {
				fmt.Printf("%s ", name)
			}
			return
		}
		cmdCompletion(sub)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		ui.Error("Unexpected fault: %v", err)
		os.Exit(1)
	}
}

func cmdCompletion(bashOrZsh string) {
	bash := `_nido_completions() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="ls spawn start ssh info stop template config register version delete doctor prune completion"

    case "${prev}" in
        spawn)
            COMPREPLY=( $(compgen -W "$(nido completion list-templates)" -- ${cur}) )
            return 0
            ;;
        ssh|info|start|stop|delete)
            COMPREPLY=( $(compgen -W "$(nido completion list-vms)" -- ${cur}) )
            return 0
            ;;
        template)
            COMPREPLY=( $(compgen -W "create list" -- ${cur}) )
            return 0
            ;;
        create)
            if [[ ${COMP_WORDS[COMP_CWORD-2]} == "template" ]]; then
                COMPREPLY=( $(compgen -W "$(nido completion list-vms)" -- ${cur}) )
                return 0
            fi
            ;;
    esac

    if [[ ${COMP_CWORD} -eq 1 ]] ; then
        COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
        return 0
    fi
}
complete -F _nido_completions nido
`
	zsh := `_nido_completion() {
  local -a commands
  commands=(
    'ssh:Connect to a VM via SSH bridge'
    'ls:List all life forms in the nest'
    'start:Revive VM from deep sleep'
    'stop:Put VM into deep sleep'
    'delete:Evict VM from the nest'
    'prune:Vaporize all stopped life forms'
    'config:Dump genetic configuration'
    'register:Prepare MCP handshake'
    'version:Check evolutionary state'
    'doctor:Run a system health check'
    'completion:Generate shell completions'
  )

  _arguments -C \
    '1: :->command' \
    '*:: :->args'

  case $state in
    command)
      _describe -t commands 'nido command' commands
      ;;
    args)
      case $words[1] in
        spawn)
          _values 'templates' $(nido completion list-templates)
          ;;
        ssh|info|start|stop|delete)
          _values 'vms' $(nido completion list-vms)
          ;;
        template)
          if (( CURRENT == 2 )); then
            _values 'actions' 'create' 'list'
          elif (( CURRENT == 3 )) && [[ $words[2] == "create" ]]; then
            _values 'vms' $(nido completion list-vms)
          fi
          ;;
      esac
      ;;
  esac
}
compdef _nido_completion nido
`
	if bashOrZsh == "bash" {
		fmt.Print(bash)
	} else if bashOrZsh == "zsh" {
		fmt.Print(zsh)
	} else {
		ui.Error("Unsupported shell: %s. Only bash and zsh are supported.", bashOrZsh)
		os.Exit(1)
	}
}

func cmdSsh(prov provider.VMProvider, name string, args []string) {
	cmdStr, err := prov.SSHCommand(name)
	if err != nil {
		ui.Error("Target acquisition failed: %v", err)
		os.Exit(1)
	}
	ui.Info("Bridging to %s...", name)
	if len(args) == 0 {
		ui.Ironic("Establishing secure neural link...")
	}

	parts := strings.Split(cmdStr, " ")
	// ssh -p [port] [user]@[ip] [args...]
	// Inject options to skip fingerprint check for ephemeral VMs
	sshOptions := []string{
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "LogLevel=ERROR", // Reduce noise
		"-o", "BatchMode=yes", // Fail instead of prompting for password
	}

	baseArgs := append(sshOptions, parts[1:]...)
	finalArgs := append(baseArgs, args...)

	cmd := exec.Command(parts[0], finalArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Pass through exit code if possible
		if exitError, ok := err.(*exec.ExitError); ok {
			os.Exit(exitError.ExitCode())
		}
		os.Exit(1)
	}
}

func cmdDoctor(prov provider.VMProvider) {
	ui.Header("Nido System Diagnostics")
	reports := prov.Doctor()
	for _, r := range reports {
		// Parse report for UI coloring
		icon := ui.IconSuccess
		if strings.Contains(r, "[FAIL]") {
			icon = ui.IconError
		}
		fmt.Printf("  %s %s\n", icon, r)
	}
	fmt.Println("")
	ui.Success("Audit complete. The nest is structurally sound.")
}

func printUsage() {
	ui.Header("Nido: The Universal Nest")
	fmt.Printf("Usage: %snido %s<command>%s [args...]\n\n", ui.Bold, ui.Cyan, ui.Reset)

	fmt.Printf("%sVM MANAGEMENT%s\n", ui.Bold, ui.Reset)
	fmt.Printf("  %-10s %sCreate and hatch a new VM from a template%s\n", "spawn", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sRevive a VM from deep sleep%s\n", "start", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sConnect to a VM via SSH bridge%s\n", "ssh", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sList all life forms currently in the nest%s\n", "ls", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sInspect a specific VM's neural links%s\n", "info", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sPut a VM into deep sleep (graceful stop)%s\n", "stop", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sEvict a VM from the nest permanently%s\n", "delete", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sVaporize all stopped VMs%s\n", "prune", ui.Dim, ui.Reset)

	fmt.Printf("\n%sSTORAGE & GENETICS%s\n", ui.Bold, ui.Reset)
	fmt.Printf("  %-10s %sArchive a VM into a cold, compressed template%s\n", "template", ui.Dim, ui.Reset)

	fmt.Printf("\n%sSYSTEM OPS%s\n", ui.Bold, ui.Reset)
	fmt.Printf("  %-10s %sRun a system health check%s\n", "doctor", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sDump current genetic configuration%s\n", "config", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sPrepare the MCP handshake for AI agents%s\n", "register", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sCheck the evolutionary state of Nido%s\n", "version", ui.Dim, ui.Reset)
	fmt.Printf("  %-10s %sGenerate shell completion scripts%s\n", "completion", ui.Dim, ui.Reset)

	fmt.Printf("\n%s\"It's not a VM, it's a lifestyle.\"%s\n\n", ui.Dim, ui.Reset)
}

func cmdConfig(cfg *config.Config, path string) {
	ui.Header("Nido Genetic Configuration")
	ui.FancyLabel("Config Path", path)
	fmt.Println("")
	ui.FancyLabel("Backup Dir", cfg.BackupDir)
	ui.FancyLabel("Default Tpl", cfg.TemplateDefault)
	ui.FancyLabel("SSH User", cfg.SSHUser)
}

func cmdRegister() {
	exe, _ := os.Executable()
	ui.Header("MCP Handshake Protocol")
	ui.Info("To authorize your AI Agent (Claude, Antigravity) to access the Nest,")
	ui.Info("copy the following JSON blob into your agent's configuration file:")
	fmt.Println("")
	fmt.Printf("%s%s%s%s\n", ui.Dim, ui.Bold, `{
  "mcpServers": {
    "nido-local-vm-manager": {
      "command": "`+exe+`",
      "args": ["mcp"]
    }
  }
}`, ui.Reset)
	fmt.Println("")
}

func cmdMcpJsonList(prov provider.VMProvider) {
	vms, _ := prov.List()
	if vms == nil {
		vms = []provider.VMStatus{}
	}
	data, _ := json.Marshal(vms)
	fmt.Println(string(data))
}

func cmdVersion() {
	fmt.Printf("%sNido %s%s%s (State: Evolved)\n", ui.Dim, ui.Reset, ui.Bold, Version)
	ui.Ironic("Hypervisor handshake protocol v3.0 stable.")
}
