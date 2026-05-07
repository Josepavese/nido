package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Josepavese/nido/internal/build"
	"github.com/Josepavese/nido/internal/builder"
	"github.com/Josepavese/nido/internal/image"
	"github.com/Josepavese/nido/internal/pkg/sysutil"
	"github.com/Josepavese/nido/internal/provider"
)

// JSONRPCRequest represents a standard JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a standard JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// CallParams wraps the tool name and arguments for a tools/call request.
type CallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ResourceReadParams struct {
	URI string `json:"uri"`
}

type PromptGetParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// Server handles MCP protocol over stdio.
// It exposes a compact MCP surface tuned for agentic use.
type Server struct {
	Provider provider.VMProvider
}

// NewServer creates a new MCP server with the given VM provider.
func NewServer(p provider.VMProvider) *Server {
	return &Server{Provider: p}
}

// ToolsCatalog returns the compact MCP tool catalog.
func ToolsCatalog() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "nido_vm",
			"description": "Operate the VM fleet through a single high-power tool. Use actions such as list, info, create, start, stop, delete, ssh, prune, config_update, port_forward, port_unforward, and port_list. Prefer resources like nido://fleet/vms or nido://vm/{name} for inspection when your client supports resources; use this tool for mutations or as a universal fallback.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action":        map[string]interface{}{"type": "string", "enum": []string{"list", "info", "create", "start", "stop", "delete", "ssh", "prune", "config_update", "port_forward", "port_unforward", "port_list"}},
					"name":          map[string]interface{}{"type": "string", "description": "VM name for any action that targets a specific VM."},
					"template":      map[string]interface{}{"type": "string", "description": "Template name for action=create."},
					"image":         map[string]interface{}{"type": "string", "description": "Image tag like ubuntu:24.04 for action=create."},
					"user_data":     map[string]interface{}{"type": "string", "description": "Cloud-init user-data content for action=create."},
					"gui":           map[string]interface{}{"type": "boolean"},
					"cmdline":       map[string]interface{}{"type": "string"},
					"memory_mb":     map[string]interface{}{"type": "integer"},
					"vcpus":         map[string]interface{}{"type": "integer"},
					"ports":         map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}, "description": "Port rules like [\"http:80:30080/tcp\"]."},
					"raw_qemu_args": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"accelerators":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
					"mapping":       map[string]interface{}{"type": "string", "description": "Single port mapping used by action=port_forward."},
					"guest_port":    map[string]interface{}{"type": "integer", "description": "Guest port used by action=port_unforward."},
					"protocol":      map[string]interface{}{"type": "string", "description": "Protocol used by action=port_unforward, typically tcp or udp."},
					"ssh_port":      map[string]interface{}{"type": "integer"},
					"vnc_port":      map[string]interface{}{"type": "integer"},
					"ssh_user":      map[string]interface{}{"type": "string"},
				},
				"required": []string{"action"},
			},
		},
		{
			"name":        "nido_template",
			"description": "Manage reusable VM templates with a single tool. Supported actions are list, create, and delete. Use nido://fleet/templates when you only need to inspect template names.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action":        map[string]interface{}{"type": "string", "enum": []string{"list", "create", "delete"}},
					"vm_name":       map[string]interface{}{"type": "string", "description": "Source VM for action=create."},
					"template_name": map[string]interface{}{"type": "string", "description": "Template name for action=create."},
					"name":          map[string]interface{}{"type": "string", "description": "Template name for action=delete."},
				},
				"required": []string{"action"},
			},
		},
		{
			"name":        "nido_image",
			"description": "Manage the image catalog and cache through one namespaced tool. Supported actions are list, info, pull, remove, refresh_catalog, cache_list, cache_info, cache_remove, and cache_prune. Prefer resources like nido://catalog/images, nido://image/{tag}, and nido://storage/cache for inspection to reduce token usage.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action":      map[string]interface{}{"type": "string", "enum": []string{"list", "info", "pull", "remove", "refresh_catalog", "cache_list", "cache_info", "cache_remove", "cache_prune"}},
					"image":       map[string]interface{}{"type": "string", "description": "Image tag like debian:12."},
					"unused_only": map[string]interface{}{"type": "boolean", "description": "Used by action=cache_prune."},
				},
				"required": []string{"action"},
			},
		},
		{
			"name":        "nido_system",
			"description": "Access system-wide Nido operations that are not tied to one VM. Supported actions are doctor, config_get, and build_image. Use resources nido://system/config and nido://system/doctor for read-only inspection when possible.",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action":         map[string]interface{}{"type": "string", "enum": []string{"doctor", "config_get", "build_image"}},
					"blueprint_name": map[string]interface{}{"type": "string", "description": "Blueprint used by action=build_image."},
				},
				"required": []string{"action"},
			},
		},
	}
}

