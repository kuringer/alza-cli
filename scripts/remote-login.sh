#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

DISPLAY_NUM="${ALZA_DISPLAY:-:99}"
VNC_PORT="${ALZA_VNC_PORT:-5901}"
STATE_DIR="${ALZA_STATE_DIR:-$HOME/.config/alza/remote-login}"
PROFILE_DIR="${ALZA_PROFILE_DIR:-$HOME/.config/alza/pw-profile}"
TOKEN_TIMEOUT_MS="${ALZA_TOKEN_TIMEOUT_MS:-900000}"
CDP_PORT="${ALZA_CDP_PORT:-9222}"
LOG_DIR="$STATE_DIR/log"
VNC_PASS_FILE="$STATE_DIR/vnc.pass"
CHROME_PID_FILE="$STATE_DIR/chromium.pid"
REFRESH_MODE="${ALZA_REFRESH_MODE:-cookies}"
CLI_BIN="${ALZA_CLI_BIN:-}"

mkdir -p "$STATE_DIR" "$LOG_DIR"

Xvfb_PID_FILE="$STATE_DIR/xvfb.pid"
X11VNC_PID_FILE="$STATE_DIR/x11vnc.pid"

generate_pass() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -base64 9 | tr -d '\n'
    return
  fi
  LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 12
}

if [ -n "${ALZA_VNC_PASS:-}" ]; then
  VNC_PASS="$ALZA_VNC_PASS"
else
  VNC_PASS="$(generate_pass)"
fi
if [ -z "$VNC_PASS" ]; then
  echo "Failed to generate VNC password. Set ALZA_VNC_PASS explicitly."
  exit 1
fi

is_running() {
  local pid_file="$1"
  [ -f "$pid_file" ] && kill -0 "$(cat "$pid_file")" >/dev/null 2>&1
}

start_xvfb() {
  if is_running "$Xvfb_PID_FILE"; then
    return
  fi
  Xvfb "$DISPLAY_NUM" -screen 0 1280x720x24 -nolisten tcp -ac \
    >"$LOG_DIR/xvfb.log" 2>&1 &
  echo $! >"$Xvfb_PID_FILE"
}

start_x11vnc() {
  if is_running "$X11VNC_PID_FILE"; then
    return
  fi
  x11vnc -storepasswd "$VNC_PASS" "$VNC_PASS_FILE" >/dev/null 2>&1
  x11vnc -display "$DISPLAY_NUM" -localhost -rfbauth "$VNC_PASS_FILE" -rfbport "$VNC_PORT" -shared -forever \
    >"$LOG_DIR/x11vnc.log" 2>&1 &
  echo $! >"$X11VNC_PID_FILE"
}

cleanup_profile_lock() {
  if [ -d "$PROFILE_DIR" ]; then
    pkill -f "--user-data-dir=$PROFILE_DIR" >/dev/null 2>&1 || true
    rm -f "$PROFILE_DIR/SingletonLock" "$PROFILE_DIR/SingletonSocket" "$PROFILE_DIR/SingletonCookie"
  fi
}

start_chromium() {
  if is_running "$CHROME_PID_FILE"; then
    return
  fi

  if command -v chromium >/dev/null 2>&1; then
    CHROME_BIN="chromium"
  elif command -v google-chrome >/dev/null 2>&1; then
    CHROME_BIN="google-chrome"
  else
    echo "Chromium not found. Install chromium or set ALZA_CHROME_BIN."
    exit 1
  fi

  if [ -n "${ALZA_CHROME_BIN:-}" ]; then
    CHROME_BIN="$ALZA_CHROME_BIN"
  fi

  DISPLAY="$DISPLAY_NUM" "$CHROME_BIN" \
    --no-first-run \
    --disable-dev-shm-usage \
    --disable-features=Translate,OptimizationHints \
    --password-store=basic \
    --use-mock-keychain \
    --user-data-dir="$PROFILE_DIR" \
    --remote-debugging-address=127.0.0.1 \
    --remote-debugging-port="$CDP_PORT" \
    "https://www.alza.sk/" \
    >"$LOG_DIR/chromium.log" 2>&1 &

  echo $! >"$CHROME_PID_FILE"
}

cleanup() {
  if [ "${STARTED_X11VNC:-0}" = "1" ] && is_running "$X11VNC_PID_FILE"; then
    kill "$(cat "$X11VNC_PID_FILE")" >/dev/null 2>&1 || true
  fi
  if [ "${STARTED_XVFB:-0}" = "1" ] && is_running "$Xvfb_PID_FILE"; then
    kill "$(cat "$Xvfb_PID_FILE")" >/dev/null 2>&1 || true
  fi
  if is_running "$CHROME_PID_FILE"; then
    kill "$(cat "$CHROME_PID_FILE")" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT

if ! command -v Xvfb >/dev/null 2>&1; then
  echo "Xvfb not found. Install xorg-server-xvfb first."
  exit 1
fi

if ! command -v x11vnc >/dev/null 2>&1; then
  echo "x11vnc not found. Install x11vnc first."
  exit 1
fi

start_xvfb
STARTED_XVFB=1

start_x11vnc
STARTED_X11VNC=1

cleanup_profile_lock
start_chromium

cat <<EOF
VNC is listening on localhost:${VNC_PORT} (server-side).
Password: ${VNC_PASS}
Chrome profile: ${PROFILE_DIR}
On your Mac, create a tunnel and open the viewer:

  ssh -L ${VNC_PORT}:localhost:${VNC_PORT} <user>@<server>
  open vnc://localhost:${VNC_PORT}

Log in to Alza in the browser window. After login, return here.
EOF

if [ -z "$CLI_BIN" ]; then
  if [ -x "$SCRIPT_DIR/../alza" ]; then
    CLI_BIN="$SCRIPT_DIR/../alza"
  elif command -v alza >/dev/null 2>&1; then
    CLI_BIN="alza"
  fi
fi

if [ "$REFRESH_MODE" = "cdp" ]; then
  ALZA_TOKEN_TIMEOUT_MS="$TOKEN_TIMEOUT_MS" ALZA_CDP_URL="http://127.0.0.1:${CDP_PORT}" \
    "$SCRIPT_DIR/legacy/refresh-token-cdp.sh"
  exit $?
fi

read -r -p "Press Enter to refresh token using Chrome cookies..." _

if [ -n "$CLI_BIN" ]; then
  "$CLI_BIN" token refresh --chrome-profile "$PROFILE_DIR"
else
  cat <<EOF
alza binary not found. Run manually:
  alza token refresh --chrome-profile "$PROFILE_DIR"
EOF
  exit 1
fi
