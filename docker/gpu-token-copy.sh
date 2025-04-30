#!/usr/bin/env sh

while true; do
  if ! cmp -s /config/client_configuration_token.tok /host-token/client_configuration_token.tok; then
    echo "Detected token change, propagating..."

    echo "--- Differences ---"
    diff /config/client_configuration_token.tok /host-token/client_configuration_token.tok
    echo "-------------------"

    cp /config/client_configuration_token.tok /host-token/.tmp &&
    mv /host-token/.tmp /host-token/client_configuration_token.tok &&
    chmod 644 /host-token/client_configuration_token.tok &&
    echo "Token updated successfully at $(date)"
  else
    echo "No change detected"
  fi
  sleep 2
done

