#!/usr/bin/env bash

# Import Postgres credentials to secrets manager
# Run this after postgres has been initialized for the first time

LAB_DIR="/home/mlovera/lab"
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${CYAN}Importing Postgres credentials to secrets manager...${NC}"

# Ensure secrets tool is available
SECRETS_TOOL="${LAB_DIR}/shared/secrets"
if [ ! -f "$SECRETS_TOOL" ]; then
    echo -e "${RED}Error: secrets tool not found at $SECRETS_TOOL${NC}"
    exit 1
fi

# Self-healing: if the file is missing, try to regenerate it from env vars
if ! docker exec postgres test -f /var/lib/postgresql/data/.init-credentials 2>/dev/null; then
    echo -e "${YELLOW}No credentials file found. Postgres may already be initialized.${NC}"
    echo -e "Attempting to regenerate it from environment variables..."
    
    PG_USER=$(docker exec postgres sh -c 'echo $POSTGRES_USER')
    PG_PASS=$(docker exec postgres sh -c 'echo $POSTGRES_PASSWORD')
    
    if [[ -n "$PG_USER" && -n "$PG_PASS" ]]; then
        # Initialize file with admin user
        docker exec postgres sh -c "echo \"admin:$PG_USER:$PG_PASS\" > /var/lib/postgresql/data/.init-credentials"
        echo -e "${GREEN}Regenerated .init-credentials for admin user.${NC}"
        
        # Try to regenerate other databases if POSTGRES_MULTIPLE_DATABASES is set
        DBS=$(docker exec postgres sh -c 'echo $POSTGRES_MULTIPLE_DATABASES' | tr ',' ' ')
        for db in $DBS; do
            # Try to get DB-specific password, fallback to global password
            DB_PASS=$(docker exec postgres sh -c "printenv ${db}_PASSWORD")
            if [[ -z "$DB_PASS" ]]; then
                DB_PASS=$PG_PASS
            fi
            docker exec postgres sh -c "echo \"$db:$db:$DB_PASS\" >> /var/lib/postgresql/data/.init-credentials"
            echo -e "${GREEN}Regenerated entry for database: $db${NC}"
        done
    else
        echo -e "${RED}Failed to regenerate. Please ensure Postgres is running and env vars are set.${NC}"
        exit 1
    fi
fi

# Process the credentials file
docker exec postgres cat /var/lib/postgresql/data/.init-credentials | while IFS=: read -r db user password; do
    if [[ -n "$db" && -n "$user" && -n "$password" ]]; then
        echo -e "${GREEN}Storing credentials for: $db${NC}"
        "$SECRETS_TOOL" store "postgres_${db}_password" "$password"
        
        # For the admin superuser, also store the username
        if [[ "$db" == "admin" ]]; then
            "$SECRETS_TOOL" store "postgres_admin_user" "$user"
        fi
    fi
done

echo -e "${GREEN}Done! All credentials imported.${NC}"
