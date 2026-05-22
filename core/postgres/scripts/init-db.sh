#!/bin/bash

set -e
set -u

CREDENTIALS_FILE="/var/lib/postgresql/data/.init-credentials"

function create_user_and_database() {
	local database=$1
	local password_var="${database}_PASSWORD"
	local password="${!password_var}"
	
	if [[ -z "$password" ]]; then
		password="$POSTGRES_PASSWORD"
	fi
	
	echo "  Creating user and database '$database'"
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
	    CREATE USER $database WITH PASSWORD '$password';
	    CREATE DATABASE $database;
	    GRANT ALL PRIVILEGES ON DATABASE $database TO $database;
	EOSQL
	# Grant permissions on the public schema for the new database
	psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" -d "$database" -c "GRANT ALL ON SCHEMA public TO $database;"
	
	# Log credentials for secrets manager
	echo "$database:$database:$password" >> "$CREDENTIALS_FILE"
}

# Log superuser credentials for secrets manager
echo "admin:$POSTGRES_USER:$POSTGRES_PASSWORD" > "$CREDENTIALS_FILE"

if [ -n "$POSTGRES_MULTIPLE_DATABASES" ]; then
	echo "Multiple database creation requested: $POSTGRES_MULTIPLE_DATABASES"
	for db in $(echo $POSTGRES_MULTIPLE_DATABASES | tr ',' ' '); do
		create_user_and_database "$db"
	done
	echo "Multiple databases created"
fi

echo "Credentials logged to $CREDENTIALS_FILE"