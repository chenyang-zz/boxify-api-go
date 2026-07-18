# Launch and Diagnosis Reference

## Contents

- Project discovery
- Simulator preparation
- Native Xcode projects
- Expo and React Native projects
- Installation and runtime inspection
- Networking and ATS
- Evidence capture
- Failure classification

## Project discovery

Inspect the repository before choosing commands:

```sh
rg --files -g '*.xcworkspace' -g '*.xcodeproj' -g 'Package.swift' \
  -g 'app.json' -g 'app.config.*' -g 'package.json' -g 'Podfile' \
  -g 'Taskfile.yml' -g 'Makefile'
```

Then inspect documented scripts, schemes, and environment variable names. Prefer a repository QA script over reconstructing an equivalent command. Do not read secret-bearing `.env` files unless their contents are necessary and the task authorizes it; variable names are usually discoverable from source and example files.

Determine these values before launch:

- workspace or project path
- scheme or package script
- configuration (`Debug` unless the task requires another)
- bundle ID
- app URL scheme or universal link
- bundler port for JavaScript apps
- API base URL variable and development transport policy

## Simulator preparation

List devices and runtimes:

```sh
xcrun simctl list devices booted
xcrun simctl list devices available
xcrun simctl list runtimes
```

Use an explicit UDID for multi-step work:

```sh
xcrun simctl boot "$SIMULATOR_ID"
xcrun simctl bootstatus "$SIMULATOR_ID" -b
open -a Simulator
```

Booting an already-booted device can return a non-zero result. Check current state rather than treating that result as an app failure.

## Native Xcode projects

Discover schemes before building:

```sh
xcodebuild -list -workspace /absolute/path/App.xcworkspace
xcodebuild -showdestinations -workspace /absolute/path/App.xcworkspace -scheme App
```

Build into a task-specific DerivedData directory so the product is easy to inspect:

```sh
task_derived_data="$(mktemp -d)"
xcodebuild build \
  -workspace /absolute/path/App.xcworkspace \
  -scheme App \
  -configuration Debug \
  -destination "platform=iOS Simulator,id=$SIMULATOR_ID" \
  -derivedDataPath "$task_derived_data"
```

If the repository uses an `.xcodeproj`, replace `-workspace` with `-project`. Do not guess a scheme from the filename; read `xcodebuild -list` or project instructions.

Locate and install the built app:

```sh
find "$task_derived_data/Build/Products" -maxdepth 3 -type d -name '*.app' -print
xcrun simctl install "$SIMULATOR_ID" /absolute/path/App.app
xcrun simctl launch "$SIMULATOR_ID" com.example.app
```

## Expo and React Native projects

Inspect `package.json`, Expo config, native project files, and repository instructions first. Prefer checked-in package-manager scripts.

Typical Expo development-client flow:

```sh
pnpm exec expo run:ios --device "$SIMULATOR_ID"
pnpm exec expo start --dev-client
```

Typical bare React Native flow uses the repository's `ios` script or a direct Xcode build. Run CocoaPods installation only when dependency state requires it; do not mutate Pods merely as a first diagnostic step.

Keep these layers distinct:

- native app installed and process launched
- development client connected to Metro
- JavaScript bundle loaded
- application API reachable
- authenticated state restored

For deep links, use the scheme and path documented by the app:

```sh
xcrun simctl openurl "$SIMULATOR_ID" 'example-app://known/path'
```

Do not invent an Expo URL or route. Confirm the development server URL, scheme, and router path from current output/configuration.

## Installation and runtime inspection

Inspect installed application metadata:

```sh
xcrun simctl listapps "$SIMULATOR_ID"
xcrun simctl get_app_container "$SIMULATOR_ID" com.example.app app
xcrun simctl spawn "$SIMULATOR_ID" log stream \
  --style compact \
  --predicate 'process == "AppProcessName"'
```

Use a bounded log capture when possible. Sanitize tokens, credentials, personal data, and request bodies before reporting logs.

Inspect a built plist rather than assuming the source plist is the installed value:

```sh
plutil -p /absolute/path/App.app/Info.plist
```

## Networking and ATS

Check in this order:

1. resolved API base URL in the app build/runtime
2. host service listening on the intended interface and port
3. Simulator route to that host
4. ATS policy in the installed bundle
5. request authentication and response shape
6. application error mapping and UI state

`localhost` inside iOS Simulator normally reaches the Mac host, but proxies, VPNs, IPv6 resolution, container binding, and app-specific networking can still differ. A successful host request proves only host reachability.

For HTTP development endpoints, verify whether the app intentionally enables a scoped ATS exception. Do not add a production-wide arbitrary-load exception as a diagnostic shortcut.

## Evidence capture

Use explicit, new output paths:

```sh
xcrun simctl io "$SIMULATOR_ID" screenshot /absolute/path/baseline.png
xcrun simctl io "$SIMULATOR_ID" recordVideo /absolute/path/flow.mp4
```

Stop `recordVideo` with an interrupt after the flow. Include intermediate frames and avoid overwriting prior evidence.

## Failure classification

| Symptom | First checks |
| --- | --- |
| No booted device | runtime availability, explicit UDID, `bootstatus` |
| Build failure | first compiler/linker error, scheme, destination, dependencies |
| Install failure | app product path, architecture, bundle validity, Simulator state |
| Immediate crash | process logs, crash report, installed plist and entitlements |
| Development-client home | Metro URL and explicit dev-client connection |
| Blank JavaScript screen | bundler output, bundle load, runtime exception |
| API unavailable | resolved URL, service binding, Simulator networking, ATS |
| Repeated login | secure storage/local session, refresh rotation, clock, 401 handling |
| Tap misses | current screenshot, accessibility state, keyboard/overlay obstruction |
| White navigation flash | root/navigation/screen background across intermediate frames |
| Keyboard jump | safe-area and keyboard primitives, focused control, recording evidence |
