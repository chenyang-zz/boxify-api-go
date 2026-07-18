---
name: ios-simulator
description: Build, launch, inspect, automate, debug, and visually validate iOS apps in Apple Simulator with xcodebuild, simctl, and Codex Computer Use as the default stack. Use when a task mentions iOS Simulator, simctl, Xcode Simulator, SwiftUI/UIKit, Expo or React Native on iOS, deep links, Simulator networking or ATS, native navigation, screenshots, screen recordings, or end-to-end iOS UI flows. Use WebDriverAgent or Midscene only for explicitly requested or specialized visual automation.
---

# iOS Simulator

Use a real Simulator build and collect reproducible evidence. Diagnose the first failing layer before changing code. Do not treat a successful build, a host-side network request, or a resting screenshot as proof that the requested user flow works.

## Default Stack

Use this combination unless the task requires a different backend:

1. Prefer repository-provided launch and QA scripts when they are current and documented.
2. Use `xcodebuild`, `xcrun simctl`, and [scripts/simctl.sh](scripts/simctl.sh) for deterministic device, install, launch, deep-link, appearance, screenshot, and recording operations.
3. Use Codex Computer Use for normal foreground UI inspection and interaction in the Simulator.
4. Capture screenshots, recordings, sanitized logs, and test results as evidence.

WDA + Midscene is an optional escalation path, not the default. Use it only when the user requests it, Computer Use cannot reliably reach the required UI, or a long repeated visual flow materially justifies the setup. Read [references/wda-and-midscene.md](references/wda-and-midscene.md) before using it.

## Choose the Control Path

| Need | Preferred path |
| --- | --- |
| Build, install, launch, deep link, appearance, screenshot, or recording | repository scripts, `xcodebuild`, and `simctl` |
| One-off native UI flow, visual diagnosis, or interactive validation | Computer Use by default |
| Incomplete accessibility tree | Computer Use with screenshots and coordinate actions |
| Long or repeated visual flow, or an explicit WDA/Midscene request | WDA + Midscene |
| Deterministic regression or headless CI | the project's XCUITest, Maestro, Detox, or equivalent test framework |

Use the narrowest path that can prove the requirement. Prefer an existing project-native test framework when the requested test must be reproducible in CI; do not treat Computer Use or Midscene as a substitute for deterministic regression coverage.

### Operate the Simulator with Computer Use

- Load the installed Computer Use skill completely before controlling the Simulator, and follow its confirmation policy.
- Target Simulator by app name or bundle identifier. Read the current UI state before the first action and again after every action or transition.
- Prefer accessibility `element_index` actions. Never reuse an index from stale UI state.
- Fall back to screenshots, coordinate clicks, and key presses only when accessibility data is incomplete or unreliable.
- Use Computer Use only for foreground GUI work. Do not depend on it for headless CI.

Never claim to have used a tool that is not exposed in the current session. If an optional automation backend is unavailable, continue with `simctl`, accessibility-aware Computer Use, and manual evidence where possible.

## Core Workflow

### 1. Establish the baseline

- Read applicable repository instructions before acting.
- Record `git status --short` and preserve every pre-existing change.
- Identify the app technology, canonical build command, bundle ID, URL scheme, API configuration, and expected runtime.
- List booted and available devices. Use an explicit Simulator UDID once selected; avoid relying on `booted` when multiple devices are running.
- Confirm whether the request is diagnosis only or authorizes a code change.

Read [references/launch-and-diagnose.md](references/launch-and-diagnose.md) for project-specific discovery and launch commands.

### 2. Prepare one known target

- Reuse the intended booted Simulator when it matches the task.
- Otherwise boot one explicit device and wait for `bootstatus -b` to complete.
- Record device name, UDID, iOS runtime, appearance, and orientation.
- Avoid erasing a Simulator or resetting content unless the user explicitly requests it; those actions destroy local test state.

### 3. Build, install, and launch

- Prefer the project's canonical scheme, workspace, dev-client, or task script.
- Keep build output available for diagnosis; do not pipe it through `tail` before the failure is understood.
- Verify the installed bundle ID instead of inferring it from a source config.
- Launch the app explicitly and capture the first visible state.
- For JavaScript-based apps, distinguish the installed native shell from the bundler connection. A development-client home screen is not the application UI.

### 4. Diagnose by layer

Stop at the first failing layer and collect evidence before moving deeper:

1. Xcode command-line tools and Simulator runtime
2. native compile, link, signing, and bundle creation
3. installation and process launch
4. Metro/Vite/dev-client connection when applicable
5. Simulator-to-service networking and ATS
6. authentication and persisted session
7. application render and state management
8. accessibility, gestures, keyboard, safe areas, sheets, and navigation

Do not patch application code to compensate for a stale bundle, disconnected bundler, wrong bundle ID, wrong API base URL, or unavailable backend.

### 5. Exercise the user flow

- Start from a known route and state. Prefer a documented deep link when it preserves the behavior under test.
- Capture a baseline screenshot before interaction.
- Use specific, unique accessibility or visual descriptions such as “the Save button in the bottom toolbar,” not “button.”
- Reacquire visible state after every navigation, animation, keyboard transition, drawer, alert, sheet, or appearance change.
- Wait only as long as the UI transition or network request requires, then inspect state again; do not use long blind sleeps.
- Verify intermediate states, not only the final screen.
- Record push/pop, interactive Back, keyboard, rotation, and animation behavior. A still image cannot prove motion or lifecycle semantics.

### 6. Protect user data

- Do not invent credentials, read secret environment files unnecessarily, or print tokens and API keys.
- Do not send a real chat message, place an order, publish content, delete data, or perform another consequential mutation solely to create test state.
- Prefer an existing test account and disposable fixtures supplied by the user or repository.
- Inspect destructive dialogs by cancelling them unless completing the mutation is explicitly part of the requested test.
- If a flow creates disposable data, clean it up only when cleanup is safe and authorized.

### 7. Validate and hand off

Capture enough evidence to reproduce the result:

- project/runtime type and command used
- Simulator model, UDID, and iOS version
- bundle ID, app version/build when available, and bundler URL when applicable
- API base URL without credentials
- exact route, initial state, appearance, orientation, and interaction sequence
- screenshot and recording paths
- build, typecheck, unit/UI test results
- first failing layer and remaining environment-only exceptions

Run `git status --short` again after native builds. Generated Xcode, Pods, and Expo files may change. Revert only files proven to be build noise and never discard pre-existing user work.

## Evidence Standard

- Pixel correctness: screenshot at the relevant state.
- Animation or navigation: screen recording containing intermediate frames.
- Lifecycle: demonstrate preserved or reset state after push/pop, background/foreground, or authentication transition.
- Network/authentication: app-visible result plus relevant sanitized logs; a host-side `curl` alone is insufficient.
- Fix verification: reproduce the failure before the change when practical, then repeat the same sequence after the change.