// ResourcesCatalog returns fixed read-only resources.
func ResourcesCatalog() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "Fleet VMs", "uri": "nido://fleet/vms", "mimeType": "application/json", "description": "Compact fleet summary for all known VMs."},
		{"name": "Fleet Templates", "uri": "nido://fleet/templates", "mimeType": "application/json", "description": "Template names available for cloning."},
		{"name": "Image Catalog", "uri": "nido://catalog/images", "mimeType": "application/json", "description": "Compact image catalog summary optimized for agent browsing."},
		{"name": "Cache Summary", "uri": "nido://storage/cache", "mimeType": "application/json", "description": "Cache stats plus cached image entries."},
		{"name": "System Config", "uri": "nido://system/config", "mimeType": "application/json", "description": "Current provider configuration."},
		{"name": "System Doctor", "uri": "nido://system/doctor", "mimeType": "application/json", "description": "Diagnostic report for the local environment."},
	}
}

// ResourceTemplatesCatalog returns parameterized resources for targeted inspection.
func ResourceTemplatesCatalog() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "VM Detail", "uriTemplate": "nido://vm/{name}", "mimeType": "application/json", "description": "Detailed state for one VM."},
		{"name": "Image Detail", "uriTemplate": "nido://image/{tag}", "mimeType": "application/json", "description": "Detailed catalog metadata for one image tag."},
	}
}

// PromptsCatalog returns helper prompts for MCP clients.
func PromptsCatalog() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name":        "nido_task_router",
			"description": "Guidance for using Nido efficiently through MCP with low token overhead.",
			"arguments":   []map[string]interface{}{},
		},
	}
}

// HelpPayload returns the machine-readable MCP usage guide.
func HelpPayload() map[string]interface{} {
	return map[string]interface{}{
		"summary": "Nido exposes a compact MCP surface for agents: use resources for inspection and tools for mutations.",
		"transport": map[string]interface{}{
			"type":    "stdio",
			"command": "nido",
			"args":    []string{"mcp"},
		},
		"discovery": map[string]interface{}{
			"tool_listing_method":              "tools/list",
			"resource_listing_method":          "resources/list",
			"resource_template_listing_method": "resources/templates/list",
			"prompt_listing_method":            "prompts/list",
			"prompt_get_method":                "prompts/get",
			"resource_read_method":             "resources/read",
		},
		"usage_rules": []string{
			"Prefer resources for read-only inspection because they are cheaper and easier for agents to plan around.",
			"Use tools only for mutations or when your MCP client cannot read resources.",
			"Use nido_vm for VM lifecycle, inspection fallback, config changes, and port operations.",
			"Use nido_template for template lifecycle.",
			"Use nido_image for catalog and cache operations.",
			"Use nido_system for doctor, config_get, and build_image.",
			"Every high-power tool requires an action field.",
		},
		"examples": []map[string]interface{}{
			{
				"goal": "List VMs cheaply",
				"call": map[string]interface{}{
					"method": "resources/read",
					"params": map[string]interface{}{"uri": "nido://fleet/vms"},
				},
			},
			{
				"goal": "Inspect one VM",
				"call": map[string]interface{}{
					"method": "resources/read",
					"params": map[string]interface{}{"uri": "nido://vm/agent-01"},
				},
			},
			{
				"goal": "Create a VM from an image",
				"call": map[string]interface{}{
					"method": "tools/call",
					"params": map[string]interface{}{
						"name": "nido_vm",
						"arguments": map[string]interface{}{
							"action": "create",
							"name":   "agent-01",
							"image":  "ubuntu:24.04",
						},
					},
				},
			},
			{
				"goal": "Refresh catalog",
				"call": map[string]interface{}{
					"method": "tools/call",
					"params": map[string]interface{}{
						"name":      "nido_image",
						"arguments": map[string]interface{}{"action": "refresh_catalog"},
					},
				},
			},
		},
		"tools":              ToolsCatalog(),
		"resources":          ResourcesCatalog(),
		"resource_templates": ResourceTemplatesCatalog(),
		"prompts":            PromptsCatalog(),
	}
}

