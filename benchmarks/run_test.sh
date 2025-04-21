#!/bin/bash

# Directory containing Docker Compose files
DIR_PATH="./tests"

# Only run yml and yaml files
for file in "$DIR_PATH"/*.y*ml; do
  if [ -f "$file" ]; then
    docker compose -f "$file" up --abort-on-container-exit --remove-orphans --force-recreate
    docker compose -f "$file" down
  fi
done