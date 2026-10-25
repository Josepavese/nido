package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Josepavese/nido/v5/internal/config"
	"github.com/Josepavese/nido/v5/internal/hardware"
	"github.com/Josepavese/nido/v5/internal/provider"
	"github.com/Josepavese/nido/v5/internal/tui/app/ops"
	"github.com/Josepavese/nido/v5/internal/tui/app/pages/about"
	configpage "github.com/Josepavese/nido/v5/internal/tui/app/pages/config"
	"github.com/Josepavese/nido/v5/internal/tui/app/pages/fleet"
	"github.com/Josepavese/nido/v5/internal/tui/app/pages/hatchery"
	"github.com/Josepavese/nido/v5/internal/tui/app/pages/registry"
	kit "github.com/Josepavese/nido/v5/internal/tui/kit/app"
	"github.com/Josepavese/nido/v5/internal/tui/kit/view"
	"github.com/Josepavese/nido/v5/internal/tui/kit/widget"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

// The gallery uses real page rendering with offline results. No command returned
// by a page is executed: confirmation, update and uninstall have no side effects.
// Optional JSON is consumed by tests/tui_gallery.py for visual inspection.
func TestScreenGallery(t *testing.T) {
	previousProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(previousProfile) })
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	cfg := &config.Config{Theme: "dark", SSHUser: "fixture", ImageDir: filepath.Join(root, "images"), BackupDir: filepath.Join(root, "backups")}
	prov := provider.NewQemuProvider(root, cfg, hardware.New())
	f, h, r, c := fleet.NewFleet(prov), hatchery.NewHatchery(prov), registry.NewRegistry(prov), configpage.NewConfig(cfg, nil)
	shell := kit.NewApp()
	for _, route := range []struct {
		key  string
		page view.Viewlet
	}{{"FLEET", f}, {"HATCHERY", h}, {"REGISTRY", r}, {"SYSTEM", c}, {"NIDO", about.NewAbout()}} {
		shell.AddRoute(route.key, route.key, "", route.page)
	}
	model := &NidoApp{App: shell, ErrorModal: widget.NewAlertModal("Operation failed", "offline global error", nil)}
	type frame struct {
		Name          string
		Width, Height int
		Content       string
	}
	var frames []frame
	capture := func(name, route string) {
		shell.Shell.SwitchTo(route)
		for _, size := range [][2]int{{120, 36}, {80, 24}, {60, 15}, {40, 10}, {120, 36}} {
			shell.Shell.Resize(size[0], size[1])
			content := model.View()
			if lipgloss.Width(content) > size[0] || lipgloss.Height(content) > size[1] {
				t.Fatalf("%s overflow %dx%d", name, size[0], size[1])
			}
			text := ansi.Strip(content)
			if !strings.Contains(text, "^C quit") {
				t.Fatalf("%s lost exit hint", name)
			}
			if shell.Shell.IsModalActive() && strings.Contains(text, "flow") {
				t.Fatalf("%s advertises blocked route navigation", name)
			}
			frames = append(frames, frame{name, size[0], size[1], content})
		}
	}
	capture("Fleet empty", "FLEET")
	f.Update(ops.VMListMsg{Items: []ops.VMItem{{Name: "robin-città-🐦-long-name", State: "stopped"}}})
	f.Update(ops.VMDetailMsg{Name: "robin-città-🐦-long-name", Detail: provider.VMDetail{Name: "robin-città-🐦-long-name", State: "stopped", IP: "127.0.0.1", MemoryMB: 2048, VCPUs: 2, DiskPath: "/private/fixtures/long-path/robin.qcow2"}})
	capture("Fleet list", "FLEET")
	f.MasterDetail.SetFocus(widget.FocusDetail)
	capture("Fleet detail", "FLEET")
	f.TemplateModal.Show("robin", nil)
	capture("Create template", "FLEET")
	f.TemplateModal.Hide()
	f.ConfirmDelete.Message = "Delete 'robin-città-🐦-long-name'? This cannot be undone."
	f.ConfirmDelete.Show()
	capture("Delete VM", "FLEET")
	f.ConfirmDelete.Hide()
	f.ErrorModal.Message = strings.Repeat("A long provider error with città 🐦. ", 12)
	f.ErrorModal.Show()
	capture("Fleet error", "FLEET")
	f.ErrorModal.Hide()
	f.ModalAccel.List.SetItems([]list.Item{widget.SimpleListItem("None"), widget.SimpleListItem("GPU - città 🐦 device with long name")})
	f.ModalAccel.Show()
	capture("Fleet accelerator", "FLEET")
	f.ModalAccel.Hide()
	capture("Hatchery loading", "HATCHERY")
	h.Update(ops.SourcesLoadedMsg{Sources: []string{"[TEMPLATE] robin"}})
	capture("Hatchery sources", "HATCHERY")
	h.MasterDetail.SetFocus(widget.FocusDetail)
	capture("Incubator", "HATCHERY")
	h.ModalAddPort.Show()
	capture("Add forwarding", "HATCHERY")
	h.ModalAddPort.Hide()
	h.Incubator.AddPort(provider.PortForward{Label: "web-admin", GuestPort: 8080, Protocol: "tcp"})
	h.OpenDeletePortModal(0)
	capture("Remove forwarding", "HATCHERY")
	h.ConfirmDeletePort.Hide()
	h.Update(ops.TemplateUsageMsg{Name: "robin"})
	capture("Delete template", "HATCHERY")
	h.ConfirmDelete.Hide()
	h.ModalAccel.Show()
	capture("Hatchery accelerator empty", "HATCHERY")
	h.ModalAccel.Hide()
	capture("Registry loading", "REGISTRY")
	r.Update(ops.RegistryListMsg{Images: []ops.RegistryImage{{Name: "ubuntu", Version: "24.04", Registry: "nido", Description: strings.Repeat("A useful image. ", 20)}}})
	capture("Registry catalog", "REGISTRY")
	r.MasterDetail.SetFocus(widget.FocusDetail)
	r.Update(view.SelectionMsg{Item: registry.RegistryItem{Name: "ubuntu", Version: "24.04", Provider: "nido"}})
	capture("Registry detail", "REGISTRY")
	r.Update(ops.CacheListMsg{Items: []ops.CacheItem{{Name: "cached-bird", Version: "1", Kind: "cloud"}}})
	r.Update(view.SelectionMsg{Item: registry.RegistryItem{Name: "cached-bird", Version: "1", IsLocal: true}})
	r.Update(tea.KeyMsg{Type: tea.KeyDelete})
	capture("Delete image", "REGISTRY")
	r.ConfirmDelete.Hide()
	for _, key := range []string{"SETTINGS", "APPEARANCE", "UPDATE", "DOCTOR", "CACHE", "UNINSTALL"} {
		c.Pages.SwitchTo(key)
		c.MasterDetail.SetFocus(widget.FocusDetail)
		capture("System "+key, "SYSTEM")
	}
	c.Pages.SwitchTo("SETTINGS")
	c.PageGlobal.Modal.Show()
	capture("Save core", "SYSTEM")
	c.PageGlobal.Modal.Hide()
	c.Pages.SwitchTo("APPEARANCE")
	c.PageAppearance.ThemeModal.Show()
	capture("Theme picker", "SYSTEM")
	c.PageAppearance.ThemeModal.Hide()
	c.PageAppearance.Modal.Show()
	capture("Save appearance", "SYSTEM")
	c.PageAppearance.Modal.Hide()
	c.Pages.SwitchTo("UPDATE")
	c.PageUpdate.ConfirmModal.Show()
	capture("Confirm update", "SYSTEM")
	c.PageUpdate.ConfirmModal.Hide()
	for _, result := range []ops.UpdateCheckMsg{{Manual: true, Current: "1", Latest: "1"}, {Manual: true, Err: errors.New("offline fixture")}} {
		c.Update(result)
		label := "Update check current"
		if result.Err != nil {
			label = "Update check failed"
		}
		capture(label, "SYSTEM")
		c.PageUpdate.ResultModal.Hide()
	}
	for _, err := range []error{nil, errors.New("offline fixture")} {
		c.Update(ops.ApplyUpdateMsg{Err: err})
		label := "Update success"
		if err != nil {
			label = "Update failed"
		}
		capture(label, "SYSTEM")
		c.PageUpdate.ResultModal.Hide()
	}
	c.Pages.SwitchTo("DOCTOR")
	c.PageDoctor.Focus()
	c.Update(ops.DoctorResultMsg{Err: errors.New("offline diagnostic fixture")})
	capture("Doctor error", "SYSTEM")
	c.PageDoctor.Modal.Hide()
	c.Update(ops.DoctorResultMsg{Reports: []ops.DoctorReport{{Label: "Runtime", Passed: true, Details: "offline success fixture"}, {Label: "Storage", Passed: false, Details: strings.Repeat("Long diagnostic detail. ", 15)}}})
	capture("Doctor report", "SYSTEM")
	c.PageDoctor.Modal.Hide()
	c.Pages.SwitchTo("CACHE")
	c.Update(ops.OpResultMsg{Op: "prune", Data: ops.PruneStats{Count: 2, Reclaimed: 2048}})
	capture("Cache result", "SYSTEM")
	c.PageCache.ResultModal.Hide()
	c.Pages.SwitchTo("UNINSTALL")
	c.PageUninstall.Modal.Show()
	capture("Confirm uninstall", "SYSTEM")
	c.PageUninstall.Modal.Hide()
	capture("About", "NIDO")
	model.ErrorModal.Show()
	capture("Global error", "FLEET")
	model.ErrorModal.Hide()
	shell.Shell.UpdateAction("fixture", "Looking after the nest", 0.5)
	capture("Background progress", "SYSTEM")
	h.ModalAddPort.Show()
	capture("Forwarding during progress", "HATCHERY")
	h.ModalAddPort.Hide()
	shell.Shell.FinishAction("fixture")
	if path := os.Getenv("NIDO_TUI_GALLERY_JSON"); path != "" {
		data, err := json.Marshal(frames)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("Gallery: %d rendered frames, offline data, no operation commands executed\n", len(frames))
}