// Serve starts the MCP server loop, reading JSON-RPC requests from stdin
// and writing responses to stdout. Runs until EOF or a fatal error.
func (s *Server) Serve() {
	decoder := json.NewDecoder(os.Stdin)
	for {
		var req JSONRPCRequest
		if err := decoder.Decode(&req); err != nil {
			if err == io.EOF {
				break
			}
			s.sendError(nil, -32700, "Parse error")
			continue
		}
		s.handleRequest(req)
	}
}

func (s *Server) handleRequest(req JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.sendResponse(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools":     map[string]interface{}{},
				"resources": map[string]interface{}{},
				"prompts":   map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "nido-local-vm-manager",
				"version": build.Version,
			},
		})
	case "notifications/initialized":
		return
	case "tools/list":
		s.sendResponse(req.ID, map[string]interface{}{"tools": ToolsCatalog()})
	case "tools/call":
		s.handleToolsCall(req)
	case "resources/list":
		s.sendResponse(req.ID, map[string]interface{}{"resources": ResourcesCatalog()})
	case "resources/templates/list":
		s.sendResponse(req.ID, map[string]interface{}{"resourceTemplates": ResourceTemplatesCatalog()})
	case "resources/read":
		s.handleResourceRead(req)
	case "prompts/list":
		s.sendResponse(req.ID, map[string]interface{}{"prompts": PromptsCatalog()})
	case "prompts/get":
		s.handlePromptGet(req)
	default:
		s.sendError(req.ID, -32601, "Method not found")
	}
}

func (s *Server) handleToolsCall(req JSONRPCRequest) {
	var params CallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	var (
		payload interface{}
		err     error
	)

	switch params.Name {
	case "nido_vm":
		payload, err = s.callVMTool(params.Arguments)
	case "nido_template":
		payload, err = s.callTemplateTool(params.Arguments)
	case "nido_image":
		payload, err = s.callImageTool(params.Arguments)
	case "nido_system":
		payload, err = s.callSystemTool(params.Arguments)
	default:
		s.sendError(req.ID, -32601, "Tool not found")
		return
	}

	if err != nil {
		s.sendResponse(req.ID, map[string]interface{}{
			"content": []map[string]interface{}{{
				"type": "text",
				"text": jsonText(map[string]interface{}{
					"ok":    false,
					"error": err.Error(),
				}),
			}},
			"isError": true,
		})
		return
	}

	s.sendResponse(req.ID, map[string]interface{}{
		"content": []map[string]interface{}{{
			"type": "text",
			"text": jsonText(payload),
		}},
	})
}

