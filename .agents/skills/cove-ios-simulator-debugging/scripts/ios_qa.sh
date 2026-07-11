#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../../.." && pwd)"
DEFAULT_BUNDLE="${REPO_ROOT}/bin/cove.ios-dev.app"
IOS_BUNDLE_ID="${IOS_BUNDLE_ID:-io.github.chenyangzz.cove}"
VITE_PORT="${WAILS_VITE_PORT:-9245}"
VITE_URL="http://127.0.0.1:${VITE_PORT}"

usage() {
  cat <<'EOF'
Usage: ios_qa.sh <command> [argument]

Commands:
  preflight                 Check the Cove iOS debugging environment.
  dev                       Run the canonical `wails3 task ios:dev` flow.
  inspect-bundle [app]      Inspect a bundle (default: bin/cove.ios-dev.app).
  logs                      Stream Simulator logs filtered to Cove.
  screenshot [path]         Save a Simulator PNG.
  record [path]             Record Simulator video until Ctrl-C.
  help                      Show this help.
EOF
}

fail() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

require_booted_simulator() {
  local devices
  devices="$(xcrun simctl list devices booted 2>/dev/null)" || fail "cannot query CoreSimulatorService"
  grep -q '(Booted)' <<<"${devices}" || fail "no booted iOS Simulator"
}

print_env_value() {
  local key="$1"
  local file value
  for file in \
    "${REPO_ROOT}/frontend/.env.development.local" \
    "${REPO_ROOT}/frontend/.env.local" \
    "${REPO_ROOT}/frontend/.env.development"; do
    if [[ -f "${file}" ]]; then
      value="$(sed -n "s/^${key}=//p" "${file}" | tail -n 1)"
      if [[ -n "${value}" ]]; then
        printf '%s=%s (%s)\n' "${key}" "${value}" "${file#${REPO_ROOT}/}"
        return 0
      fi
    fi
  done
  printf '%s=<unset; application fallback applies>\n' "${key}"
}

preflight() {
  local status=0

  printf 'Repository: %s\n' "${REPO_ROOT}"
  printf 'Expected bundle ID: %s\n' "${IOS_BUNDLE_ID}"
  printf 'Vite URL: %s\n' "${VITE_URL}"
  print_env_value VITE_API_BASE_URL

  for command in xcode-select xcrun plutil wails3 pnpm curl; do
    if command -v "${command}" >/dev/null 2>&1; then
      printf 'ok: %s -> %s\n' "${command}" "$(command -v "${command}")"
    else
      printf 'missing: %s\n' "${command}" >&2
      status=1
    fi
  done

  if xcode-select -p >/dev/null 2>&1; then
    printf 'Xcode developer directory: %s\n' "$(xcode-select -p)"
  else
    printf 'error: Xcode developer directory is unavailable\n' >&2
    status=1
  fi

  if xcrun simctl list devices booted 2>/dev/null | grep -q '(Booted)'; then
    xcrun simctl list devices booted | grep '(Booted)'
  else
    printf 'error: no booted Simulator or CoreSimulatorService is unavailable\n' >&2
    status=1
  fi

  if curl --silent --fail --max-time 1 "${VITE_URL}/" >/dev/null 2>&1; then
    printf 'ok: Vite is reachable at %s\n' "${VITE_URL}"
  else
    printf 'info: Vite is not currently reachable; the dev command will start it\n'
  fi

  printf '%s\n' 'Working tree (preserve pre-existing changes):'
  git -C "${REPO_ROOT}" status --short
  return "${status}"
}

run_dev() {
  require_command wails3
  cd "${REPO_ROOT}"
  exec wails3 task ios:dev
}

inspect_bundle() {
  local bundle="${1:-${DEFAULT_BUNDLE}}"
  local plist executable identifier executable_path
  local ats_arbitrary ats_web_content ats_local

  [[ "${bundle}" = /* ]] || bundle="${REPO_ROOT}/${bundle}"
  [[ -d "${bundle}" ]] || fail "bundle does not exist: ${bundle}"
  plist="${bundle}/Info.plist"
  [[ -f "${plist}" ]] || fail "Info.plist does not exist: ${plist}"

  executable="$(plutil -extract CFBundleExecutable raw -o - "${plist}" 2>/dev/null || true)"
  identifier="$(plutil -extract CFBundleIdentifier raw -o - "${plist}" 2>/dev/null || true)"
  ats_arbitrary="$(plutil -extract NSAppTransportSecurity.NSAllowsArbitraryLoads raw -o - "${plist}" 2>/dev/null || true)"
  ats_web_content="$(plutil -extract NSAppTransportSecurity.NSAllowsArbitraryLoadsInWebContent raw -o - "${plist}" 2>/dev/null || true)"
  ats_local="$(plutil -extract NSAppTransportSecurity.NSAllowsLocalNetworking raw -o - "${plist}" 2>/dev/null || true)"
  executable_path="${bundle}/${executable}"

  printf 'Bundle: %s\n' "${bundle}"
  printf 'CFBundleExecutable: %s\n' "${executable:-<missing>}"
  printf 'CFBundleIdentifier: %s\n' "${identifier:-<missing>}"
  printf 'Expected identifier: %s\n' "${IOS_BUNDLE_ID}"
  printf 'Executable file: %s\n' "${executable_path}"
  [[ -n "${executable}" && -f "${executable_path}" ]] || fail "CFBundleExecutable does not match a file in the bundle"

  if [[ "${identifier}" != "${IOS_BUNDLE_ID}" ]]; then
    printf 'warning: bundle identifier differs from the ios:dev launch target\n' >&2
  fi

  printf 'ATS arbitrary loads: %s\n' "${ats_arbitrary:-<unset>}"
  printf 'ATS WebKit arbitrary loads: %s\n' "${ats_web_content:-<unset>}"
  printf 'ATS local networking: %s\n' "${ats_local:-<unset>}"
  codesign --verify --verbose=2 "${bundle}" 2>&1 || fail "bundle code signature is invalid"
}

stream_logs() {
  require_command wails3
  require_booted_simulator
  cd "${REPO_ROOT}"
  exec wails3 task ios:logs
}

take_screenshot() {
  local output="${1:-/private/tmp/cove-ios-$(date +%Y%m%d-%H%M%S).png}"
  require_booted_simulator
  mkdir -p "$(dirname "${output}")"
  xcrun simctl io booted screenshot "${output}"
  printf 'Screenshot: %s\n' "${output}"
}

record_video() {
  local output="${1:-/private/tmp/cove-ios-$(date +%Y%m%d-%H%M%S).mp4}"
  require_booted_simulator
  mkdir -p "$(dirname "${output}")"
  printf 'Recording to %s; press Ctrl-C to stop.\n' "${output}"
  exec xcrun simctl io booted recordVideo --codec=h264 "${output}"
}

case "${1:-help}" in
  preflight)
    preflight
    ;;
  dev)
    run_dev
    ;;
  inspect-bundle)
    inspect_bundle "${2:-}"
    ;;
  logs)
    stream_logs
    ;;
  screenshot)
    take_screenshot "${2:-}"
    ;;
  record)
    record_video "${2:-}"
    ;;
  help|-h|--help)
    usage
    ;;
  *)
    usage >&2
    fail "unknown command: $1"
    ;;
esac
