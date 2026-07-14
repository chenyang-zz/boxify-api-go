# Cove Native Navigation Architecture

## Source of Truth

Read these files together before changing the navigation protocol:

| Responsibility | Source |
| --- | --- |
| Typed Web → native actions and route detection | `frontend/src/app/nativeNavigation.ts` |
| Default authentication entry and Web fallback | `frontend/src/app/App.tsx` |
| Route-specific React entry selection | `frontend/src/main.tsx` |
| Authenticated chat entry and session handshake | `frontend/src/app/NativeChatApp.tsx` |
| Registration entry | `frontend/src/app/NativeRegisterApp.tsx` |
| Profile entry and navigation lock | `frontend/src/app/NativeProfileApp.tsx` |
| UIKit controller stack and WKScriptMessageHandler | `build/ios/cove_navigation_ios.m` |
| Generated Xcode integration | `build/ios/scripts/full_bleed_overlay.go` |

Do not edit the copied overlay in `build/ios/xcode/wails-full-bleed`. Regeneration replaces it.

## Ownership Boundary

UIKit owns:

- `UINavigationController` stack membership
- animated push and pop
- interactive back gesture
- controller/WebView allocation and release
- forwarding native lifecycle events to a route-specific page

React owns:

- page content and accessibility
- form state and validation
- authentication/session persistence
- sheet state and the signal that page navigation is temporarily locked
- a Web fallback when `window.webkit.messageHandlers.coveNavigation` is absent

The bridge connects the two owners. It must not become a duplicate router or a token store.

## Controller and Route Map

| Native controller role | `coveRoute` | React root | Lifetime |
| --- | --- | --- | --- |
| Authentication root | none | `App` / `AuthScreen` | Root while anonymous |
| Registration | `register` | `NativeRegisterApp` | Preload or create on demand; destroy after pop to login |
| Authenticated chat | `chat` | `NativeChatApp` | Becomes the only stack root after authentication |
| Profile | `profile` | `NativeProfileApp` | Push from chat; preserve chat underneath |

The current Objective-C name `coveChatController` refers to the initial authentication controller. Do not infer behavior from that legacy name; inspect stack membership.

Every route-specific `WKWebView` uses `WKWebsiteDataStore.defaultDataStore`. This is why the chat WebView can read the session written by the login or registration WebView.

## Bridge Protocol

Keep this table synchronized with `NativeNavigationAction` and `CoveNavigationMessageHandler`:

| Action | Direction | Meaning |
| --- | --- | --- |
| `prepareRegister` | Web → native | Preload the registration controller |
| `pushRegister` | Web → native | Push the fresh preloaded registration controller immediately |
| `registerReady` | Web → native | Registration React root has mounted |
| `popRegister` | Web → native | Pop registration to login |
| `prepareChat` | Web → native | Preload authenticated chat |
| `chatReady` | Web → native | Chat React root has mounted |
| `authCompleted` | Web → native | Login or registration persisted a valid session |
| `cove:native-chat-authenticated` | native → Web event | Tell preloaded chat to reread shared storage |
| `chatSessionReady` | Web → native | Chat confirmed a stored session and can be shown |
| `chatLogout` | Web → native | Reset to authentication and release chat |
| `pushProfile` | Web → native | Prepare and push profile from chat |
| `profileReady` | Web → native | Profile React root has mounted |
| `cove:native-profile-activate` | native → Web event | Refresh profile state before a push |
| `profileNavigationLock` | Web → native | Disable/enable interactive pop for sheets or saves |
| `popProfile` | Web → native | Pop profile to chat |
| `profileSessionChanged` | Web → native | Notify chat and auth roots to reread session state |
| `profileLogout` | Web → native | Clear authenticated navigation and return to login |

When adding an action, update the TypeScript union, its sender, the native message handler, the receiver, tests, and this table in the same change.

## Required Sequences

### Login or Registration to Chat

```text
authentication page writes localStorage session
  → Web posts authCompleted
  → native creates/preloads coveRoute=chat WebView
  → chat posts chatReady after mounting
  → native dispatches cove:native-chat-authenticated
  → chat rereads shared localStorage
  → chat posts chatSessionReady
  → UINavigationController pushes chat animated
  → didShow replaces the stack with [chat]
```

Do not push chat immediately after `authCompleted`. A newly created WebView can still be loading or may not yet see the state expected by its React root.

### Login to Registration and Back

