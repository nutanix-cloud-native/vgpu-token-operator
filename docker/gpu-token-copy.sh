#!/usr/bin/env sh

while true; do
  if ! cmp -s /config/client_configuration_token.tok /host-token/client_configuration_token.tok; then
    echo "Detected token change, propagating..."
    cp /config/client_configuration_token.tok /host-token/.tmp &&
      mv /host-token/.tmp /host-token/client_configuration_token.tok &&
      chmod 644 /host-token/client_configuration_token.tok &&
      echo "Token updated successfully at $(date)"
  fi
  sleep 2
done
