#!/usr/bin/env bash
# Create + apply the Hetzner Cloud Firewall for the vieko-ssh box.
#
# This is the network-edge layer that sits IN FRONT of the box's nftables:
# it exposes the public ports to the world and restricts admin SSH (:2200)
# to a single trusted source, so admin is invisible to internet scanners.
#
# Prereqs:
#   - hcloud CLI + an active context (hcloud context create <name> --token-from-env)
#   - the server already exists (default name: vieko-ssh)
#
# Usage:
#   ADMIN_SRC_V4=1.2.3.4/32 ADMIN_SRC_V6=2a01:...::/64 ./deploy/hcloud-firewall.sh
#
# ADMIN_SRC_* is where YOU connect admin SSH from. Here it's the VPN egress
# (egress-pdx). If that IP ever changes you'll lose :2200 until you update the
# rule. Recover via the Hetzner Console (edit rule) or the server's VNC
# Console (bypasses the network firewall entirely).
set -euo pipefail

FW="${FW_NAME:-vieko-ssh}"
SERVER="${SERVER_NAME:-vieko-ssh}"
ADMIN_SRC_V4="${ADMIN_SRC_V4:?set ADMIN_SRC_V4=<your-ip>/32}"
ADMIN_SRC_V6="${ADMIN_SRC_V6:-}"

hcloud firewall create --name "$FW"

# Public: SSH TUI front door (:22 -> :2222 via the box's local DNAT), plus
# Caddy HTTP/HTTPS for the vieko.sh -> vieko.dev redirect and ACME.
hcloud firewall add-rule "$FW" --direction in --protocol tcp  --port 22  --source-ips 0.0.0.0/0 --source-ips ::/0
hcloud firewall add-rule "$FW" --direction in --protocol tcp  --port 80  --source-ips 0.0.0.0/0 --source-ips ::/0
hcloud firewall add-rule "$FW" --direction in --protocol tcp  --port 443 --source-ips 0.0.0.0/0 --source-ips ::/0
hcloud firewall add-rule "$FW" --direction in --protocol icmp --source-ips 0.0.0.0/0 --source-ips ::/0

# Admin SSH: restricted to your source(s) only.
if [ -n "$ADMIN_SRC_V6" ]; then
	hcloud firewall add-rule "$FW" --direction in --protocol tcp --port 2200 --source-ips "$ADMIN_SRC_V4" --source-ips "$ADMIN_SRC_V6"
else
	hcloud firewall add-rule "$FW" --direction in --protocol tcp --port 2200 --source-ips "$ADMIN_SRC_V4"
fi

# NOTE: :2222 is intentionally NOT exposed. The front door is :22, and the
# box's internal DNAT (:22 -> :2222) happens after this firewall.
# Outbound is left unset => Hetzner allows all egress (needed for apt/ACME).

hcloud firewall apply-to-resource "$FW" --type server --server "$SERVER"
hcloud firewall describe "$FW"
