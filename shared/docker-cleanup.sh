#!/usr/bin/env bash

# Docker cleanup script - reclaim disk space

echo "=== Docker Cleanup ==="
echo ""

# 1. Remove exited/stopped containers
echo "Removing stopped containers..."
docker container prune -f

# 2. Remove unused images
echo "Removing unused images..."
docker image prune -a -f

# 3. Remove orphaned volumes
echo "Removing unused volumes..."
docker volume prune -f

# 4. Show reclaimed space
echo ""
echo "=== Current Docker Usage ==="
docker system df

echo ""
echo "Cleanup complete!"
