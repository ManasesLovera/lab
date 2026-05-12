#!/usr/bin/env bash

# Import Postgres credentials to secrets manager
# Run this after postgres has been initialized for the first time

LAB_DIR="/home/mlovera/lab"
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${CYAN}Importing Postgres credentials to secrets manager...${NC}"

if ! docker exec postgres test -f /var/lib/postgresql/data/.init-credentials 2>/dev/null; then
    echo -e "${YELLOW}No credentials file found. Postgres may not be initialized yet.${NC}"
    echo "Run 'lab up postgres' first, then run this script."
    exit 1
fi

docker exec postgres cat /var/lib/postgresql/data/.init-credentials | while IFS=: read -r db user password; do
    if [[ -n "$db" && -n "$user" && -n "$password" ]]; then
        echo -e "${GREEN}Storing credentials for: $db${NC}"
        secrets store "postgres_${db}_password" "$password"
    fi
done

echo -e "${GREEN}Done! All credentials imported.${NC}"