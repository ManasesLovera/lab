#!/usr/bin/env bash

# Cleanup script to remove openclaw, dockge, and openproject images and containers

echo "Cleaning up openclaw, dockge, and openproject containers and images..."

# Stop and remove containers
for container in openclaw dockge openproject; do
  if docker ps -a --format '{{.Names}}' | grep -q "^${container}$"; then
    echo "Stopping and removing container: $container"
    docker stop "$container" 2>/dev/null || true
    docker rm "$container" 2>/dev/null || true
  fi
done

# Remove images
for image in louislam/dockge openproject/openproject; do
  if docker images --format '{{.Repository}}:{{.Tag}}' | grep -q "^${image}$"; then
    echo "Removing image: $image"
    docker rmi "$image" 2>/dev/null || true
  fi
done

echo "Cleanup complete."
