#!/usr/bin/env bash

# Safety Check: Ensure the script is 'sourced' to activate aliases in the current shell
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "Error: Please run this script using 'source' or '.'"
    echo "Usage: source ./shared/setup-lab.sh"
    exit 1
fi

echo "--- Lab Environment Initialization ---"

# 1. System & Permissions Check
if [[ "$OSTYPE" != "linux-gnu"* ]]; then
    echo "Error: This lab is designed for Linux (Ubuntu recommended)."
    return 1 2>/dev/null || exit 1
fi

# Docker Group Permissions
if ! groups $USER | grep &>/dev/null "\bdocker\b"; then
    echo "Adding $USER to the docker group..."
    sudo usermod -aG docker $USER
    echo "Warning: Docker permissions updated. You may need to run 'newgrp docker' or log out/in."
else
    echo "Docker permissions are correctly configured."
fi

# 2. Idempotent Docker DNS Configuration
DOCKER_CONFIG="/etc/docker/daemon.json"
RESTART_REQUIRED=false

if [ ! -f "$DOCKER_CONFIG" ]; then
    echo "Creating Docker config with DNS..."
    echo '{"dns": ["8.8.8.8", "1.1.1.1"]}' | sudo tee "$DOCKER_CONFIG" > /dev/null
    RESTART_REQUIRED=true
else
    if ! grep -q '"dns":\s*\["8.8.8.8",\s*"1.1.1.1"\]' "$DOCKER_CONFIG"; then
        echo "Updating Docker DNS configuration..."
        sudo cp "$DOCKER_CONFIG" "$DOCKER_CONFIG.bak"
        echo '{"dns": ["8.8.8.8", "1.1.1.1"]}' | sudo tee "$DOCKER_CONFIG" > /dev/null
        RESTART_REQUIRED=true
    fi
fi

if [ "$RESTART_REQUIRED" = true ]; then
    echo "Restarting Docker to apply DNS changes..."
    sudo systemctl restart docker
fi

# 3. Lab CLI & PATH Setup
mkdir -p "$HOME/.local/bin"
mkdir -p "/home/mlovera/lab/shared/data"
for tool in lab secrets; do
    if [ ! -L "$HOME/.local/bin/$tool" ]; then
        echo "Creating symlink for $tool CLI..."
        ln -sf "/home/mlovera/lab/shared/$tool" "$HOME/.local/bin/$tool"
    fi
done

if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo "Adding ~/.local/bin to PATH..."
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
    export PATH="$HOME/.local/bin:$PATH"
fi

# 4. Alias Injection
ALIAS_CLOUDFLARED="alias cloudflared='docker exec -it lab-cloudflared cloudflared'"
if ! grep -qF "$ALIAS_CLOUDFLARED" ~/.bashrc; then
    echo "Adding cloudflared alias..."
    echo "$ALIAS_CLOUDFLARED" >> ~/.bashrc
fi

# 5. Network Initialization
NETWORK_NAME=lab-network
if docker network inspect $NETWORK_NAME >/dev/null 2>&1; then
    echo "Network '$NETWORK_NAME' already exists."
else
    echo "Creating '$NETWORK_NAME'..."
    docker network create $NETWORK_NAME
fi

# Refresh session
source ~/.bashrc

echo "---------------------------------------"
echo "Setup complete. The 'lab', 'secrets', and 'cloudflared' aliases are active."
echo "Note: No services were started. Use 'lab up <name>' to start a project."
ed. Use 'lab up <name>' to start a project."
