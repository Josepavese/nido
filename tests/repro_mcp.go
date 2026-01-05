package main

import (
	"bufio"
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("./nido", "mcp")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Start()

	// 1. Initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0.0"}}}`
	fmt.Fprintln(stdin, initReq)

	scanner := bufio.NewScanner(stdout)
	if scanner.Scan() {
		fmt.Printf("Init Response: %s\n", scanner.Text())
	}

	// 2. Tools List
	listReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	fmt.Fprintln(stdin, listReq)

	if scanner.Scan() {
		fmt.Printf("List Response: %s\n", scanner.Text())
	}

	// 3. Config Get
	configReq := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"config_get","arguments":{}}}`
	fmt.Fprintln(stdin, configReq)

	if scanner.Scan() {
		fmt.Printf("Config Response: %s\n", scanner.Text())
	}

	// 4. PRUNE
	pruneReq := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"vm_prune","arguments":{}}}`
	fmt.Fprintln(stdin, pruneReq)

	if scanner.Scan() {
		fmt.Printf("Prune Response: %s\n", scanner.Text())
	}

	stdin.Close()
	cmd.Wait()
}
