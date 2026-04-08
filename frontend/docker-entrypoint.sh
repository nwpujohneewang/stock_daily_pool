#!/bin/sh
# Substitute env vars into nginx config
export BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
export PORT="${PORT:-80}"
envsubst '${BACKEND_URL} ${PORT}' < /etc/nginx/conf.d/default.conf.template > /etc/nginx/conf.d/default.conf
exec nginx -g 'daemon off;'
