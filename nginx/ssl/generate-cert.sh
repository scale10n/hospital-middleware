#!/bin/sh
set -e

DIR="$(cd "$(dirname "$0")" && pwd)"
DOMAIN="${1:-localhost}"
DAYS="${2:-${DAYS:-365}}"

if [ -f "$DIR/server.crt" ] && [ -f "$DIR/server.key" ]; then
  echo "Notice: Existing certificates found. Overwriting..."
fi

echo "Generating SSL certificate for: $DOMAIN (valid for $DAYS days)"

SAN="DNS:$DOMAIN,IP:127.0.0.1,IP:::1"
if [ "$DOMAIN" != "localhost" ]; then
  SAN="DNS:$DOMAIN,DNS:localhost,IP:127.0.0.1,IP:::1"
fi

if command -v openssl >/dev/null 2>&1; then
  openssl req -x509 -nodes -days "$DAYS" -newkey rsa:2048 \
    -keyout "$DIR/server.key" \
    -out "$DIR/server.crt" \
    -subj "/C=TH/ST=Bangkok/L=Bangkok/O=Hospital/OU=IT/CN=$DOMAIN" \
    -addext "subjectAltName=$SAN"
elif command -v docker >/dev/null 2>&1; then
  echo "openssl not found on host, using Docker..."
  USER_ID=$(id -u)
  GROUP_ID=$(id -g)
  docker run --rm -v "$DIR":/certs alpine sh -c "
    apk add --no-cache openssl >/dev/null 2>&1 && \
    openssl req -x509 -nodes -days $DAYS -newkey rsa:2048 \
      -keyout /certs/server.key \
      -out /certs/server.crt \
      -subj \"/C=TH/ST=Bangkok/L=Bangkok/O=Hospital/OU=IT/CN=$DOMAIN\" \
      -addext \"subjectAltName=$SAN\" && \
    chown $USER_ID:$GROUP_ID /certs/server.crt /certs/server.key
  "
else
  echo "Error: Neither 'openssl' nor 'docker' is installed." >&2
  exit 1
fi

chmod 600 "$DIR/server.key"
chmod 644 "$DIR/server.crt"

echo "Certificates generated successfully:"
echo " - $DIR/server.crt"
echo " - $DIR/server.key"
