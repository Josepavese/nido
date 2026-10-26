package main

import (
	"github.com/Josepavese/nido/v4/internal/events"
	"github.com/Josepavese/nido/v4/internal/hardware"
	"github.com/Josepavese/nido/v4/internal/lifecycle"
	"os"
	"path/filepath"

	"github.com/Josepavese/nido/v4/internal/config"
	"github.com/Josepavese/nido/v4/internal/pkg/sysutil"
	"github.com/Josepavese/nido/v4/internal/provider"
)

type appContext struct {
	NidoDir    string
	Cwd        string
	ConfigPath string
	Config     *config.Config
	Provider   provider.VMProvider
	Updater    lifecycle.Updater
	Events     *events.Client
}

func newAppContext() *appContext {
	nidoDir := sysutil.NidoHome()
	cwd, _ := os.Getwd()

	cfgPath := filepath.Join(nidoDir, "config.env")
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = filepath.Join(cwd, "config", "config.env")
	}

	cfg, _ := config.LoadConfig(cfgPath)
	host := hardware.New()
	qemu := provider.NewQemuProvider(nidoDir, cfg, host)
	bus := events.NewClient(nidoDir, provider.ObserveEvents(nidoDir, cfgPath, host))
	qemu.Events = bus.Publish

	return &appContext{
		NidoDir:    nidoDir,
		Events:     bus,
		Cwd:        cwd,
		ConfigPath: cfgPath,
		Config:     cfg,
		Provider:   qemu,
		Updater:    lifecycle.NewUpdater(lifecycle.UpdaterOptions{}),
	}
}

func (a *appContext) ImageDir() string {
	if a.Config != nil && a.Config.ImageDir != "" {
		return a.Config.ImageDir
	}
	return filepath.Join(a.NidoDir, "images")
}
