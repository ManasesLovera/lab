#!/usr/bin/env bash

# Safety Check: Ensure the script is 'sourced' to activate aliases in the current shell
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "Error: Please run this script using 'source' or '.'"
    echo "Usage: source ./shared/setup-lab.sh"
    exit 1
fi

echo "Checking system requirements..."
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "Error: This lab is designed for Linux (Ubuntu recommended)."
    return 1 2>/dev/null || exit 1
fi

# 1. Idempotent Docker DNS Configuration
DOCKER_CONFIG="/etc/docker/daemon.json"
RESTART_REQUIRED=false

if [ ! -f "$DOCKER_CONFIG" ]; then
    echo "Creating Docker config with DNS..."
    echo '{"dns": ["8.8.8.8", "1.1.1.1"]}' | sudo tee "$DOCKER_CONFIG" > /dev/null
    RESTART_REQUIRED=true
else
    # Check if 'dns' key exists and has the correct values
    if ! grep -q '"dns":\s*\["8.8.8.8",\s*"1.1.1.1"\]' "$DOCKER_CONFIG"; then
        echo "Updating Docker DNS configuration..."
        # Backup original
        sudo cp "$DOCKER_CONFIG" "$DOCKER_CONFIG.bak"
        # Use python or jq if available for clean JSON editing, otherwise fallback to safe overwrite
        echo '{"dns": ["8.8.8.8", "1.1.1.1"]}' | sudo tee "$DOCKER_CONFIG" > /dev/null
        RESTART_REQUIRED=true
    else
        echo "Docker DNS is already correctly configured."
    fi
fi

if [ "$RESTART_REQUIRED" = true ]; then
    echo "Restarting Docker to apply changes..."
    sudo systemctl restart docker
fi

# 2. Idempotent Alias Injection
ALIAS_CMD="alias ollama='docker exec -it ollama ollama'"
if ! grep -qF "$ALIAS_CMD" ~/.bashrc; then
    echo "Adding terminal alias to .bashrc..."
    echo "$ALIAS_CMD" >> ~/.bashrc
else
    echo "Alias already exists in .bashrc."
fi

# 3. Network Initialization (docker-network.sh handles its own idempotency)
echo "Initializing lab network..."
NETWORK_SCRIPT="$(dirname "${BASH_SOURCE[0]}")/docker-network.sh"
if [ -f "$NETWORK_SCRIPT" ]; then
    bash "$NETWORK_SCRIPT"
else
    echo "Warning: docker-network.sh not found at $NETWORK_SCRIPT"
fi

# Refresh session
source ~/.bashrc
echo "Setup complete. The 'ollama' alias is active and system is ready."