func (s *Server) callVMTool(raw json.RawMessage) (interface{}, error) {
	var args struct {
		Action       string   `json:"action"`
		Name         string   `json:"name"`
		Template     string   `json:"template"`
		Image        string   `json:"image"`
		UserData     string   `json:"user_data"`
		Gui          bool     `json:"gui"`
		Cmdline      string   `json:"cmdline"`
		MemoryMB     int      `json:"memory_mb"`
		VCPUs        int      `json:"vcpus"`
		Ports        []string `json:"ports"`
		RawQemuArgs  []string `json:"raw_qemu_args"`
		Accelerators []string `json:"accelerators"`
		Mapping      string   `json:"mapping"`
		GuestPort    int      `json:"guest_port"`
		Protocol     string   `json:"protocol"`
		SSHPort      *int     `json:"ssh_port"`
		VNCPort      *int     `json:"vnc_port"`
		SSHUser      *string  `json:"ssh_user"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}

	switch args.Action {
	case "list":
		vms, err := s.Provider.List()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "list", "vms": vms}, nil
	case "info":
		info, err := s.Provider.Info(args.Name)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "info", "vm": info}, nil
	case "create":
		opts := provider.VMOptions{
			Gui:          args.Gui,
			Cmdline:      args.Cmdline,
			MemoryMB:     args.MemoryMB,
			VCPUs:        args.VCPUs,
			RawQemuArgs:  args.RawQemuArgs,
			Accelerators: args.Accelerators,
		}
		for _, ps := range args.Ports {
			pf, err := parsePortString(ps)
			if err != nil {
				return nil, err
			}
			opts.Forwarding = append(opts.Forwarding, pf)
		}
		if args.UserData != "" {
			tmpDir, _ := os.MkdirTemp("", "nido-mcp-*")
			tmpFile := filepath.Join(tmpDir, "user-data")
			if err := os.WriteFile(tmpFile, []byte(args.UserData), 0644); err != nil {
				return nil, err
			}
			opts.UserDataPath = tmpFile
			defer os.RemoveAll(tmpDir)
		}
		if args.Image != "" {
			imgPath, err := s.resolveImagePath(args.Image)
			if err != nil {
				return nil, err
			}
			opts.DiskPath = imgPath
		} else if args.Template != "" {
			opts.DiskPath = args.Template
		}
		if err := s.Provider.Spawn(args.Name, opts); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "create", "name": args.Name, "source": chooseSource(args.Image, args.Template), "status": "created"}, nil
	case "start":
		if err := s.Provider.Start(args.Name, provider.VMOptions{Gui: args.Gui, Cmdline: args.Cmdline}); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "start", "name": args.Name, "status": "started"}, nil
	case "stop":
		if err := s.Provider.Stop(args.Name, true); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "stop", "name": args.Name, "status": "stopped"}, nil
	case "delete":
		if err := s.Provider.Delete(args.Name); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "delete", "name": args.Name, "status": "deleted"}, nil
	case "ssh":
		cmd, err := s.Provider.SSHCommand(args.Name)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "ssh", "name": args.Name, "command": cmd}, nil
	case "prune":
		count, err := s.Provider.Prune()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "prune", "removed_count": count}, nil
	case "config_update":
		updates := provider.VMConfigUpdates{
			MemoryMB:     intPtrIfPresent(args.MemoryMB, raw, "memory_mb"),
			VCPUs:        intPtrIfPresent(args.VCPUs, raw, "vcpus"),
			SSHPort:      args.SSHPort,
			VNCPort:      args.VNCPort,
			Gui:          boolPtr(args.Gui, raw, "gui"),
			SSHUser:      args.SSHUser,
			Cmdline:      stringPtrIfPresent(args.Cmdline, raw, "cmdline"),
			RawQemuArgs:  slicePtrIfPresent(args.RawQemuArgs, raw, "raw_qemu_args"),
			Accelerators: slicePtrIfPresent(args.Accelerators, raw, "accelerators"),
		}
		if fieldPresent(raw, "ports") {
			var fwd []provider.PortForward
			for _, p := range args.Ports {
				pf, err := provider.ParsePortForward(p)
				if err != nil {
					return nil, fmt.Errorf("invalid port: %w", err)
				}
				fwd = append(fwd, pf)
			}
			updates.Forwarding = &fwd
		}
		if err := s.Provider.UpdateConfig(args.Name, updates); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "config_update", "name": args.Name, "status": "updated"}, nil
	case "port_forward":
		pf, err := parsePortString(args.Mapping)
		if err != nil {
			return nil, err
		}
		res, err := s.Provider.PortForward(args.Name, pf)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "port_forward", "name": args.Name, "forward": res}, nil
	case "port_unforward":
		if err := s.Provider.PortUnforward(args.Name, args.GuestPort, args.Protocol); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "port_unforward", "name": args.Name, "guest_port": args.GuestPort, "protocol": args.Protocol}, nil
	case "port_list":
		list, err := s.Provider.PortList(args.Name)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "port_list", "name": args.Name, "forwarding": list}, nil
	default:
		return nil, fmt.Errorf("unsupported nido_vm action %q", args.Action)
	}
}

func (s *Server) callTemplateTool(raw json.RawMessage) (interface{}, error) {
	var args struct {
		Action       string `json:"action"`
		VMName       string `json:"vm_name"`
		TemplateName string `json:"template_name"`
		Name         string `json:"name"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}
	switch args.Action {
	case "list":
		tpls, err := s.Provider.ListTemplates()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "list", "templates": tpls}, nil
	case "create":
		path, err := s.Provider.CreateTemplate(args.VMName, args.TemplateName)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "create", "vm_name": args.VMName, "template_name": args.TemplateName, "path": path}, nil
	case "delete":
		if err := s.Provider.DeleteTemplate(args.Name, false); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "delete", "name": args.Name, "status": "deleted"}, nil
	default:
		return nil, fmt.Errorf("unsupported nido_template action %q", args.Action)
	}
}

