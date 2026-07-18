#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage:
  simctl.sh list
  simctl.sh boot <simulator-udid>
  simctl.sh wait [simulator-udid|booted]
  simctl.sh list-apps [simulator-udid|booted]
  simctl.sh install <absolute-app-path> [simulator-udid|booted]
  simctl.sh launch <bundle-id> [simulator-udid|booted]
  simctl.sh terminate <bundle-id> [simulator-udid|booted]
  simctl.sh open-url <url> [simulator-udid|booted]
  simctl.sh appearance <light|dark> [simulator-udid|booted]
  simctl.sh screenshot <absolute-output.png> [simulator-udid|booted]
  simctl.sh record <absolute-output.mp4> [simulator-udid|booted]

Use an explicit UDID when more than one Simulator is booted.
The record command runs until interrupted.
EOF
}

require_xcrun() {
  if ! command -v xcrun >/dev/null 2>&1; then
    echo "xcrun is unavailable; install or select Xcode command-line tools." >&2
    exit 1
  fi
}

target_or_booted() {
  printf '%s' "${1:-booted}"
}

require_absolute_output() {
  local output_path="$1"
  case "$output_path" in
    /*) ;;
    *)
      echo "Output path must be absolute: $output_path" >&2
      exit 2
      ;;
  esac
  if [[ -e "$output_path" ]]; then
    echo "Refusing to overwrite existing evidence: $output_path" >&2
    exit 2
  fi
  mkdir -p "$(dirname "$output_path")"
}

require_xcrun
command_name="${1:-}"

case "$command_name" in
  list)
    xcrun simctl list devices booted
    xcrun simctl list devices available
    ;;
  boot)
    [[ $# -eq 2 ]] || { usage >&2; exit 2; }
    xcrun simctl boot "$2"
    xcrun simctl bootstatus "$2" -b
    ;;
  wait)
    [[ $# -le 2 ]] || { usage >&2; exit 2; }
    xcrun simctl bootstatus "$(target_or_booted "${2:-}")" -b
    ;;
  list-apps)
    [[ $# -le 2 ]] || { usage >&2; exit 2; }
    xcrun simctl listapps "$(target_or_booted "${2:-}")"
    ;;
  install)
    [[ $# -ge 2 && $# -le 3 ]] || { usage >&2; exit 2; }
    [[ "$2" = /* && -d "$2" ]] || { echo "App path must be an existing absolute .app directory." >&2; exit 2; }
    xcrun simctl install "$(target_or_booted "${3:-}")" "$2"
    ;;
  launch)
    [[ $# -ge 2 && $# -le 3 ]] || { usage >&2; exit 2; }
    xcrun simctl launch "$(target_or_booted "${3:-}")" "$2"
    ;;
  terminate)
    [[ $# -ge 2 && $# -le 3 ]] || { usage >&2; exit 2; }
    xcrun simctl terminate "$(target_or_booted "${3:-}")" "$2"
    ;;
  open-url)
    [[ $# -ge 2 && $# -le 3 ]] || { usage >&2; exit 2; }
    xcrun simctl openurl "$(target_or_booted "${3:-}")" "$2"
    ;;
  appearance)
    [[ $# -ge 2 && $# -le 3 ]] || { usage >&2; exit 2; }
    [[ "$2" == "light" || "$2" == "dark" ]] || { echo "Appearance must be light or dark." >&2; exit 2; }
    xcrun simctl ui "$(target_or_booted "${3:-}")" appearance "$2"
    ;;
  screenshot)
    [[ $# -ge 2 && $# -le 3 ]] || { usage >&2; exit 2; }
    require_absolute_output "$2"
    xcrun simctl io "$(target_or_booted "${3:-}")" screenshot "$2"
    ;;
  record)
    [[ $# -ge 2 && $# -le 3 ]] || { usage >&2; exit 2; }
    require_absolute_output "$2"
    xcrun simctl io "$(target_or_booted "${3:-}")" recordVideo "$2"
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac
