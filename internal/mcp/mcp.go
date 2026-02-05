package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

// Server handles MCP protocol over stdio.
// It exposes VM management operations as MCP tools for AI agents.
type Server struct {
	Provider provider.VMProvider
}

// NewServer creates a new MCP server with the given VM provider.
func NewServer(p provider.VMProvider) *Server {
	return &Server{Provider: p}
}

// ToolsCatalog returns all MCP tools with descriptions and schemas.
func ToolsCatalog() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "vm_list", "description": "List all virtual machines", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_create", "description": "Create a VM from image or template", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "template": map[string]interface{}{"type": "string"}, "image": map[string]interface{}{"type": "string"}, "user_data": map[string]interface{}{"type": "string"}, "gui": map[string]interface{}{"type": "boolean"}, "cmdline": map[string]interface{}{"type": "string"}, "memory_mb": map[string]interface{}{"type": "integer"}, "vcpus": map[string]interface{}{"type": "integer"}, "ports": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}}}, "required": []string{"name"}}},
		{"name": "vm_start", "description": "Start a VM", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "gui": map[string]interface{}{"type": "boolean"}, "cmdline": map[string]interface{}{"type": "string"}}, "required": []string{"name"}}},
		{"name": "vm_stop", "description": "Stop a VM", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}}},
		{"name": "vm_delete", "description": "Delete a VM", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}}},
		{"name": "vm_info", "description": "Get VM info", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}}},
		{"name": "vm_ssh", "description": "Get SSH command for VM", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}}},
		{"name": "vm_prune", "description": "Prune stopped VMs", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_template_list", "description": "List templates", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_template_create", "description": "Create template from VM", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"vm_name": map[string]interface{}{"type": "string"}, "template_name": map[string]interface{}{"type": "string"}}, "required": []string{"vm_name", "template_name"}}},
		{"name": "vm_template_delete", "description": "Delete template", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}}},
		{"name": "vm_images_list", "description": "List catalog images", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_images_info", "description": "Get catalog info for image", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"image": map[string]interface{}{"type": "string"}}, "required": []string{"image"}}},
		{"name": "vm_images_pull", "description": "Pull image into cache", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"image": map[string]interface{}{"type": "string"}}, "required": []string{"image"}}},
		{"name": "vm_images_remove", "description": "Remove cached image", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"image": map[string]interface{}{"type": "string"}}, "required": []string{"image"}}},
		{"name": "vm_images_update", "description": "Refresh image catalog", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_cache_list", "description": "List cached images", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_cache_info", "description": "Cache stats", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_cache_remove", "description": "Remove cached image", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"image": map[string]interface{}{"type": "string"}}, "required": []string{"image"}}},
		{"name": "vm_cache_prune", "description": "Prune cached images", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}},
		{"name": "vm_port_forward", "description": "Add port forward", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "mapping": map[string]interface{}{"type": "string"}}, "required": []string{"name", "mapping"}}},
		{"name": "vm_port_unforward", "description": "Remove port forward", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "guest_port": map[string]interface{}{"type": "number"}, "protocol": map[string]interface{}{"type": "string"}}, "required": []string{"name", "guest_port", "protocol"}}},
		{"name": "vm_port_list", "description": "List port forwards", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}, "required": []string{"name"}}},
		{"name": "vm_config_update", "description": "Update VM configuration (Next Boot)", "inputSchema": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name":         map[string]interface{}{"type": "string"},
				"memory_mb":    map[string]interface{}{"type": "integer"},
				"vcpus":        map[string]interface{}{"type": "integer"},
				"ssh_port":     map[string]interface{}{"type": "integer"},
				"vnc_port":     map[string]interface{}{"type": "integer"},
				"gui":          map[string]interface{}{"type": "boolean"},
				"ssh_user":     map[string]interface{}{"type": "string"},
				"ssh_password": map[string]interface{}{"type": "string"},
				"cmdline":      map[string]interface{}{"type": "string"},
				"ports":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			},
			"required": []string{"name"},
		}},
		{"name": "nido_build_image", "description": "Build a VM image from a blueprint", "inputSchema": map[string]interface{}{"type": "object", "properties": map[string]interface{}{"blueprint_name": map[string]interface{}{"type": "string"}}, "required": []string{"blueprint_name"}}},
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
	fmt.Fprintf(os.Stderr, "[MCP] Handling method: %s\n", req.Method)
	switch req.Method {
	case "initialize":
		s.sendResponse(req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]string{
				"name":    "nido-local-vm-manager",
				"version": "3.1.0",
			},
		})
	case "notifications/initialized":
		return
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	default:
		s.sendError(req.ID, -32601, "Method not found")
	}
}

