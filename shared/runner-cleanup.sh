#!/usr/bin/env bash

# Reclaim space used by self-hosted GitHub Actions runners.
#
# The runner's own updater keeps every version it ever downloads: on each
# self-update it renames bin/externals to bin.<old> and externals.<old>, keeps
# the ~450MB _work/_update payload, and never prunes. This script removes
# everything except the currently linked version, plus old _diag logs.
#
# Usage: ./shared/runner-cleanup.sh [--dry-run] [--keep-logs N] [runner_root ...]
#   --dry-run       show what would be removed, change nothing
#   --keep-logs N   number of _diag logs to keep per runner (default 7)
#
# With no runner_root, every directory under ~/runners containing a .runner
# file is cleaned. Safe to run while the runner is online: directories still
# referenced by a running process are skipped.

set -u

DRY_RUN=0
KEEP_LOGS=7
ROOTS=()

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY_RUN=1 ;;
    --keep-logs) shift; KEEP_LOGS="${1:?--keep-logs needs a value}" ;;
    -h|--help) sed -n '2,17p' "$0" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) ROOTS+=("$1") ;;
  esac
  shift
done

if [ "${#ROOTS[@]}" -eq 0 ]; then
  for dir in "$HOME"/runners/*/; do
    [ -f "${dir}.runner" ] && ROOTS+=("${dir%/}")
  done
fi

if [ "${#ROOTS[@]}" -eq 0 ]; then
  echo "No runner directories found under $HOME/runners"
  exit 0
fi

run() {
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "[dry-run] $*"
  else
    "$@"
  fi
}

# A path is in use when a live process runs from it (cmdline, exe or cwd).
# The runner updater keeps the old version alive until the listener restarts,
# so never delete a version that is still backing a process.
in_use() {
  local path="$1" proc cmdline target
  for proc in /proc/[0-9]*; do
    cmdline="$(tr '\0' ' ' < "$proc/cmdline" 2>/dev/null)" || continue
    case "$cmdline" in *"$path"*) return 0 ;; esac
    for target in exe cwd; do
      case "$(readlink "$proc/$target" 2>/dev/null)" in
        "$path"|"$path"/*) return 0 ;;
      esac
    done
  done
  return 1
}

remove() {
  local path="$1"
  echo "  removing $path"
  run rm -rf -- "$path"
}

prune_logs() {
  local dir="$1" prefix="$2"
  [ -d "$dir" ] || return 0
  ls -1t "$dir/$prefix"* 2>/dev/null | tail -n +"$((KEEP_LOGS + 1))" | while read -r log; do
    remove "$log"
  done
}

total_before="$(du -sh "$HOME/runners" 2>/dev/null | cut -f1)"

for root in "${ROOTS[@]}"; do
  [ -f "$root/.runner" ] || { echo "Skipping $root (not a runner root)"; continue; }
  echo "Runner: $root"

  for kind in bin externals; do
    link="$root/$kind"
    current="$(basename "$(readlink -f "$link" 2>/dev/null)")"
    if [ ! -L "$link" ] || [ -z "$current" ]; then
      echo "  skipping $kind (not a versioned symlink)"
      continue
    fi
    for dir in "$root/$kind".*; do
      [ -d "$dir" ] || continue
      [ "$(basename "$dir")" = "$current" ] && continue
      if in_use "$dir"; then
        echo "  keeping $dir (in use by a running process)"
        continue
      fi
      remove "$dir"
    done
  done

  update_dir="$root/_work/_update"
  if [ -d "$update_dir" ]; then
    if in_use "$update_dir"; then
      echo "  keeping $update_dir (update in progress)"
    else
      remove "$update_dir"
    fi
  fi

  prune_logs "$root/_diag" "Runner_"
  prune_logs "$root/_diag" "Worker_"
done

total_after="$(du -sh "$HOME/runners" 2>/dev/null | cut -f1)"
echo "runners: $total_before -> $total_after"
