#!/usr/bin/env sh

# This script assumes that it will be run in a Daemonset created and configured by pkg/generators
# /config is the mount of the VGPUToken secret
# /host-token is the mount of the hostDirectory created with DirectoryOrCreate
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
