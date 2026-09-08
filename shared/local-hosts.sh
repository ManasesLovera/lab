#!/usr/bin/env bash
#
# local-hosts.sh — sync *.rpi.local hostnames into /etc/hosts.
#
# The nginx proxy (core/proxy/conf.d/proxy.conf) dispatches by server_name, so
# every *.rpi.local service is already reachable at http://<pi-ip>:80 — it is
# only DNS that is missing. This script makes those names resolve on this host
# by writing a managed block into /etc/hosts.
#
# Usage:
#   sudo ./shared/local-hosts.sh
#
# Optional: pass an IP explicitly (defaults to the first hostname -I address).
#   sudo ./shared/local-hosts.sh 10.90.100.139
#
# The managed block is rebuilt from proxy.conf on every run, so removed or
# renamed services are cleaned up automatically. It is delimited by:
#   # BEGIN lab rpi.local
#   # END lab rpi.local
set -euo pipefail

LAB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROXY_CONF="$LAB_DIR/core/proxy/conf.d/proxy.conf"
HOSTS_FILE="/etc/hosts"
BEGIN_MARK="# BEGIN lab rpi.local"
END_MARK="# END lab rpi.local"

PI_IP="${1:-$(hostname -I | awk '{print $1}')}"
if [[ -z "$PI_IP" ]]; then
    echo "error: could not determine the Pi's LAN IP (pass one explicitly)" >&2
    exit 1
fi

# Collect every server_name ending in .rpi.local, in order of appearance.
mapfile -t HOSTNAMES < <(
    grep -oP 'server_name\s+\K[^;]+' "$PROXY_CONF" \
        | tr ' ' '\n' \
        | grep -E '\.rpi\.local$' \
        | sort -u
)

if [[ ${#HOSTNAMES[@]} -eq 0 ]]; then
    echo "no *.rpi.local hostnames found in $PROXY_CONF"
    exit 0
fi

echo "Resolving ${#HOSTNAMES[@]} *.rpi.local hostnames to $PI_IP"
printf '  %s\n' "${HOSTNAMES[@]}"

# Rebuild the managed block in a temp copy, preserving everything else.
TMP_HOSTS="$(mktemp)"
trap 'rm -f "$TMP_HOSTS"' EXIT

sed "/^$BEGIN_MARK\$/,/^$END_MARK\$/d" "$HOSTS_FILE" > "$TMP_HOSTS"
{
    echo "$BEGIN_MARK"
    for h in "${HOSTNAMES[@]}"; do
        echo "$PI_IP $h"
    done
    echo "$END_MARK"
} >> "$TMP_HOSTS"

cp "$HOSTS_FILE" "$HOSTS_FILE.bak"
cat "$TMP_HOSTS" > "$HOSTS_FILE"

echo "done: /etc/hosts updated (backup at $HOSTS_FILE.bak)"