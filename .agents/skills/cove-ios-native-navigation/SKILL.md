---
name: cove-ios-native-navigation
description: Design, implement, refactor, and debug Cove page transitions with UIKit UINavigationController, independent UIViewControllers, and route-specific WKWebViews. Use when work involves native iOS page pushes or pops, login/register/chat/profile navigation, right-to-left transitions, interactive back gestures, authentication stack cleanup, Web-to-native navigation bridge messages, or a transition that looks like a React/CSS swap instead of a native iOS transition.
---

# Cove iOS Native Navigation

Implement navigation as an iOS controller stack while keeping React responsible for each page's content and API state. Treat a transition as native only when UIKit changes the visible `UIViewController`.

## Establish the Exact Navigation Edge

Name the edge before changing code: login → register, register → login, login/register → chat, chat → profile, profile → chat, or logout → login. Do not validate one edge and claim another is fixed.

Read [references/architecture.md](references/architecture.md) before changing the bridge, controller lifecycle, authentication transition, or native build overlay.

## Follow the Required Workflow

1. Read the repository `AGENTS.md` and obey its GitNexus rules. Run upstream impact analysis before editing every existing symbol. Warn before proceeding when risk is HIGH or CRITICAL.
2. Inspect the current route entry, TypeScript action union, Objective-C handler, controller stack mutation, and generated-project overlay. Keep the bridge contract synchronized across TypeScript and Objective-C.
3. Preserve the non-iOS fallback. When the native bridge is absent, keep the React navigation path functional.
4. Give every native destination an independent `UIViewController` and `WKWebView`. Use `UINavigationController.pushViewController(_:animated:)` and pop APIs for the visible transition.
5. Share `WKWebsiteDataStore.defaultDataStore` so route-specific WebViews can read the same `localStorage` session. Do not copy tokens through ad-hoc bridge payloads.
6. Preload exactly one route-relevant destination when appropriate. Push destinations without protected state, such as registration, immediately. Do not mask a cold route with a generic full-screen spinner; fix its preload lifecycle. For authenticated chat, wait for the session handshake rather than pushing before required state is readable.
7. Give every route-specific controller a native root surface using the same dynamic page color as the web app. Keep an unready WKWebView transparent until its React root reports readiness so WebKit's default white frame never appears during an immediate push.
8. Remove authentication controllers from history after authenticated chat becomes visible. On logout, clear the session, return to the authentication root, and release the authenticated chat controller.
9. Let UIKit own the back gesture. Disable interactive pop while a profile sheet is open or a save is in progress, then restore it when navigation is safe.
10. Edit source-of-truth files only. Do not patch generated copies under `build/ios/xcode/wails-full-bleed`; update `build/ios/cove_navigation_ios.m` or the overlay installer that copies it.
11. Run focused tests, the frontend production build, Go tests, and iOS build. Use the Air iOS Simulator by default and verify the exact transition with a screen recording.

## Preserve These Invariants

- A React state change, CSS transform, history change, or route query change alone is not native navigation.
- Each native page owns one React entry selected by `coveRoute`; the native controller owns the page stack.
- Bridge messages are commands or readiness signals, not a second source of application state.
- The authenticated chat page must confirm that shared session storage is readable before UIKit pushes it.
- Login and registration must not remain reachable through Back after authentication succeeds.
- Chat state should survive a chat → profile → chat round trip.
- Profile Back closes an open sheet first; page-level pop remains locked until the sheet or save operation is complete.
- Hidden WebViews must have a lifecycle reason. Release pages that are no longer reachable instead of accumulating one WebView per visit.
- A cold destination may reveal its native page-color surface briefly, but it must never reveal WebKit's default white background or a generic loading overlay.

## Reject False Positives

Do not accept any of the following as proof of native navigation:

- Objective-C symbols exist in the binary.
- The destination page eventually appears.
- A final screenshot looks correct.
- A React test confirms a route changed.
- A CSS animation resembles the iOS push curve.

Require runtime evidence showing intermediate UIKit transition frames, the expected source and destination controllers, and correct Back behavior.

## Validate on iOS

Use the project `cove-ios-simulator-debugging` skill for build, launch, logs, screenshots, and recordings.

1. Run its preflight and record the dirty worktree.
2. Build/install/launch with its `ios_qa.sh dev` workflow.
3. Record the exact user sequence, including the tap before the transition and the Back gesture after it.
4. Inspect intermediate recording frames for the UIKit push/pop parallax, not only the resting destination.
5. Verify light/dark backgrounds, safe areas, keyboard behavior, and horizontal locking on the same build.
6. Confirm stack cleanup after authentication and logout, then check for generated Xcode noise.

Never claim completion from cached browser content or a stale installed app. Confirm the installed bundle was rebuilt from the current source when the visual result appears unchanged.
