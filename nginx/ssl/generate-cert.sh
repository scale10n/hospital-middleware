#!/bin/sh
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
DOMAIN="${1:-localhost}"

echo "Generating SSL certificate for: $DOMAIN, 127.0.0.1"

SAN="DNS:$DOMAIN,IP:127.0.0.1"
if [ "$DOMAIN" != "localhost" ]; then
  SAN="DNS:$DOMAIN,DNS:localhost,IP:127.0.0.1"
fi

if command -v openssl >/dev/null 2>&1; then
  openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$DIR/server.key" \
    -out "$DIR/server.crt" \
    -subj "/C=TH/ST=Bangkok/L=Bangkok/O=Hospital/OU=IT/CN=$DOMAIN" \
    -addext "subjectAltName=$SAN"
elif command -v docker >/dev/null 2>&1; then
  echo "openssl not found on host, using Docker..."
  docker run --rm -v "$DIR":/certs alpine sh -c "
    apk add --no-cache openssl >/dev/null 2>&1 && \
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
      -keyout /certs/server.key \
      -out /certs/server.crt \
      -subj \"/C=TH/ST=Bangkok/L=Bangkok/O=Hospital/OU=IT/CN=$DOMAIN\" \
      -addext \"subjectAltName=$SAN\"
  "
else
  echo "Error: Neither 'openssl' nor 'docker' is installed." >&2
  exit 1
fi

echo "Certificates generated successfully:"
echo " - $DIR/server.crt"
echo " - $DIR/server.key"
