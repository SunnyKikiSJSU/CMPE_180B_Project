#!/bin/bash
# First-boot setup for the app instance (Amazon Linux 2023): Docker, the
# Compose and Buildx plugins, and git. Deploying the app itself is manual
# (AWS_PLAN.md §4.3). Plugin downloads are pinned and checksum-verified.
set -euo pipefail

dnf install -y docker git
systemctl enable --now docker
usermod -aG docker ec2-user

PLUGINS=/usr/local/lib/docker/cli-plugins
mkdir -p "$PLUGINS"

install_plugin() { # name url sha256
  curl -fsSL --retry 3 -o "/tmp/$1" "$2"
  echo "$3  /tmp/$1" | sha256sum -c -
  install -m 0755 "/tmp/$1" "$PLUGINS/$1"
  rm -f "/tmp/$1"
}

install_plugin docker-compose \
  https://github.com/docker/compose/releases/download/v5.8.0/docker-compose-linux-x86_64 \
  b8840f5008c447f41fce0a57ae3adeab42c89b1eb6e6012524bcabbfb4ffbaf6
install_plugin docker-buildx \
  https://github.com/docker/buildx/releases/download/v0.39.1/buildx-v0.39.1.linux-amd64 \
  903090816da68f8bc1a2ff5346dd163a5adacfce4548d003c75df42cd47a9bf4

docker compose version
docker buildx version
