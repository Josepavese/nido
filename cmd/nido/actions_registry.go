package main

import (
	climeta "github.com/Josepavese/nido/internal/cli"
	"github.com/spf13/cobra"
)

func buildActionRegistry(app *appContext) map[string]climeta.ActionFunc {
	return map[string]climeta.ActionFunc{
		"vm.list":                      actionVMList(app),
		"vm.info":                      actionVMInfo(app),
		"vm.spawn":                     actionVMSpawn(app),
		"vm.start":                     actionVMStart(app),
		"vm.stop":                      actionVMStop(app),
		"vm.ssh":                       actionVMSSH(app),
		"vm.delete":                    actionVMDelete(app),
		"vm.prune":                     actionVMPrune(app),
		"ui.gui":                       func(cmd *cobra.Command, args []string) { cmdGUI(app.Provider, app.Config) },
		"template.list":                actionTemplateList(app),
		"template.create":              actionTemplateCreate(app),
		"template.delete":              actionTemplateDelete(app),
		"cache.list":                   actionCacheList(app),
		"cache.info":                   actionCacheInfo(app),
		"cache.remove":                 actionCacheRemove(app),
		"cache.prune":                  actionCachePrune(app),
		"images.list":                  actionImagesList(app),
		"images.pull":                  actionImagesPull(app),
		"images.info":                  actionImagesInfo(app),
		"images.remove":                actionImagesRemove(app),
		"images.update":                actionImagesUpdate(app),
		"build":                        actionBuild(app),
		"system.doctor":                actionDoctor(app),
		"system.accel.list":            actionAccelList(app),
		"system.config":                actionConfig(app),
		"system.config.set":            actionConfigSet(app),
		"system.register":              actionRegister(app),
		"system.version":               actionVersion(app),
		"system.update":                actionUpdate(app),
		"system.uninstall":             actionUninstall(app),
		"system.completion.bash":       actionShellCompletion("bash"),
		"system.completion.zsh":        actionShellCompletion("zsh"),
		"system.completion.fish":       actionShellCompletion("fish"),
		"system.completion.powershell": actionShellCompletion("powershell"),
		"system.mcp_help":              actionMCPHelp(app),
		"system.mcp":                   actionMCP(app),
	}
}
