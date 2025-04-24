#!/bin/bash

WATCH_DIR="/etc/nvidia/ClientConfigToken/"
INOTIFYWAIT_PATH="/usr/bin/inotifywait"

while true; do
  output=$("$INOTIFYWAIT_PATH" -e create,delete,modify "$WATCH_DIR" --format=%e)
  exit_code=$?

  if [ $exit_code -eq 0 ]; then
    event="$output"
    case "$event" in
    CREATE)
      echo "$(date) - Detected CREATE event in $WATCH_DIR. Restarting nvidia-gridd..." >>/var/log/nvidia_token_monitor.log 2>&1
      systemctl restart nvidia-gridd >>/var/log/nvidia_token_monitor.log 2>&1
      ;;
    DELETE)
      echo "$(date) - Detected DELETE event in $WATCH_DIR. Restarting nvidia-gridd..." >>/var/log/nvidia_token_monitor.log 2>&1
      systemctl restart nvidia-gridd >>/var/log/nvidia_token_monitor.log 2>&1
      ;;
    MODIFY)
      echo "$(date) - Detected MODIFY event in $WATCH_DIR. Restarting nvidia-gridd..." >>/var/log/nvidia_token_monitor.log 2>&1
      systemctl restart nvidia-gridd >>/var/log/nvidia_token_monitor.log 2>&1
      ;;
    esac
  else
    echo "$(date) - Error occurred with inotifywait (exit code $exit_code): $output. Restarting monitoring loop in 5 seconds..." >>/var/log/nvidia_token_monitor.log 2>&1
    sleep 5
  fi
  sleep 1
done
