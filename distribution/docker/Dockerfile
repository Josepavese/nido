# syntax=docker/dockerfile:1
#
# Nido container image.
#
# The build context contains the Linux binaries taken from the release tarball
# (`nido-linux-amd64/`): nido and nido-validator. QEMU is installed from Debian
# so the image can boot VMs on a host that exposes /dev/kvm.
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive \
    NIDO_HOME=/home/nido/.nido

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      qemu-system-x86 \
      qemu-utils \
      openssh-client \
      sshpass \
      ca-certificates \
      tar \
      iproute2 \
 && rm -rf /var/lib/apt/lists/*

COPY nido /usr/local/bin/nido
COPY nido-validator /usr/local/bin/nido-validator
RUN chmod 0755 /usr/local/bin/nido /usr/local/bin/nido-validator \
 && useradd --create-home --uid 1000 nido \
 && mkdir -p "$NIDO_HOME/bin" "$NIDO_HOME/vms" "$NIDO_HOME/run" "$NIDO_HOME/backups" \
 && chown -R nido:nido /home/nido

LABEL org.opencontainers.image.title="Nido" \
      org.opencontainers.image.description="Local VM manager for AI agents: QEMU/KVM virtual machines through MCP, CLI or TUI." \
      org.opencontainers.image.url="https://github.com/Josepavese/nido" \
      org.opencontainers.image.source="https://github.com/Josepavese/nido" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.vendor="Josepavese"

USER nido
WORKDIR /home/nido
VOLUME ["/home/nido/.nido"]

# `docker run -i --device /dev/kvm ghcr.io/josepavese/nido mcp` speaks MCP on stdio.
ENTRYPOINT ["nido"]
CMD ["--help"]
