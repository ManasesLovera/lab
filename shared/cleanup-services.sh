#!/usr/bin/env bash

# Cleanup script to remove legacy images and containers

echo "Reclaiming space by removing unused Docker resources..."

# 1. Purge legacy specific containers if they exist
# Add names here if you want to explicitly remove old services
for container in legacy-proxy legacy-ollama; do
  if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
    echo "Removing legacy container: $container"
    docker stop "$container" 2>/dev/null || true
    docker rm "$container" 2>/dev/null || true
  fi
done

# 2. Run general cleanup
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
if [ -f "$SCRIPT_DIR/docker-cleanup.sh" ]; then
    bash "$SCRIPT_DIR/docker-cleanup.sh"
else
    echo "Running basic prune..."
    docker system prune -f
fi

echo "Cleanup complete."
