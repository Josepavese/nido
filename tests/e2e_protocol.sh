#!/bin/bash
MCP_BIN="$HOME/.nido/bin/nido"
unset VMOPS_CONFIG

# Helper to send RPC (synchronous for test simplicity, though real MCP is async-ish)
send_rpc() {
    echo "$1"
}

# 1. Create VM
echo "--- Step 1: Create ---"
echo '{"jsonrpc":"2.0", "id": 1, "method": "tools/call", "params": {"name": "vm_create", "arguments": {"name": "test-final", "template": "template-headless"}}}' | $MCP_BIN mcp > /tmp/test_step1.json 2>&1
cat /tmp/test_step1.json
grep -q "success" /tmp/test_step1.json || { echo "Create failed"; exit 1; }

# 2. Start VM
echo "--- Step 2: Start ---"
echo '{"jsonrpc":"2.0", "id": 2, "method": "tools/call", "params": {"name": "vm_start", "arguments": {"name": "test-final"}}}' | $MCP_BIN mcp > /tmp/test_step2.json 2>&1
cat /tmp/test_step2.json
grep -q "started" /tmp/test_step2.json || { echo "Start failed"; exit 1; }

# 3. Info
echo "--- Step 3: Info ---"
echo '{"jsonrpc":"2.0", "id": 3, "method": "tools/call", "params": {"name": "vm_info", "arguments": {"name": "test-final"}}}' | $MCP_BIN mcp > /tmp/test_step3.json 2>&1
cat /tmp/test_step3.json

# 4. Delete
echo "--- Step 4: Delete ---"
echo '{"jsonrpc":"2.0", "id": 4, "method": "tools/call", "params": {"name": "vm_delete", "arguments": {"name": "test-final"}}}' | $MCP_BIN mcp > /tmp/test_step4.json 2>&1
cat /tmp/test_step4.json
grep -q "deleted" /tmp/test_step4.json || { echo "Delete failed"; exit 1; }

echo "--- TEST SUITE PASSED ---"