func (s *Server) callImageTool(raw json.RawMessage) (interface{}, error) {
	var args struct {
		Action     string `json:"action"`
		Image      string `json:"image"`
		UnusedOnly bool   `json:"unused_only"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}

	switch args.Action {
	case "list":
		summaries, err := s.imageCatalogSummary()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "list", "images": summaries}, nil
	case "info":
		detail, err := s.imageDetail(args.Image)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "info", "image": detail}, nil
	case "pull":
		if _, err := s.resolveImagePath(args.Image); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "pull", "image": args.Image, "status": "ready"}, nil
	case "remove":
		name, ver := splitImageTag(args.Image)
		if ver == "" {
			ver = "latest"
		}
		catalog, err := s.loadCatalog(image.DefaultCacheTTL)
		if err != nil {
			return nil, err
		}
		if err := catalog.RemoveCachedImage(s.imageDir(), name, ver); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "remove", "image": fmt.Sprintf("%s:%s", name, ver), "status": "removed"}, nil
	case "refresh_catalog":
		cachePath := filepath.Join(s.imageDir(), image.CatalogCacheFile)
		_ = os.Remove(cachePath)
		summaries, err := s.imageCatalogSummaryWithTTL(0)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "refresh_catalog", "image_count": len(summaries)}, nil
	case "cache_list":
		catalog, err := s.loadCatalog(image.DefaultCacheTTL)
		if err != nil {
			return nil, err
		}
		cached, err := catalog.GetCachedImages(s.imageDir())
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "cache_list", "entries": cached}, nil
	case "cache_info":
		catalog, err := s.loadCatalog(image.DefaultCacheTTL)
		if err != nil {
			return nil, err
		}
		stats, err := catalog.GetCacheStats(s.imageDir())
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "cache_info", "stats": stats}, nil
	case "cache_remove":
		name, ver := splitImageTag(args.Image)
		if ver == "" {
			ver = "latest"
		}
		catalog, err := s.loadCatalog(image.DefaultCacheTTL)
		if err != nil {
			return nil, err
		}
		if err := catalog.RemoveCachedImage(s.imageDir(), name, ver); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "cache_remove", "image": fmt.Sprintf("%s:%s", name, ver), "status": "removed"}, nil
	case "cache_prune":
		removed, reclaimed, err := s.Provider.CachePrune(args.UnusedOnly)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "cache_prune", "removed_count": removed, "reclaimed_bytes": reclaimed}, nil
	default:
		return nil, fmt.Errorf("unsupported nido_image action %q", args.Action)
	}
}

func (s *Server) callSystemTool(raw json.RawMessage) (interface{}, error) {
	var args struct {
		Action        string `json:"action"`
		BlueprintName string `json:"blueprint_name"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, err
	}

	switch args.Action {
	case "doctor":
		return map[string]interface{}{"action": "doctor", "reports": s.Provider.Doctor()}, nil
	case "config_get":
		return map[string]interface{}{"action": "config_get", "config": s.Provider.GetConfig()}, nil
	case "build_image":
		home, _ := sysutil.UserHome()
		cwd, _ := os.Getwd()
		searchPaths := []string{
			filepath.Join(cwd, "registry", "blueprints", args.BlueprintName+".yaml"),
			filepath.Join(home, ".nido", "blueprints", args.BlueprintName+".yaml"),
		}

		var bpPath string
		for _, p := range searchPaths {
			if _, err := os.Stat(p); err == nil {
				bpPath = p
				break
			}
		}
		if bpPath == "" {
			return nil, fmt.Errorf("blueprint %q not found", args.BlueprintName)
		}

		bp, err := builder.LoadBlueprint(bpPath)
		if err != nil {
			return nil, err
		}
		nidoDir := filepath.Join(home, ".nido")
		eng := builder.NewEngine(filepath.Join(nidoDir, "cache"), filepath.Join(nidoDir, "tmp"), filepath.Join(nidoDir, "images"))
		if err := eng.Build(bp); err != nil {
			return nil, err
		}
		return map[string]interface{}{"action": "build_image", "blueprint_name": args.BlueprintName, "output_image": bp.OutputImage, "status": "built"}, nil
	default:
		return nil, fmt.Errorf("unsupported nido_system action %q", args.Action)
	}
}

