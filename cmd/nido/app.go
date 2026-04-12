package main

import (
	"os"
	"path/filepath"

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
	Qemu       *provider.QemuProvider
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
	qemu := provider.NewQemuProvider(nidoDir, cfg)

	return &appContext{
		NidoDir:    nidoDir,
		Cwd:        cwd,
		ConfigPath: cfgPath,
		Config:     cfg,
		Provider:   qemu,
		Qemu:       qemu,
	}
}

func (a *appContext) ImageDir() string {
	if a.Config != nil && a.Config.ImageDir != "" {
		return a.Config.ImageDir
	}
	return filepath.Join(a.NidoDir, "images")
}
