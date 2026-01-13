#!/bin/sh
set -e

# Wait for Postgres
echo "Waiting for Postgres at $POSTGRES_HOST:5432..."
until nc -z "$POSTGRES_HOST" "5432"; do
  echo "Postgres not ready, sleeping 1s..."
  sleep 1
done
echo "Postgres is up!"

# Run migrations using goose
if [ -d "sql/schema" ]; then
  echo "Running DB migrations..."
  goose -dir ./sql/schema postgres "postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@db:5432/$POSTGRES_DB?sslmode=disable" up
fi

# Start the app
echo "Starting app..."
exec ./commission-tracker
