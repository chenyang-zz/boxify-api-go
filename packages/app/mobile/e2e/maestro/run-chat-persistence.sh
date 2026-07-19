#!/usr/bin/env bash
set -euo pipefail

required_variables=(
  IOS_SIMULATOR_UDID
  MAESTRO_EXPO_DEV_CLIENT_URL
  MAESTRO_E2E_API_URL
  MAESTRO_E2E_LLM_BASE_URL
  MAESTRO_E2E_USERNAME
  MAESTRO_E2E_PASSWORD
  MAESTRO_E2E_CHAT_PROMPT
  MAESTRO_E2E_CHAT_ANSWER
)

for variable_name in "${required_variables[@]}"; do
  if [[ -z "${!variable_name:-}" ]]; then
    echo "Missing required environment variable: ${variable_name}" >&2
    exit 2
  fi
done

if (( ${#MAESTRO_E2E_CHAT_PROMPT} > 20 )); then
  echo "MAESTRO_E2E_CHAT_PROMPT must be at most 20 characters so the generated conversation title is deterministic." >&2
  exit 2
fi

for command_name in maestro node; do
  if ! command -v "${command_name}" >/dev/null 2>&1; then
    echo "Required command not found: ${command_name}" >&2
    exit 2
  fi
done

mobile_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
workspace_root="$(cd "${mobile_dir}/../../.." && pwd)"
run_id="${E2E_RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)-chat-persistence}"

if [[ ! "${run_id}" =~ ^[A-Za-z0-9._-]+$ ]]; then
  echo "E2E_RUN_ID may contain only letters, numbers, dots, underscores, and hyphens." >&2
  exit 2
fi

artifact_dir="${workspace_root}/output/ios-simulator/runs/${run_id}"
mkdir -p "${artifact_dir}/evidence" "${artifact_dir}/maestro-debug"

sanitize_sensitive_maestro_artifacts() {
  node "${mobile_dir}/e2e/maestro/sanitize-artifacts.mjs" "${artifact_dir}"
}

trap sanitize_sensitive_maestro_artifacts EXIT

export MAESTRO_CLI_NO_ANALYTICS="${MAESTRO_CLI_NO_ANALYTICS:-true}"
export MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED="${MAESTRO_CLI_ANALYSIS_NOTIFICATION_DISABLED:-true}"

node "${mobile_dir}/e2e/maestro/setup-chat-fixture.mjs"

echo "Running Cove chat persistence E2E on Simulator ${IOS_SIMULATOR_UDID}"
echo "Artifacts: ${artifact_dir}"

maestro test \
  --udid "${IOS_SIMULATOR_UDID}" \
  --config "${mobile_dir}/e2e/maestro/config.yaml" \
  --test-output-dir "${artifact_dir}/evidence" \
  --debug-output "${artifact_dir}/maestro-debug" \
  --format JUNIT \
  --output "${artifact_dir}/maestro-junit.xml" \
  "${mobile_dir}/e2e/maestro/flows/chat-persistence.yaml"
