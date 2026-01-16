package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Josepavese/nido/internal/image"
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
				"version": "3.0.0",
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
	tools := []map[string]interface{}{
		{
			"name":        "vm_list",
			"description": "List all virtual machines in the nest",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_create",
			"description": "Create a new VM from a template or cloud image",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":      map[string]interface{}{"type": "string", "description": "Name of the new VM"},
					"template":  map[string]interface{}{"type": "string", "description": "Template to use (e.g. template-headless)"},
					"image":     map[string]interface{}{"type": "string", "description": "Cloud image to pull and use (e.g. ubuntu:24.04)"},
					"user_data": map[string]interface{}{"type": "string", "description": "Optional cloud-init user-data content"},
					"gui":       map[string]interface{}{"type": "boolean", "description": "Enable GUI (VNC) for graphical desktop environments"},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "vm_start",
			"description": "Start a virtual machine",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Name of the VM to start"},
					"gui":  map[string]interface{}{"type": "boolean", "description": "Enable GUI (VNC) for graphical desktop environments"},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "vm_stop",
			"description": "Stop a virtual machine elegantly",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Name of the VM to stop"},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "vm_delete",
			"description": "Evict a VM from the nest permanently",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Name of the VM to delete"},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "vm_info",
			"description": "Inspect a specific VM's neural links and status",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Name of the VM to inspect"},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "vm_ssh",
			"description": "Get the SSH connection string for a VM",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Name of the VM"},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "vm_template_list",
			"description": "List all available VM templates in cold storage",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_template_create",
			"description": "Archive an existing VM into a cold template",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"vm_name":       map[string]interface{}{"type": "string", "description": "Name of the source VM"},
					"template_name": map[string]interface{}{"type": "string", "description": "Name for the new template"},
				},
				"required": []string{"vm_name", "template_name"},
			},
		},
		{
			"name":        "vm_template_delete",
			"description": "Delete a template from cold storage",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Name of the template to delete"},
				},
				"required": []string{"name"},
			},
		},
		{
			"name":        "vm_doctor",
			"description": "Run system diagnostics to check nest health",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_config_get",
			"description": "Get current Nido configuration",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_prune",
			"description": "Remove all stopped virtual machines",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_images_list",
			"description": "List all available VM images, distinguishing between OFFICIAL (upstream proxies) and NIDO FLAVOURS (pre-configured environments).",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_images_pull",
			"description": "Download a cloud image from the catalog",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"image": map[string]interface{}{"type": "string", "description": "Image name and tag (e.g. ubuntu:24.04)"},
				},
				"required": []string{"image"},
			},
		},
		{
			"name":        "vm_images_update",
			"description": "Force refresh the cloud image catalog",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_cache_list",
			"description": "List all cached cloud images with sizes and metadata",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_cache_info",
			"description": "Get cache statistics (total size, count, age)",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "vm_cache_remove",
			"description": "Remove a specific cached image",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":    map[string]interface{}{"type": "string", "description": "Image name (e.g. ubuntu)"},
					"version": map[string]interface{}{"type": "string", "description": "Image version (e.g. 24.04)"},
				},
				"required": []string{"name", "version"},
			},
		},
		{
			"name":        "vm_cache_prune",
			"description": "Remove all or unused cached images",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"unused_only": map[string]interface{}{"type": "boolean", "description": "Only remove images not used by any VM"},
				},
			},
		},
	}
	s.sendResponse(req.ID, map[string]interface{}{"tools": tools})
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
			Name     string `json:"name"`
			Template string `json:"template"`
			Image    string `json:"image"`
			UserData string `json:"user_data"`
			Gui      bool   `json:"gui"`
		}
		json.Unmarshal(params.Arguments, &args)

		opts := provider.VMOptions{}

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
			home, _ := os.UserHomeDir()
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
				downloader := image.Downloader{}
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

		err = s.Provider.Spawn(args.Name, opts)
		result = fmt.Sprintf("VM %s created successfully.", args.Name)
	case "vm_start":
		var args struct {
			Name string `json:"name"`
			Gui  bool   `json:"gui"`
		}
		json.Unmarshal(params.Arguments, &args)
		err = s.Provider.Start(args.Name, provider.VMOptions{Gui: args.Gui})
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
	case "vm_prune":
		count, e := s.Provider.Prune()
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Pruned %d VMs from the nest.", count)
		}
	case "vm_images_list":
		// Load catalog from default location
		home, _ := os.UserHomeDir()
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

		home, _ := os.UserHomeDir()
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
		downloader := image.Downloader{}
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
		home, _ := os.UserHomeDir()
		imgDir := filepath.Join(home, ".nido", "images")
		cachePath := filepath.Join(imgDir, image.CatalogCacheFile)
		os.Remove(cachePath)
		catalog, e := image.LoadCatalog(imgDir, 0) // Force refresh
		if e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Catalog updated. %d images available.", len(catalog.Images))
		}

	// Cache management tools
	case "vm_cache_list":
		home, _ := os.UserHomeDir()
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
		home, _ := os.UserHomeDir()
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
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		json.Unmarshal(params.Arguments, &args)
		home, _ := os.UserHomeDir()
		imgDir := filepath.Join(home, ".nido", "images")
		catalog, e := image.LoadCatalog(imgDir, image.DefaultCacheTTL)
		if e != nil {
			err = e
			break
		}
		if e := catalog.RemoveCachedImage(imgDir, args.Name, args.Version); e != nil {
			err = e
		} else {
			result = fmt.Sprintf("Removed %s:%s from cache", args.Name, args.Version)
		}
	case "vm_cache_prune":
		var args struct {
			UnusedOnly bool `json:"unused_only"`
		}
		json.Unmarshal(params.Arguments, &args)
		home, _ := os.UserHomeDir()
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
