#!/bin/bash
echo "🚨 EMERGENCY GPU RESTORE TRIGGERED 🚨"
# Kill QEMU/Nido to free device
sudo killall -9 qemu-system-x86_64 nido-validator nido

# Force Rescan (Kernel should reclaim device)
echo "⚡ Triggering PCI Rescan..."
echo 1 | sudo tee /sys/bus/pci/rescan
echo "✅ Rescan signal sent."