func (s *Server) handleResourceRead(req JSONRPCRequest) {
	var params ResourceReadParams
	if err := json.Unmarshal(req.Params, &params); err != nil || params.URI == "" {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	payload, err := s.readResource(params.URI)
	if err != nil {
		s.sendError(req.ID, -32602, err.Error())
		return
	}

	s.sendResponse(req.ID, map[string]interface{}{
		"contents": []map[string]interface{}{{
			"uri":      params.URI,
			"mimeType": "application/json",
			"text":     jsonText(payload),
		}},
	})
}

func (s *Server) readResource(uri string) (interface{}, error) {
	switch uri {
	case "nido://fleet/vms":
		vms, err := s.Provider.List()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"vms": vms}, nil
	case "nido://fleet/templates":
		tpls, err := s.Provider.ListTemplates()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"templates": tpls}, nil
	case "nido://catalog/images":
		summaries, err := s.imageCatalogSummary()
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"images": summaries}, nil
	case "nido://storage/cache":
		catalog, err := s.loadCatalog(image.DefaultCacheTTL)
		if err != nil {
			return nil, err
		}
		stats, err := catalog.GetCacheStats(s.imageDir())
		if err != nil {
			return nil, err
		}
		entries, err := catalog.GetCachedImages(s.imageDir())
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"stats": stats, "entries": entries}, nil
	case "nido://system/config":
		return map[string]interface{}{"config": s.Provider.GetConfig()}, nil
	case "nido://system/doctor":
		return map[string]interface{}{"reports": s.Provider.Doctor()}, nil
	default:
		if strings.HasPrefix(uri, "nido://vm/") {
			name, err := url.PathUnescape(strings.TrimPrefix(uri, "nido://vm/"))
			if err != nil {
				return nil, err
			}
			vm, err := s.Provider.Info(name)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"vm": vm}, nil
		}
		if strings.HasPrefix(uri, "nido://image/") {
			tag, err := url.PathUnescape(strings.TrimPrefix(uri, "nido://image/"))
			if err != nil {
				return nil, err
			}
			detail, err := s.imageDetail(tag)
			if err != nil {
				return nil, err
			}
			return map[string]interface{}{"image": detail}, nil
		}
		return nil, fmt.Errorf("resource not found")
	}
}

func (s *Server) handlePromptGet(req JSONRPCRequest) {
	var params PromptGetParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}
	if params.Name != "nido_task_router" {
		s.sendError(req.ID, -32602, "Prompt not found")
		return
	}

	s.sendResponse(req.ID, map[string]interface{}{
		"description": "How to use Nido efficiently over MCP.",
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": map[string]interface{}{
					"type": "text",
					"text": "Use resources first for inspection: nido://fleet/vms, nido://vm/{name}, nido://catalog/images, nido://image/{tag}, nido://storage/cache, nido://system/config, and nido://system/doctor. Use tools only when you need to mutate state or when your client cannot read resources. Prefer the compact actions on nido_vm, nido_template, nido_image, and nido_system instead of planning around many micro-tools.",
				},
			},
		},
	})
}

func (s *Server) sendResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

func (s *Server) sendError(id interface{}, code int, message string) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

