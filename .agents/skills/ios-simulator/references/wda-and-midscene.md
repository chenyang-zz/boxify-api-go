# WebDriverAgent and Midscene iOS

This is an optional escalation path, not the default Codex workflow. For ordinary Simulator work, use `xcodebuild` or `simctl` for lifecycle control and Computer Use for foreground UI interaction.

## Contents

- When to use this path
- Prerequisites
- Start WebDriverAgent
- Connect Midscene
- Execute a flow
- Locator guidance
- Troubleshooting
- Codex configuration guidance

## When to use this path

Use WDA + Midscene only when at least one of these conditions applies:

- the user explicitly requests WDA or Midscene
- Computer Use cannot expose or operate the required controls reliably
- an already-configured WDA/Midscene setup is the project's established workflow
- a long or repeated visual flow benefits enough to justify the additional setup

Do not install or configure this stack for one-off inspection, a simple interaction, or a single screenshot. Do not use it as the sole proof for deterministic CI regression; prefer the project's XCUITest, Maestro, Detox, or equivalent test framework for that purpose.

## Prerequisites

Confirm all of the following:

- one explicit iOS Simulator UDID
- an existing local WebDriverAgent checkout
- a working Xcode toolchain
- port 8100 available
- Midscene iOS MCP exposed to the current Codex session
- model provider credentials supplied through secure user/global configuration

Never commit model API keys or place them in a project skill. Do not echo them in logs or final output.

## Start WebDriverAgent

Set task-specific variables; do not hard-code a user's Desktop path:

```sh
WDA_PROJECT_DIR=/absolute/path/to/WebDriverAgent
SIMULATOR_ID=00000000-0000-0000-0000-000000000000
cd "$WDA_PROJECT_DIR"
xcodebuild test \
  -project WebDriverAgent.xcodeproj \
  -scheme WebDriverAgentRunner \
  -destination "platform=iOS Simulator,id=$SIMULATOR_ID" \
  USE_PORT=8100
```

Keep the command in a long-lived terminal/exec session so its complete output remains available. Do not hide the first failure with `tail`. In another command session, poll readiness with short bounded checks:

```sh
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  if curl --fail --silent http://127.0.0.1:8100/status >/dev/null; then
    echo 'WebDriverAgent is ready'
    exit 0
  fi
  sleep 2
done
echo 'WebDriverAgent did not become ready within 20 seconds' >&2
exit 1
```

If readiness fails, inspect the complete Xcode output, destination UDID, scheme, signing/build errors, port conflicts, and Simulator state.

## Connect Midscene

Use the exact Midscene iOS tools exposed in the current session. Tool names and schemas can vary by version; inspect them rather than inventing a call. Connect to the selected Simulator/WDA session before issuing UI operations, and reconnect after either the MCP server or WDA restarts.

If no Midscene iOS tool is exposed, do not pretend it is connected. Continue with Computer Use or report that optional MCP configuration is required.

## Execute a flow

Structure each flow as evidence-producing steps:

1. Navigate to a documented route or establish the starting UI state.
2. Capture a baseline screenshot.
3. Perform one semantic action: tap, input, clear, swipe, scroll, long press, or keyboard action.
4. Capture or inspect the resulting state.
5. Assert the visible result and any required state/lifecycle behavior.
6. Repeat with state reacquisition after every transition.
7. Record the full sequence when motion, navigation, or keyboard behavior matters.

Prefer deep links for setup only when they are part of the supported app contract. A deep link must not bypass the behavior being tested.

## Locator guidance

Describe a unique target using stable features:

- visible text or accessibility label
- control role
- container or section
- relative position only when necessary
- color only as a secondary clue

Good: “the Save button in the bottom profile toolbar.”

Weak: “the orange button.”

After a failed or ambiguous action, take a screenshot and inspect the current page before retrying. Do not repeat blind taps at the same coordinates.

## Troubleshooting

| Problem | Response |
| --- | --- |
| WDA status unavailable | inspect Xcode session, UDID, port 8100, and Simulator state |
| MCP connection lost | verify WDA status, restart/reconnect the MCP session |
| Semantic tap misses | reacquire screenshot, add text/role/container detail |
| No visible change | check animation/request state, overlays, keyboard, and app logs |
| Input goes to wrong field | clear focus/keyboard state and locate by unique label |
| Deep link opens wrong screen | verify bundle scheme, router path, and installed build |
| Tool response is slow | wait for the current call; do not issue overlapping actions |

## Codex configuration guidance

Keep Midscene as an optional MCP dependency rather than embedding it in the skill. Configure it through the user's Codex MCP settings or a trusted project `.codex/config.toml`. Use `codex mcp add --help` or the current Codex settings UI to confirm the supported syntax for the installed version.

The server command is typically based on:

```text
npx -y @midscene/ios-mcp
```

The provider expects model name, API key, base URL, and model-family environment variables. Store secrets in user-managed environment/configuration, not in the repository. Restart Codex after changing MCP configuration, then verify that the iOS connection and interaction tools are actually exposed before using this workflow.
