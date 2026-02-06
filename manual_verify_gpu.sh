#!/bin/bash
set -e

# Configuration
VM_NAME="gpu-test-vm"
IMAGE="ubuntu:24.04"
# CHANGE THIS to your target PCI device (e.g., 01:00.0)
PCI_ID="$1"

if [ -z "$PCI_ID" ]; then
    echo "Usage: $0 <pci-id>"
    echo "Example: $0 0000:01:00.0"
    echo "List devices with 'lspci'"
    exit 1
fi

echo "--> checks: IOMMU and VFIO status"
if ! grep -qE "intel_iommu=on|amd_iommu=on" /proc/cmdline; then
    echo "WARNING: IOMMU not enabled in kernel cmdline. Passthrough will likely fail."
fi

if ! lsmod | grep -q vfio; then
    echo "WARNING: vfio modules not loaded. Run 'modprobe vfio-pci'."
fi

echo "--> 1. Checking Host State for $PCI_ID"
lspci -s $PCI_ID -k

echo "--> 2. Spawning VM with GPU Passthrough args"
# We deliberately do not use a high-level --gpu flag (Phase 3), but the Raw Args (Phase 1)
nido spawn $VM_NAME --image $IMAGE --memory 4096 \
    --qemu-arg "-device" \
    --qemu-arg "vfio-pci,host=$PCI_ID" \
    --json

echo "--> 3. Verifying Process Arguments"
if pgrep -a qemu | grep -q "vfio-pci,host=$PCI_ID"; then
    echo "SUCCESS: QEMU process is running with vfio-pci argument."
else
    echo "FAIL: QEMU process not found or missing argument."
    nido logs $VM_NAME
    exit 1
fi

echo "--> 4. (Manual) Verify Host Driver Binding"
echo "Check if the device is now bound to vfio-pci on the host:"
lspci -s $PCI_ID -k

echo "--> 5. (Manual) Verify Guest"
echo "SSH into the guest and run 'lspci' to see the passed device."
echo "nido ssh $VM_NAME"

echo "--> DONE. Remember to delete the VM to release the device."
echo "nido delete $VM_NAME"
