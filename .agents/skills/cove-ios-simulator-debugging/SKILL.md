---
name: cove-ios-simulator-debugging
description: Debug and visually validate Cove on an iOS Simulator, including Wails/Vite launch failures, bundle or Info.plist mismatches, ATS and server connectivity errors, authenticated chat flows, WKWebView keyboard movement, safe-area spacing, shadows, screenshots, and recordings. Use when requests mention Cove iOS, iPhone Simulator, “无法连接到服务器”, ATS -1022, an app that will not launch, keyboard-induced whole-screen movement, or iOS visual QA.
---

# Cove iOS Simulator Debugging

Use the repository's real Cove application and real authenticated state. Do not invent mock data or patch an installed `.app` as the final fix.

## Start Here

1. Run `scripts/ios_qa.sh preflight` from any directory.
2. Record `git status --short` before building. Preserve every pre-existing change.
3. Use `scripts/ios_qa.sh dev` as the default build/install/launch path. It delegates to `wails3 task ios:dev` and uses the generated Cove bundle.
4. Do not start with `wails3 task ios:run`. Its legacy `build/ios/Info.dev.plist` currently names `check` and `com.example.check.dev`, while the task tries `com.wails.cove.dev` and the executable is `cove`.
5. Run `scripts/ios_qa.sh inspect-bundle` after packaging when launch identity or ATS is suspect.
6. Use `scripts/ios_qa.sh logs` to classify native failures before changing application code.

Read [references/troubleshooting.md](references/troubleshooting.md) when diagnosing connectivity, launch identity, keyboard movement, safe areas, or visual regressions.

## Debugging Order

Follow this order and stop when the first failing layer is identified:

1. Verify Xcode tools, a booted iPhone Simulator, Vite, API configuration, and the working tree.
2. Build and launch through `ios:dev`.
3. Compare the packaged executable, `CFBundleExecutable`, `CFBundleIdentifier`, expected bundle ID, and files inside the bundle.
4. Separate Mac reachability, Simulator/WebKit ATS, authentication, CORS, and frontend rendering failures using logs and direct requests.
5. Request user interaction only when real state is required: ask the user to log in, open or close the keyboard, switch appearance, rotate, or reproduce an intermittent gesture. Continue immediately after confirmation.
6. Capture screenshots and a recording. For intermittent keyboard issues, record at least three open/close cycles and inspect frames across the full animation.

Do not claim that a server is unreachable until ATS and native logs have been ruled out. Treat HTTPS as the production solution. Limit any HTTP exception to a development Plist; never present an edit to a built bundle as persistent configuration.

## Keyboard and Safe-Area Invariants

For this Cove WKWebView chat layout:

- Track `window.visualViewport.height` on `resize`.
- Keep the application root fixed at `top: 0` and size it from the visual viewport height.
- Do not compensate with `visualViewport.offsetTop`.
- Do not subscribe to visual viewport `scroll` for layout correction.
- Move only the composer region when the keyboard changes the viewport.
- Reduce keyboard-open bottom padding and hide nonessential footer copy when it produces a second safe-area gap.
- Keep the header and status-bar relationship fixed throughout keyboard animation.

Validate behavior, not just the final frame: no intermittent whole-screen lift, no drawer shadow leaking from the closed left edge, no composer overlap, and no abrupt mismatch between repeated keyboard cycles.

## Evidence and Handoff

- Capture the resting screen with `scripts/ios_qa.sh screenshot [path]`.
- Record interactions with `scripts/ios_qa.sh record [path]`; stop with `Ctrl-C`.
- Report the exact Simulator model/runtime, bundle ID, API base URL, tested appearance, interaction sequence, and observed result.
- Re-run `git status --short` after builds because iOS generation can rewrite tracked Xcode files. Revert only confirmed build noise and never discard unrelated user changes.
- If a repository fix is requested, follow GitNexus impact-analysis rules before editing existing symbols and run change detection before committing.

