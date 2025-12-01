#!/usr/bin/env bash
set -euo pipefail

# Simple entrypoint that waits for Mongo to be reachable (TCP) before exec'ing the app.
# It supports MONGO_URI in the form mongodb://host:port and falls back to mongo:27017.

echo "entrypoint: starting with IMPORT_MEALS_ON_START=${IMPORT_MEALS_ON_START:-false}"

if [ -z "${MONGO_URI:-}" ]; then
  echo "MONGO_URI not set, starting app immediately"
  exec /root/app
fi

uri="$MONGO_URI"
# strip prefix
hostport=$(echo "$uri" | sed -E 's#^mongodb://##' | sed -E 's#/.*##')
# strip credentials if present
hostport=$(echo "$hostport" | sed -E 's#.*@##')
host=$(echo "$hostport" | cut -d: -f1)
port=$(echo "$hostport" | cut -s -d: -f2)
if [ -z "$host" ]; then
  host="mongo"
fi
if [ -z "$port" ]; then
  port=27017
fi

echo "entrypoint: waiting for mongo at ${host}:${port} (timeout ${MONGO_WAIT_SECONDS:-30}s)"
max_wait=${MONGO_WAIT_SECONDS:-30}
i=0
while ! bash -c "</dev/tcp/${host}/${port}" >/dev/null 2>&1; do
  i=$((i+1))
  if [ "$i" -ge "$max_wait" ]; then
    echo "entrypoint: timeout waiting for mongo after ${max_wait}s, continuing"
    break
  fi
  sleep 1
done

echo "entrypoint: starting /root/app"
exec /root/app
