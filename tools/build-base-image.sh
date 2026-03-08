#!/bin/bash
set -euo pipefail

# Build a pre-provisioned Lima base image with tmux and Claude Code.
# The resulting qcow2 image is saved to ~/.cache/mother/ and used by the coder
# engine to skip provisioning on every VM boot.

CACHE_DIR="${HOME}/.cache/mother"
ARCH="$(uname -m)"
VM_NAME="mother-base-builder-$$"

# Map uname arch to Lima/qcow2 naming
case "$ARCH" in
    arm64|aarch64) ARCH_LABEL="arm64" ;;
    x86_64|amd64)  ARCH_LABEL="amd64" ;;
    *)             echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

IMAGE_PATH="${CACHE_DIR}/base-image-${ARCH_LABEL}.qcow2"

echo "Building base image for ${ARCH_LABEL}..."
echo "Output: ${IMAGE_PATH}"

mkdir -p "$CACHE_DIR"

# Write a minimal Lima config with provisioning
LIMA_CONFIG=$(mktemp /tmp/mother-base-XXXXXX.yaml)
trap 'rm -f "$LIMA_CONFIG"; limactl delete --force "$VM_NAME" 2>/dev/null || true' EXIT

cat > "$LIMA_CONFIG" <<'EOF'
images:
  - location: "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-arm64.img"
    arch: "aarch64"
  - location: "https://cloud-images.ubuntu.com/releases/24.04/release/ubuntu-24.04-server-cloudimg-amd64.img"
    arch: "x86_64"

cpus: 2
memory: "4GiB"
disk: "20GiB"

provision:
  - mode: system
    script: |
      #!/bin/bash
      set -eux
      apt-get update
      apt-get install -y tmux
      curl -fsSL https://claude.ai/install.sh | bash
      cp /root/.local/bin/claude /usr/local/bin/claude
      chmod +x /usr/local/bin/claude
EOF

echo "Starting temporary VM ${VM_NAME} (this will take a few minutes)..."
limactl start --tty=false --name "$VM_NAME" "$LIMA_CONFIG"

echo "Provisioning complete. Stopping VM..."
limactl stop "$VM_NAME"

echo "Extracting disk image..."
DIFFDISK="${HOME}/.lima/${VM_NAME}/diffdisk"
if [ ! -f "$DIFFDISK" ]; then
    echo "Error: diffdisk not found at ${DIFFDISK}" >&2
    exit 1
fi

# Convert the overlay disk (follows backing chain) into a standalone qcow2
qemu-img convert -O qcow2 "$DIFFDISK" "$IMAGE_PATH"

echo "Cleaning up temporary VM..."
limactl delete --force "$VM_NAME"

SIZE=$(du -h "$IMAGE_PATH" | cut -f1)
echo ""
echo "Base image built successfully:"
echo "  Path: ${IMAGE_PATH}"
echo "  Size: ${SIZE}"
echo "  Arch: ${ARCH_LABEL}"
