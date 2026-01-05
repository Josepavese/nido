package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

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
			"description": "Create a new VM from a template",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":     map[string]interface{}{"type": "string", "description": "Name of the new VM"},
					"template": map[string]interface{}{"type": "string", "description": "Template to use (e.g. template-headless)"},
				},
				"required": []string{"name", "template"},
			},
		},
		{
			"name":        "vm_start",
			"description": "Start a virtual machine",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Name of the VM to start"},
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
			"name":        "template_list",
			"description": "List all available VM templates in cold storage",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "template_create",
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
			"name":        "doctor",
			"description": "Run system diagnostics to check nest health",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			"name":        "config_get",
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
		}
		json.Unmarshal(params.Arguments, &args)
		err = s.Provider.Spawn(args.Name, provider.VMOptions{DiskPath: args.Template})
		result = fmt.Sprintf("VM %s created successfully.", args.Name)
	case "vm_start":
		var args struct {
			Name string `json:"name"`
		}
		json.Unmarshal(params.Arguments, &args)
		err = s.Provider.Start(args.Name)
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
	case "template_list":
		tpls, e := s.Provider.ListTemplates()
		if e != nil {
			err = e
		} else {
			data, _ := json.Marshal(tpls)
			result = string(data)
		}
	case "template_create":
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
	case "doctor":
		reports := s.Provider.Doctor()
		data, _ := json.Marshal(reports)
		result = string(data)
	case "config_get":
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