func (s *Server) handleToolsList(req JSONRPCRequest) {
	s.sendResponse(req.ID, map[string]interface{}{
		"tools": ToolsCatalog(),
	})
}

func (s *Server) handleToolsCall(req JSONRPCRequest) {
	var params CallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params")
		return
	}

	fmt.Fprintf(os.Stderr, "[MCP] Calling tool: %s\n", params.Name)

	var result string
	var err error

	switch params.Name {
	case "vm_list":
		vms, _ := s.Provider.List()
		data, _ := json.Marshal(vms)
		result = string(data)
	case "vm_create":
		var args struct {
			Name     string   `json:"name"`
			Template string   `json:"template"`
			Image    string   `json:"image"`
			UserData string   `json:"user_data"`
			Gui      bool     `json:"gui"`
			Cmdline  string   `json:"cmdline"`
			MemoryMB int      `json:"memory_mb"`
			VCPUs    int      `json:"vcpus"`
			Ports    []string `json:"ports"`
		}
		json.Unmarshal(params.Arguments, &args)

		opts := provider.VMOptions{}

		// Parse ports if provided
		for _, ps := range args.Ports {
			pf, errp := parsePortString(ps)
			if errp != nil {
				err = errp
				break
			}
			opts.Forwarding = append(opts.Forwarding, pf)
		}
		if err != nil {
			break
		}

		// Handle UserData
		if args.UserData != "" {
			tmpDir, _ := os.MkdirTemp("", "nido-mcp-*")
			tmpFile := filepath.Join(tmpDir, "user-data")
			os.WriteFile(tmpFile, []byte(args.UserData), 0644)
			opts.UserDataPath = tmpFile
			defer os.RemoveAll(tmpDir)
		}

		// Handle Image/Template
		if args.Image != "" {
			// Resolve and Pull logic (simplified for MCP for now, or just pass as DiskPath if Provider handles it?)
			// Looking at provider/qemu.go:Spawn, it expects a path.
			// So we MUST resolve/pull here in MCP layer if we want parity.
			home, _ := sysutil.UserHome()
			imgDir := filepath.Join(home, ".nido", "images")
			catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
			if e != nil {
				err = e
				break
			}

			pName, pVer := args.Image, ""
			if strings.Contains(args.Image, ":") {
				parts := strings.Split(args.Image, ":")
				pName, pVer = parts[0], parts[1]
			}

			img, ver, e := catalog.FindImage(pName, pVer)
			if e != nil {
				err = e
				break
			}

			imgPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
			if _, e := os.Stat(imgPath); os.IsNotExist(e) {
				downloader := image.Downloader{Quiet: true}
				downloadPath := imgPath
				isCompressed := strings.HasSuffix(ver.URL, ".tar.xz")
				if isCompressed {
					downloadPath = imgPath + ".tar.xz"
				}

				var downloadErr error
				if len(ver.PartURLs) > 0 {
					downloadErr = downloader.DownloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes)
				} else {
					downloadErr = downloader.Download(ver.URL, downloadPath, ver.SizeBytes)
				}

				if downloadErr != nil {
					err = downloadErr
					break
				}

				if isCompressed {
					// Verify archive
					if e := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); e != nil {
						os.Remove(downloadPath)
						err = fmt.Errorf("archive verification failed: %w", e)
						break
					}
					// Decompress
					if e := downloader.Decompress(downloadPath, imgPath); e != nil {
						os.Remove(downloadPath)
						err = fmt.Errorf("decompression failed: %w", e)
						break
					}
					os.Remove(downloadPath)
				} else {
					// Standard verify
					if e := image.VerifyChecksum(imgPath, ver.Checksum, ver.ChecksumType); e != nil {
						os.Remove(imgPath)
						err = fmt.Errorf("image verification failed: %w", e)
						break
					}
				}
			}
			opts.DiskPath = imgPath
		} else if args.Template != "" {
			opts.DiskPath = args.Template
		}
		opts.Gui = args.Gui
		opts.Cmdline = args.Cmdline
		opts.MemoryMB = args.MemoryMB
		opts.VCPUs = args.VCPUs

		err = s.Provider.Spawn(args.Name, opts)
		result = fmt.Sprintf("VM %s created successfully.", args.Name)
	case "vm_start":
		var args struct {
			Name    string `json:"name"`
			Gui     bool   `json:"gui"`
			Cmdline string `json:"cmdline"`
		}
		json.Unmarshal(params.Arguments, &args)
		err = s.Provider.Start(args.Name, provider.VMOptions{Gui: args.Gui, Cmdline: args.Cmdline})
		result = fmt.Sprintf("VM %s started.", args.Name)
	case "vm_stop":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal(params.Arguments, &args)
		err = s.Provider.Stop(args.Name, true)
		result = fmt.Sprintf("VM %s stopped.", args.Name)
	case "vm_delete":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal(params.Arguments, &args)
		err = s.Provider.Delete(args.Name)
		result = fmt.Sprintf("VM %s deleted.", args.Name)
	case "vm_info":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal(params.Arguments, &args)
		info, e := s.Provider.Info(args.Name)
		if e != nil {
			err = e
		} else {
			data, _ := json.Marshal(info)
			result = string(data)
		}
	case "vm_ssh":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal(params.Arguments, &args)
		result, err = s.Provider.SSHCommand(args.Name)
	case "vm_template_list":
		tpls, e := s.Provider.ListTemplates()
		if e != nil {
			err = e
		} else {
			data, _ := json.Marshal(tpls)
			result = string(data)
		}
	case "vm_template_create":
		var args struct {
			VMName       string `json:"vm_name"`
			TemplateName string `json:"template_name"`
		}
		json.Unmarshal(params.Arguments, &args)
		path, e := s.Provider.CreateTemplate(args.VMName, args.TemplateName)
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Template created at: %s", path)
		}
	case "vm_template_delete":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal(params.Arguments, &args)
		if e := s.Provider.DeleteTemplate(args.Name, false); e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Template '%s' deleted.", args.Name)
		}
	case "vm_doctor":
		reports := s.Provider.Doctor()
		data, _ := json.Marshal(reports)
		result = string(data)
	case "vm_config_get":
		cfg := s.Provider.GetConfig()
		data, _ := json.Marshal(cfg)
		result = string(data)
	case "vm_config_update":
		// Pointers used to detect presence of fields
		var args struct {
			Name        string   `json:"name"`
			MemoryMB    *int     `json:"memory_mb"`
			VCPUs       *int     `json:"vcpus"`
			SSHPort     *int     `json:"ssh_port"`
			VNCPort     *int     `json:"vnc_port"`
			Gui         *bool    `json:"gui"`
			SSHUser     *string  `json:"ssh_user"`
			SSHPassword *string  `json:"ssh_password"`
			Cmdline     *string  `json:"cmdline"`
			Ports       []string `json:"ports"`
		}
		json.Unmarshal(params.Arguments, &args)

		updates := provider.VMConfigUpdates{
			MemoryMB:    args.MemoryMB,
			VCPUs:       args.VCPUs,
			SSHPort:     args.SSHPort,
			VNCPort:     args.VNCPort,
			Gui:         args.Gui,
			SSHUser:     args.SSHUser,
			SSHPassword: args.SSHPassword,
			Cmdline:     args.Cmdline,
		}

		if len(args.Ports) > 0 {
			var fwd []provider.PortForward
			for _, p := range args.Ports {
				pf, err := provider.ParsePortForward(p)
				if err != nil {
					s.sendError(req.ID, -32602, fmt.Sprintf("Invalid port: %v", err))
					return
				}
				fwd = append(fwd, pf)
			}
			updates.Forwarding = &fwd
		}

		if e := s.Provider.UpdateConfig(args.Name, updates); e != nil {
			err = e
		} else {
			result = fmt.Sprintf("VM %s configuration updated.", args.Name)
		}
	case "vm_prune":
		count, e := s.Provider.Prune()
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Pruned %d VMs from the nest.", count)
		}
	case "vm_images_list":
		// Load catalog from default location
		home, _ := sysutil.UserHome()
		imageDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imageDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
		} else {
			data, _ := json.Marshal(catalog)
			result = string(data)
		}
	case "vm_images_pull":
		var args struct {
			Image string `json:"image"`
		}
		json.Unmarshal(params.Arguments, &args)

		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}

		pName, pVer := args.Image, ""
		if strings.Contains(args.Image, ":") {
			parts := strings.Split(args.Image, ":")
			pName, pVer = parts[0], parts[1]
		}

		img, ver, e := catalog.FindImage(pName, pVer)
		if e != nil {
			err = e
			break
		}

		imgPath := filepath.Join(imgDir, fmt.Sprintf("%s-%s.qcow2", img.Name, ver.Version))
		downloader := image.Downloader{Quiet: true}
		downloadPath := imgPath
		isCompressed := strings.HasSuffix(ver.URL, ".tar.xz")
		if isCompressed {
			downloadPath = imgPath + ".tar.xz"
		}

		var downloadErr error
		if len(ver.PartURLs) > 0 {
			downloadErr = downloader.DownloadMultiPart(ver.PartURLs, downloadPath, ver.SizeBytes)
		} else {
			downloadErr = downloader.Download(ver.URL, downloadPath, ver.SizeBytes)
		}

		if downloadErr != nil {
			err = downloadErr
		} else {
			if isCompressed {
				if e := image.VerifyChecksum(downloadPath, ver.Checksum, ver.ChecksumType); e != nil {
					os.Remove(downloadPath)
					err = fmt.Errorf("archive verification failed: %w", e)
				} else if e := downloader.Decompress(downloadPath, imgPath); e != nil {
					os.Remove(downloadPath)
					err = fmt.Errorf("decompression failed: %w", e)
				} else {
					os.Remove(downloadPath)
					result = fmt.Sprintf("Image %s pulled, verified and decompressed.", args.Image)
				}
			} else {
				if e := image.VerifyChecksum(imgPath, ver.Checksum, ver.ChecksumType); e != nil {
					os.Remove(imgPath)
					err = fmt.Errorf("image verification failed: %w", e)
				} else {
					result = fmt.Sprintf("Image %s pulled and verified.", args.Image)
				}
			}
		}
	case "vm_images_update":
		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		cachePath := filepath.Join(imgDir, image.CatalogCacheFile)
		os.Remove(cachePath)
		catalog, e := image.LoadCatalog(imgDir, 0) // Force refresh
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Catalog updated. %d images available.", len(catalog.Images))
		}
	case "vm_images_info":
		var args struct {
			Image string `json:"image"`
		}
		json.Unmarshal(params.Arguments, &args)
		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}
		pName, pVer := args.Image, ""
		if strings.Contains(args.Image, ":") {
			parts := strings.Split(args.Image, ":")
			pName, pVer = parts[0], parts[1]
		}
		_, ver, e := catalog.FindImage(pName, pVer)
		if e != nil {
			err = e
			break
		}
		data, _ := json.Marshal(ver)
		result = string(data)
	case "vm_images_remove":
		var args struct {
			Image string `json:"image"`
		}
		json.Unmarshal(params.Arguments, &args)
		name, ver := args.Image, ""
		if strings.Contains(args.Image, ":") {
			parts := strings.Split(args.Image, ":")
			name, ver = parts[0], parts[1]
		}
		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}
		if ver == "" {
			ver = "latest"
		}
		if e := catalog.RemoveCachedImage(imgDir, name, ver); e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Removed %s:%s from cache", name, ver)
		}

	// Cache management tools
	case "vm_cache_list":
		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}
		cached, e := catalog.GetCachedImages(imgDir)
		if e != nil {
			err = e
		} else {
			data, _ := json.Marshal(cached)
			result = string(data)
		}
	case "vm_cache_info":
		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}
		stats, e := catalog.GetCacheStats(imgDir)
		if e != nil {
			err = e
		} else {
			data, _ := json.Marshal(stats)
			result = string(data)
		}
	case "vm_cache_remove":
		var args struct {
			Image string `json:"image"`
		}
		json.Unmarshal(params.Arguments, &args)
		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}
		name, ver := args.Image, ""
		if strings.Contains(args.Image, ":") {
			parts := strings.Split(args.Image, ":")
			name, ver = parts[0], parts[1]
		}
		if ver == "" {
			ver = "latest"
		}
		if e := catalog.RemoveCachedImage(imgDir, name, ver); e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Removed %s:%s from cache", name, ver)
		}
	case "vm_cache_prune":
		var args struct {
			UnusedOnly bool `json:"unused_only"`
		}
		json.Unmarshal(params.Arguments, &args)
		home, _ := sysutil.UserHome()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}
		// Get list of active VMs if needed
		var activeVMs []string
		if args.UnusedOnly {
			vms, _ := s.Provider.List()
			for _, vm := range vms {
				activeVMs = append(activeVMs, vm.Name)
			}
		}
		removed, e := catalog.PruneCache(imgDir, args.UnusedOnly, activeVMs)
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Pruned %d cached image(s)", removed)
		}

	case "vm_port_forward":
		var args struct {
			Name    string `json:"name"`
			Mapping string `json:"mapping"`
		}
		json.Unmarshal(params.Arguments, &args)
		pf, errp := parsePortString(args.Mapping)
		if errp != nil {
			err = errp
			break
		}
		res, e := s.Provider.PortForward(args.Name, pf)
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Port %d forwarded to host %d/%s.", res.GuestPort, res.HostPort, res.Protocol)
		}

	case "vm_port_unforward":
		var args struct {
			Name      string `json:"name"`
			GuestPort int    `json:"guest_port"`
			Protocol  string `json:"protocol"`
		}
		json.Unmarshal(params.Arguments, &args)
		err = s.Provider.PortUnforward(args.Name, args.GuestPort, args.Protocol)
		result = fmt.Sprintf("Port %d/%s mapping removed.", args.GuestPort, args.Protocol)

	case "vm_port_list":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal(params.Arguments, &args)
		list, e := s.Provider.PortList(args.Name)
		if e != nil {
			err = e
		} else {
			data, _ := json.Marshal(list)
			result = string(data)
		}

	case "nido_build_image":
		var args struct {
			BlueprintName string `json:"blueprint_name"`
		}
		json.Unmarshal(params.Arguments, &args)

		// Locate Blueprint
		home, _ := sysutil.UserHome()
		cwd, _ := os.Getwd()
		searchPaths := []string{
			filepath.Join(cwd, "registry", "blueprints", args.BlueprintName+".yaml"),
			filepath.Join(home, ".nido", "blueprints", args.BlueprintName+".yaml"),
		}

		var bpPath string
		for _, p := range searchPaths {
			if _, e := os.Stat(p); e == nil {
				bpPath = p
				break
			}
		}

		if bpPath == "" {
			err = fmt.Errorf("blueprint '%s' not found", args.BlueprintName)
		} else {
			// Load Blueprint
			bp, e := builder.LoadBlueprint(bpPath)
			if e != nil {
				err = e
			} else {
				// Init Engine
				nidoDir := filepath.Join(home, ".nido")
				cacheDir := filepath.Join(nidoDir, "cache")
				workDir := filepath.Join(nidoDir, "tmp")
				imageDir := filepath.Join(nidoDir, "images")
				eng := builder.NewEngine(cacheDir, workDir, imageDir)

				// Run Build
				// Note: Build is long-running. MCP normally expects synchronous response or async handling.
				// For now, we block, but set a generous timeout in client if needed.
				if e := eng.Build(bp); e != nil {
					err = e
				} else {
					result = fmt.Sprintf("Build complete: %s created.", bp.OutputImage)
				}
			}
		}

	default:
		s.sendError(req.ID, -32601, "Tool not found")
		return
	}

	if err != nil {
		s.sendResponse(req.ID, map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": "Error: " + err.Error()}},
			"isError": true,
		})
	} else {
		s.sendResponse(req.ID, map[string]interface{}{
			"content": []map[string]interface{}{{"type": "text", "text": result}},
		})
	}
}

