#!/usr/bin/env bash

# Disk cleanup script - reclaim disk space on the lab host
# Usage: sudo ./shared/disk-cleanup.sh [--dry-run] [--volumes]
#   --dry-run  show what would be removed, change nothing
#   --volumes  also prune unused Docker volumes (may hold data of stopped projects)

set -u

DRY_RUN=0
PRUNE_VOLUMES=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    --volumes) PRUNE_VOLUMES=1 ;;
    *) echo "Unknown option: $arg"; exit 1 ;;
  esac
done

if [ "$(id -u)" -ne 0 ]; then
  echo "Run with sudo: sudo $0 $*"
  exit 1
fi

OWNER="${SUDO_USER:-root}"
OWNER_HOME="$(getent passwd "$OWNER" | cut -d: -f6)"
LAB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

run() {
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "[dry-run] $*"
  else
    "$@"
  fi
}

as_owner() {
  run sudo -u "$OWNER" -H "$@"
}

used() { df -h / | awk 'NR==2 {print $3 " used (" $5 ")"}'; }

# Keep the newest N entries of a directory (by mtime), remove the rest
prune_old() {
  local dir="$1" keep="$2" skip="${3:-}"
  [ -d "$dir" ] || return 0
  ls -1t "$dir" | tail -n +"$((keep + 1))" | while read -r entry; do
    [ "$entry" = "$skip" ] && continue
    echo "  removing $dir/$entry"
    run rm -rf "${dir:?}/$entry"
  done
}

echo "=== Disk Cleanup ==="
echo "Before: $(used)"
echo ""

# 1. Docker
if command -v docker >/dev/null 2>&1; then
  echo "[1/7] Docker: build cache, stopped containers, unused images"
  run docker builder prune -af
  run docker container prune -f
  run docker image prune -a -f
  if [ "$PRUNE_VOLUMES" -eq 1 ]; then
    run docker volume prune -f
  else
    echo "  skipping volumes (use --volumes to prune)"
  fi
fi

# 2. Self-hosted GitHub runners (old versions, update payload, old logs)
echo "[2/8] self-hosted runners"
if [ -x "$LAB_DIR/shared/runner-cleanup.sh" ]; then
  if [ "$DRY_RUN" -eq 1 ]; then
    as_owner "$LAB_DIR/shared/runner-cleanup.sh" --dry-run
  else
    as_owner "$LAB_DIR/shared/runner-cleanup.sh"
  fi
fi

# 3. apt cache and unused packages
echo "[3/8] apt cache"
run apt-get clean
run apt-get autoremove -y

# 4. Logs
echo "[4/8] journal and rotated logs"
run journalctl --vacuum-size=50M
run find /var/log -type f \( -name '*.gz' -o -name '*.[0-9]' \) -delete

# 5. Git worktrees (merged and clean ones under .claude/worktrees)
echo "[5/8] git worktrees"
if git -C "$LAB_DIR" rev-parse --git-dir >/dev/null 2>&1; then
  as_owner git -C "$LAB_DIR" worktree prune
  main_branch="main"
  as_owner git -C "$LAB_DIR" worktree list --porcelain | awk '/^worktree /{print $2}' |
    grep '/\.claude/worktrees/' | while read -r wt; do
      [ "$wt" = "$PWD" ] && continue
      branch="$(git -C "$wt" rev-parse --abbrev-ref HEAD 2>/dev/null)"
      if [ -n "$(git -C "$wt" status --porcelain 2>/dev/null)" ]; then
        echo "  keeping $wt (uncommitted changes)"
      elif git -C "$LAB_DIR" merge-base --is-ancestor "$branch" "$main_branch" 2>/dev/null; then
        echo "  removing merged worktree $wt ($branch)"
        as_owner git -C "$LAB_DIR" worktree remove "$wt"
        as_owner git -C "$LAB_DIR" branch -d "$branch"
      else
        echo "  keeping $wt ($branch not merged into $main_branch)"
      fi
    done
fi

# 6. Old tool versions (keep the newest two Claude versions, current Codex release)
echo "[6/8] old tool versions"
prune_old "$OWNER_HOME/.local/share/claude/versions" 2
codex_releases="$OWNER_HOME/.codex/packages/app-server-daemon/releases"
codex_current="$(basename "$(readlink -f "$OWNER_HOME/.codex/packages/app-server-daemon/current" 2>/dev/null)")"
if [ -d "$codex_releases" ]; then
  for rel in "$codex_releases"/*; do
    [ "$(basename "$rel")" = "$codex_current" ] && continue
    echo "  removing $rel"
    run rm -rf "$rel"
  done
fi

# 7. Package manager caches
echo "[7/8] npm and user caches"
as_owner bash -lc 'command -v npm >/dev/null && npm cache clean --force'
run rm -rf "$OWNER_HOME/.cache/pip" "$OWNER_HOME/.cache/go-build"

echo "[8/8] done"
echo ""
echo "After: $(used)"
echo "Review remaining large dirs: ~/dev ~/runners ~/obscura ~/backups ~/.local/share/opencode"