```text
login root posts prepareRegister during idle setup
  → user selects Register
  → Web posts pushRegister
  → native pushes registration animated immediately
  → registration posts registerReady after mounting
  → UIKit Back gesture or popRegister returns to login
  → didShow releases the old registration controller and WKWebView
  → login preloads a new, empty registration controller
```

Do not gate the registration push on `registerReady`. Registration has no session dependency, so delaying UIKit until React mounts makes the tap appear unresponsive. Do not add a generic full-screen loading animation to compensate for a missing preload. Authenticated chat is different: it must still wait until shared session storage is confirmed.

Destroy registration only after UIKit confirms that the pop completed. An interactive pop can be cancelled; releasing at gesture start or when `popRegister` is requested would leave the visible transition without its controller. Re-entering registration must create a fresh controller so form values, errors, focus, and pending request state do not survive.

## Route-Aware Preloading

Keep at most one off-stack destination preloaded:

- While login is idle, preload registration only.
- After registration pops, destroy the old controller and immediately preload a new empty registration controller while login is visible.
- After a validated login or registration submission starts, release any off-stack registration preload and preload authenticated chat while the API request is in flight.
- When authentication fails, release the chat preload. If login is visible, create a fresh registration preload; if registration is visible, keep the active registration controller.
- After authentication succeeds, promote the prepared chat controller and remove authentication history.
- Preload profile only after authenticated chat becomes visible; never preload it from the anonymous root WebView.
- Release profile when logout returns the stack to authentication.
- Never release a controller that is still in the navigation stack. A visible registration page and the login controller beneath it are reachable state, not off-stack preloads.

This policy bounds speculative WebViews without sacrificing a fresh registration form or delaying chat initialization until after the network response.

### Chat to Profile and Back

```text
chat posts pushProfile
  → native creates/preloads profile
  → profile posts profileReady
  → native dispatches cove:native-profile-activate
  → native pushes profile animated
  → profile reports navigation lock while sheet/save is active
  → pop or interactive Back returns to the still-mounted chat controller
```

### Logout

```text
Web clears the local session
  → Web posts chatLogout or profileLogout
  → native sets stack to [authentication root]
  → native releases authenticated chat
  → authentication root rereads session state
```

The user must never be able to swipe back into authenticated content after logout.

## Performance and Lifecycle

Multiple `WKWebView` instances trade memory for native transitions and state preservation. Apply these rules:

- Keep chat alive while profile is visible so conversation state, draft, and scroll position survive Back.
- Preload registration and authenticated chat only when the expected latency justifies their memory cost.
- Avoid reloading a destination on every push; use explicit activation events to refresh mutable data.
- Release authenticated chat on logout and discard unreachable controllers after stack cleanup.
- Remove script message handlers and delegates during teardown to prevent retain cycles or callbacks into dead owners.
- Prefer at most the active controller, its reachable parent, and one justified preloaded destination.
- Back every independent WKWebView with a native root view using the same dynamic light/dark page color. Keep the WebView transparent until its React entry reports ready; this preserves an immediate UIKit push without exposing WebKit's default white first frame.

If memory becomes a measured problem, optimize controller lifecycle before replacing native navigation with CSS animation.

## Native Acceptance Evidence

Require all of the following:

1. Runtime logging or inspection confirms distinct source and destination `UIViewController` instances.
2. A Simulator recording contains intermediate right-to-left push or left-to-right pop frames.
3. The native interactive Back gesture works where allowed and is blocked while profile navigation is locked.
4. Authentication completion removes login/registration from Back history.
5. Logout removes authenticated chat/profile from history.
6. The exact installed build is current; rebuild/reinstall when cache is suspected.

A final screenshot proves layout only. It does not prove who performed the transition.

## Regression Checklist

- Login → chat uses UIKit push and cannot return to login with Back.
- Register → chat uses the same authenticated handshake.
- Login ↔ register uses native push/pop and preserves form semantics.
- Returning from register destroys its controller; entering again shows a fresh form.
- Chat → profile → chat preserves chat state.
- Profile sheet closes before page pop and locks interactive Back while necessary.
- Profile updates propagate to chat without rebuilding the entire navigation stack.
- Logout returns to login and cannot swipe back into protected content.
- Light/dark page colors remain continuous during intermediate transition frames.
- Status bar, Home Indicator, keyboard, and horizontal overscroll remain correct.
- Frontend tests/build, Go tests, iOS build, and Air Simulator recording all pass.