func (s *Server) sendResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, _ := json.Marshal(resp)
	fmt.Fprintf(os.Stderr, "[MCP] Sending response: %s\n", string(data))
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
	fmt.Fprintf(os.Stderr, "[MCP] Sending error: %s\n", string(data))
	os.Stdout.Write(data)
	os.Stdout.Write([]byte("\n"))
}

// parsePortString is a helper for MCP to reuse parsing logic.
// Implements Section 5.1 of advanced-port-forwarding.md for MCP.
func parsePortString(val string) (provider.PortForward, error) {
	pf := provider.PortForward{Protocol: "tcp"}

	// Split label if present
	if strings.Contains(val, ":") {
		parts := strings.SplitN(val, ":", 2)
		if _, err := provider.ParseInt(parts[0]); err != nil {
			pf.Label = parts[0]
			val = parts[1]
		}
	}

	// Handle protocol
	if strings.Contains(val, "/") {
		parts := strings.SplitN(val, "/", 2)
		pf.Protocol = strings.ToLower(parts[1])
		val = parts[0]
	}

	// Handle Guest:Host
	if strings.Contains(val, ":") {
		parts := strings.SplitN(val, ":", 2)
		gp, err := provider.ParseInt(parts[0])
		if err != nil {
			return pf, fmt.Errorf("invalid guest port: %v", err)
		}
		hp, err := provider.ParseInt(parts[1])
		if err != nil {
			return pf, fmt.Errorf("invalid host port: %v", err)
		}
		pf.GuestPort = gp
		pf.HostPort = hp
	} else {
		gp, err := provider.ParseInt(val)
		if err != nil {
			return pf, fmt.Errorf("invalid port: %v", err)
		}
		pf.GuestPort = gp
	}

	return pf, nil
}