func (s *Server) loadCatalog(ttl time.Duration) (*image.Catalog, error) {
	return image.LoadCatalog(s.imageDir(), ttl)
}

func (s *Server) imageCatalogSummary() ([]map[string]interface{}, error) {
	return s.imageCatalogSummaryWithTTL(image.DefaultCacheTTL)
}

func (s *Server) imageCatalogSummaryWithTTL(ttl time.Duration) ([]map[string]interface{}, error) {
	catalog, err := s.loadCatalog(ttl)
	if err != nil {
		return nil, err
	}

	var summaries []map[string]interface{}
	for _, img := range catalog.Images {
		for _, ver := range img.Versions {
			imgPath := filepath.Join(s.imageDir(), fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
			_, statErr := os.Stat(imgPath)
			summaries = append(summaries, map[string]interface{}{
				"name":       img.Name,
				"version":    ver.Version,
				"registry":   img.Registry,
				"aliases":    ver.Aliases,
				"downloaded": statErr == nil,
			})
		}
	}
	return summaries, nil
}

func (s *Server) imageDetail(tag string) (map[string]interface{}, error) {
	catalog, err := s.loadCatalog(image.DefaultCacheTTL)
	if err != nil {
		return nil, err
	}
	name, ver := splitImageTag(tag)
	_, version, err := catalog.FindImage(name, ver)
	if err != nil {
		return nil, err
	}
	data := map[string]interface{}{
		"name":          name,
		"version":       version.Version,
		"url":           version.URL,
		"size_bytes":    version.SizeBytes,
		"checksum":      version.Checksum,
		"checksum_type": version.ChecksumType,
		"aliases":       version.Aliases,
	}
	return data, nil
}

func (s *Server) resolveImagePath(tag string) (string, error) {
	catalog, err := s.loadCatalog(image.DefaultCacheTTL)
	if err != nil {
		return "", err
	}
	name, ver := splitImageTag(tag)
	img, version, err := catalog.FindImage(name, ver)
	if err != nil {
		return "", err
	}

	imgPath := filepath.Join(s.imageDir(), fmt.Sprintf("%s-%s.qcow2", img.Name, version.Version))
	if _, err := os.Stat(imgPath); err == nil {
		return imgPath, nil
	}

	downloader := image.Downloader{Quiet: true}
	if err := image.PrepareLocalImage(*version, imgPath, downloader); err != nil {
		return "", err
	}
	return imgPath, nil
}

func jsonText(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func chooseSource(imageTag, template string) string {
	if imageTag != "" {
		return "image"
	}
	if template != "" {
		return "template"
	}
	return "default"
}

func splitImageTag(tag string) (string, string) {
	if strings.Contains(tag, ":") {
		parts := strings.SplitN(tag, ":", 2)
		return parts[0], parts[1]
	}
	return tag, ""
}

func fieldPresent(raw json.RawMessage, field string) bool {
	var args map[string]json.RawMessage
	if err := json.Unmarshal(raw, &args); err != nil {
		return false
	}
	_, ok := args[field]
	return ok
}

func intPtrIfPresent(v int, raw json.RawMessage, field string) *int {
	if !fieldPresent(raw, field) {
		return nil
	}
	return &v
}

func boolPtr(v bool, raw json.RawMessage, field string) *bool {
	if !fieldPresent(raw, field) {
		return nil
	}
	return &v
}

func stringPtrIfPresent(v string, raw json.RawMessage, field string) *string {
	if !fieldPresent(raw, field) {
		return nil
	}
	return &v
}

func slicePtrIfPresent(v []string, raw json.RawMessage, field string) *[]string {
	if !fieldPresent(raw, field) {
		return nil
	}
	return &v
}

// parsePortString is a helper for MCP to reuse parsing logic.
func parsePortString(val string) (provider.PortForward, error) {
	return provider.ParsePortForward(val)
}

func (s *Server) imageDir() string {
	cfg := s.Provider.GetConfig()
	if cfg.ImageDir != "" {
		return cfg.ImageDir
	}
	home, _ := sysutil.UserHome()
	return filepath.Join(home, ".nido", "images")
}
