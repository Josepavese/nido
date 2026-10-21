package main

import (
	"github.com/Josepavese/nido/internal/hardware"
	"github.com/Josepavese/nido/internal/lifecycle"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Josepavese/nido/internal/config"
	"github.com/Josepavese/nido/internal/pkg/sysutil"
	"github.com/Josepavese/nido/internal/provider"
)

type appContext struct {
	NidoDir    string
	Cwd        string
	ConfigPath string
	Config     *config.Config
	Provider   provider.VMProvider
	Updater    lifecycle.Updater
}

func newAppContext() *appContext {
	home, _ := sysutil.UserHome()
	nidoDir := filepath.Join(home, ".nido")
	cwd, _ := os.Getwd()

	cfgPath := filepath.Join(nidoDir, "config.env")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = filepath.Join(cwd, "config", "config.env")
	}

	cfg, _ := config.LoadConfig(cfgPath)
	qemu := provider.NewQemuProvider(nidoDir, cfg, hardware.New())

	return &appContext{
		NidoDir:    nidoDir,
		Cwd:        cwd,
		ConfigPath: cfgPath,
		Config:     cfg,
		Provider:   qemu,
		Updater:    lifecycle.NewUpdater(&http.Client{Timeout: 30 * time.Second}),
	}
}

func (a *appContext) ImageDir() string {
	if a.Config != nil && a.Config.ImageDir != "" {
		return a.Config.ImageDir
	}
	return filepath.Join(a.NidoDir, "images")
}
