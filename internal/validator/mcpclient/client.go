package mcpclient

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync/atomic"
	"time"
)

// Client provides a minimal JSON-RPC client for the MCP server.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	nextID uint64
}

// Start launches the MCP server subprocess.
func Start(nidoBin string) (*Client, error) {
	cmd := exec.Command(nidoBin, "mcp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	cmd.Stderr = io.Discard // silence server logs to avoid interleaving

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		reader: bufio.NewReader(stdout),
	}

	// Initialize
	if err := c.callRaw("initialize", nil, nil); err != nil {
		c.Stop()
		return nil, fmt.Errorf("initialize failed: %w", err)
	}
	// Consume initialize response
	if _, err := c.readJSON(); err != nil {
		c.Stop()
		return nil, fmt.Errorf("decode init resp: %w", err)
	}

	return c, nil
}

// Stop terminates the MCP subprocess.
func (c *Client) Stop() {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil {
		_ = c.cmd.Process.Kill()
		_ = c.cmd.Wait()
	}
}

// CallTool invokes a tool with the given name and arguments.
func (c *Client) CallTool(name string, args map[string]interface{}) (map[string]interface{}, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": args,
	}
	if err := c.callRaw("tools/call", params, nil); err != nil {
		return nil, err
	}
	raw, err := c.readJSON()
	if err != nil {
		return nil, err
	}
	if errObj, ok := raw["error"].(map[string]interface{}); ok {
		if msg, ok := errObj["message"].(string); ok {
			return nil, fmt.Errorf("tool error: %s", msg)
		}
		return nil, fmt.Errorf("tool error: unknown")
	}
	if result, ok := raw["result"].(map[string]interface{}); ok {
		return result, nil
	}
	return nil, fmt.Errorf("tool error: malformed response")
}

func (c *Client) callRaw(method string, params interface{}, id interface{}) error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"id":      c.next(),
	}
	if params != nil {
		req["params"] = params
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	_, err = c.stdin.Write(payload)
	return err
}

func (c *Client) next() uint64 {
	return atomic.AddUint64(&c.nextID, 1)
}

func (c *Client) readJSON() (map[string]interface{}, error) {
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(line, &resp); err != nil {
			// Skip non-JSON noise (e.g., progress output) and keep listening
			continue
		}
		return resp, nil
	}
}

// CallWithTimeout wraps CallTool with a deadline.
func (c *Client) CallWithTimeout(name string, args map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	type result struct {
		res map[string]interface{}
		err error
	}
	ch := make(chan result, 1)
	go func() {
		r, err := c.CallTool(name, args)
		ch <- result{res: r, err: err}
	}()
	select {
	case out := <-ch:
		return out.res, out.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("call %s timed out", name)
	}
}

// CallMethod issues an arbitrary JSON-RPC method (e.g., tools/list).
func (c *Client) CallMethod(method string, params interface{}, timeout time.Duration) (map[string]interface{}, error) {
	type result struct {
		res map[string]interface{}
		err error
	}
	ch := make(chan result, 1)
	go func() {
		req := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  method,
			"id":      c.next(),
		}
		if params != nil {
			req["params"] = params
		}
		payload, err := json.Marshal(req)
		if err != nil {
			ch <- result{err: err}
			return
		}
		payload = append(payload, '\n')
		if _, err := c.stdin.Write(payload); err != nil {
			ch <- result{err: err}
			return
		}
		resp, err := c.readJSON()
		if err != nil {
			ch <- result{err: err}
			return
		}
		if errObj, ok := resp["error"].(map[string]interface{}); ok {
			returnErr := fmt.Errorf("rpc error: %v", errObj["message"])
			ch <- result{err: returnErr}
			return
		}
		ch <- result{res: resp}
	}()
	select {
	case out := <-ch:
		return out.res, out.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("call %s timed out", method)
	}
}
