#!/usr/bin/env bash
set -euo pipefail

# Start virtual display
Xvfb "${DISPLAY}" -screen 0 1920x1080x24 -ac +extension GLX +render -noreset &
XVFB_PID=$!
echo "Xvfb started (PID=${XVFB_PID}) on ${DISPLAY}"

# Wait for Xvfb to be ready
sleep 0.5

# Start VNC server for debugging (optional, port 5900)
if [ "${ENABLE_VNC:-0}" = "1" ]; then
  x11vnc -display "${DISPLAY}" -nopw -listen 0.0.0.0 -forever -bg &
  echo "VNC server started on :5900"
fi

# Install Playwright browsers if not already present
if [ "${INSTALL_PLAYWRIGHT:-0}" = "1" ]; then
  python3 -m playwright install chromium
fi

# Keep container alive (the node agent will exec tools into it)
echo "desktop-runner ready"
tail -f /dev/null &

# Trap SIGTERM for graceful shutdown
trap 'kill $XVFB_PID; exit 0' SIGTERM SIGINT

wait
