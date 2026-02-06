#!/bin/bash
echo "🚀 Starting Nido Validator Test (Timeout: 1m)..."
sudo ./bin/nido-validator --scenario vm-spawn-resources --boot-timeout 1m --fail-fast

echo "✅ Test completed (or timed out handled by validator)."
