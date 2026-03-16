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

# 2. Idempotent Command Injection
mkdir -p "$HOME/.local/bin"
if [ ! -L "$HOME/.local/bin/lab" ]; then
    echo "Creating symlink for lab CLI in ~/.local/bin..."
    ln -sf "/home/mlovera/lab/shared/lab" "$HOME/.local/bin/lab"
fi

if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo "Adding ~/.local/bin to PATH in .bashrc..."
    echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
    export PATH="$HOME/.local/bin:$PATH"
fi

ALIAS_OLLAMA="alias ollama='docker exec -it ollama ollama'"
ALIAS_OPENCLAW="alias openclaw='docker exec -it openclaw openclaw'"
ALIAS_DOCKGE="alias dockge='docker exec -it dockge bash'"
ALIAS_OPENCLAW_CLI="alias openclaw-cli='docker compose -f /home/mlovera/lab/external/openclaw/docker-compose.yml run --rm openclaw-cli'"

if ! grep -qF "$ALIAS_OLLAMA" ~/.bashrc; then
    echo "Adding ollama alias to .bashrc..."
    echo "$ALIAS_OLLAMA" >> ~/.bashrc
fi
if ! grep -qF "$ALIAS_OPENCLAW" ~/.bashrc; then
    echo "Adding openclaw alias to .bashrc..."
    echo "$ALIAS_OPENCLAW" >> ~/.bashrc
fi
if ! grep -qF "$ALIAS_DOCKGE" ~/.bashrc; then
    echo "Adding dockge alias to .bashrc..."
    echo "$ALIAS_DOCKGE" >> ~/.bashrc
fi
if ! grep -qF "$ALIAS_OPENCLAW_CLI" ~/.bashrc; then
    echo "Adding openclaw-cli alias to .bashrc..."
    echo "$ALIAS_OPENCLAW_CLI" >> ~/.bashrc
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

echo "Starting core infrastructure (postgres, proxy, and dockge)..."
bash "/home/mlovera/lab/shared/lab" up postgres
bash "/home/mlovera/lab/shared/lab" up proxy
bash "/home/mlovera/lab/shared/lab" up dockge

echo "Setup complete. The 'lab', 'ollama', 'openclaw', and 'dockge' aliases are active and the system is ready."
