#!/usr/bin/env bash
# Build a static Linux binary and ship it to the Hetzner box.
#
# Usage:
#   VIEKO_SSH_DEPLOY_HOST=vieko.dev VIEKO_SSH_ADMIN_PORT=2200 ./deploy/deploy.sh
#
# Assumes one-time server setup is done (user, service, nftables, admin sshd on
# the admin port). See README "Deploy to Hetzner". This just builds, uploads,
# and restarts.
set -euo pipefail

HOST="${VIEKO_SSH_DEPLOY_HOST:?set VIEKO_SSH_DEPLOY_HOST=user@host or host}"
PORT="${VIEKO_SSH_ADMIN_PORT:-2200}"
ARCH="${VIEKO_SSH_ARCH:-amd64}" # Hetzner CX = amd64; CAX (Ampere) = arm64

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

echo "==> building linux/${ARCH} binary"
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
	go build -trimpath -ldflags="-s -w" -o dist/vieko-ssh .

echo "==> uploading to ${HOST}:${PORT}"
scp -P "$PORT" dist/vieko-ssh "${HOST}:/tmp/vieko-ssh.new"
scp -P "$PORT" deploy/vieko-ssh.service "${HOST}:/tmp/vieko-ssh.service"

echo "==> installing + restarting"
ssh -p "$PORT" "$HOST" 'sudo bash -s' <<'REMOTE'
set -euo pipefail
sudo install -m0644 /tmp/vieko-ssh.service /etc/systemd/system/vieko-ssh.service
# atomic-ish swap of the binary
sudo install -m0755 /tmp/vieko-ssh.new /usr/local/bin/vieko-ssh
rm -f /tmp/vieko-ssh.new /tmp/vieko-ssh.service
sudo systemctl daemon-reload
sudo systemctl restart vieko-ssh
sudo systemctl --no-pager --lines=5 status vieko-ssh || true
REMOTE

echo "==> done. test: ssh ${HOST%@*} -p 22  (front door)  or  -p 2222 (direct)"
