# Cove Workspace Architecture

This file describes the architecture that is currently present in the Cove monorepo. Read it before changing either package, then read the closer `AGENTS.md` and area-specific rule file.

## 1. Workspace and Repository Boundaries

`cove/` is the single Git repository for the Cove project. It contains two primary packages:

```text
cove/
├── Makefile       # Monorepo command entry point; delegates into each package
└── packages/
    ├── app/       # User-facing clients: Wails desktop, React/Vite UI, Expo mobile
    └── server/    # Go API, workers, scheduler, migrations, code generation
```

- Run Git and shared Make targets from `cove/`; each recipe changes into the owning package before invoking its native tool.
- Run direct build, test, and generation commands from the package they affect.
- Do not create nested Git repositories under `packages/`.
- A cross-package feature is complete only after both sides have been validated independently and the relevant E2E boundary has been exercised.
- `packages/server/docs/openapi.json` is the machine-readable contract shared by the packages.

## 2. System Topology

```text
Wails desktop ─┐
React/Vite UI ─┼── HTTP JSON + JWT + SSE ──> Go API (:8000)
Expo mobile ───┘                               │
                                              ├── PostgreSQL: users, config, conversations, metadata
                                              ├── Redis: realtime events and async queues
                                              ├── Elasticsearch: RAG chunks and hybrid retrieval
                                              ├── Neo4j: optional memory graph
                                              ├── Object storage: documents, images, signed URLs
                                              └── LLM and MCP providers
```

The clients do not access databases or external model providers directly. They communicate through the server API. The only local Go service currently exposed to the Wails UI is application metadata through generated Wails bindings.

## 3. App Package

### 3.1 Runtime surfaces

The app package contains three related but distinct surfaces:

| Surface | Entry points | Ownership |
| --- | --- | --- |
| Wails desktop shell | `main.go`, `internal/app/app.go` | Embeds `frontend/dist`, creates the desktop webview, exposes Go services |
| React/Vite UI | `frontend/src/main.tsx`, `frontend/src/app/App.tsx` | Desktop/web authentication, chat, profile, browser navigation fallback |
| Expo React Native | `mobile/src/app/_layout.tsx`, `mobile/src/app/index.tsx` | Current native mobile product and native navigation |

The active mobile architecture is Expo Router under `packages/app/mobile/`. The Wails + UIKit + WKWebView implementation under `packages/app/build/ios/` and the `Native*App` React entries is legacy and should be changed only when a task explicitly targets it.

### 3.2 Wails desktop layering

```text
main.go
  → embeds frontend/dist
  → internal/app.New
  → Wails application/window
  → internal/services
  → frontend/bindings (generated TypeScript bridge)
```

- `internal/app/` owns Wails configuration and window creation.
- `internal/services/` owns Go services bound into the webview.
- `internal/domain/` contains Wails-independent data types.
- `internal/platform/` contains operating-system-specific helpers.
- `frontend/bindings/` is generated bridge code; update its Go source/generation path rather than treating it as handwritten application logic.

### 3.3 React/Vite layering

```text
frontend/src/main.tsx
  → app/                 application shell, routing, native bridge entries
  → features/auth/       login, registration, token refresh, session restore
  → features/chat/       conversations, messages, attachments, SSE rendering
  → features/profile/    profile and password management
  → shared/api/          Wails-bound and reusable API wrappers
```

- The web/Wails session is stored in `localStorage` under `cove.auth.session.v1`.
- `features/auth/api.ts` owns JSON envelope parsing, bearer tokens, refresh-token rotation, and one retry after HTTP 401.
- `features/chat/api.ts` owns conversation requests and incremental SSE parsing for `/api/chat/stream`.
- `VITE_API_BASE_URL` is the only supported web API base URL override; the development fallback is `http://localhost:8000`.
- Keep native bridge actions synchronized across `frontend/src/app/nativeNavigation.ts`, the legacy UIKit handler, entry components, and their tests.

### 3.4 Expo mobile layering

```text
mobile/src/app/_layout.tsx
  → Stack.Protected authentication boundary
  → app/(auth)/          login and registration routes
  → app/(app)/           chat, knowledge, and profile routes
  → providers/           session-level state
  → core/                API, secure session, SSE, chat/message transforms
  → components/ + theme/ reusable presentation and visual tokens
```

- Expo Router/native stack owns page membership, push/pop behavior, headers, gestures, and protected routes.
- `AuthProvider` owns authenticated session state, not page-local navigation or form state.
- `mobile/src/core/session.ts` persists tokens with Expo SecureStore using `cove.auth.session.v1`.
- `mobile/src/core/api.ts` owns JSON requests, bearer authentication, token refresh, and session hydration.
- `mobile/src/core/chat.ts` and `mobile/src/core/sse.ts` own streaming chat transport and event decoding.
- `EXPO_PUBLIC_API_BASE_URL` is the mobile API base URL override; the development fallback is `http://localhost:8000`.
- `EXPO_ALLOW_INSECURE_HTTP=true` is a development-only switch that relaxes iOS transport security through `app.config.ts`.

### 3.5 Client duplication boundary

The React/Vite and Expo clients intentionally have separate UI, navigation, storage, and transport implementations. When the server contract changes, audit both:

- `frontend/src/features/**/api.ts` and related types/tests.
- `mobile/src/core/*.ts`, providers, screens, and related tests.

Do not share browser-only implementations with React Native. Shared behavior may be extracted only when it remains platform-neutral and does not couple localStorage, SecureStore, DOM streams, Expo fetch, or native navigation.

## 4. Server Package

### 4.1 Process entry points

| Process | Entry point | Responsibility |
| --- | --- | --- |
| HTTP API | `cmd/api/main.go` | Loads config, creates `ServiceContext`, mounts Gin router, listens on configured address |
| Worker | `cmd/worker/main.go` | Consumes Redis/asynq background jobs |
| Scheduler | `cmd/scheduler/main.go` | Enqueues scheduled maintenance and memory work |
| Migration | `cmd/migration/main.go` | Applies database migrations |
| Codegen | `cmd/codegen/main.go` | Generates/checks routes, repositories, prompts, and OpenAPI artifacts |

### 4.2 Layering and dependency direction

```text
cmd/*
  → transport/http/routes
  → transport/http/handler
  → logic
  → domain/flow + core
  → repository interfaces
  → infrastructure adapters
  → external systems
```

- `internal/transport/http/` owns Gin setup, CORS, middleware, response envelopes, docs, and route registration.
- `internal/transport/http/handler/` validates and maps transport input/output; handlers should delegate business work.
- `internal/logic/` owns application use-case orchestration for auth, chat, conversations, documents, model configuration, personas, skills, tools, and knowledge bases.
- `internal/domain/flow/` owns longer domain workflows, especially chat orchestration.
- `internal/core/` owns reusable, business-neutral engines such as agent/ReAct, context, LLM abstractions, MCP, memory, prompts, RAG, security, skills, and tools.
- `internal/repository/` defines persistence contracts and implementations grouped by backing store.
- `internal/infrastructure/` adapts PostgreSQL, Redis, Elasticsearch, Neo4j, queues, realtime, storage, security, and LLM providers.
- `internal/svc/ServiceContext` is the composition root for infrastructure clients, repositories, core services, token/cipher services, and prompt/skill registries.
- Dependencies should point inward. Core packages must not import HTTP handlers or concrete database adapters.

### 4.3 HTTP API shape

`internal/transport/http/router.go` mounts all public application routes under `/api`:

- Public auth: `/api/auth/register`, `/api/auth/login`, `/api/auth/refresh`.
- Authenticated user operations: profile, password, avatar, conversations, chat, agent/persona configuration, model configuration, MCP, knowledge bases, documents, images, tags, skills, and tool configuration.
- Streaming chat: `POST /api/chat/stream` using Server-Sent Events.
- Health: `GET /api/health`.
- Swagger/OpenAPI routes are enabled by server configuration.

Normal JSON responses use the server envelope (`code`, `message`, optional `data` and validation `errors`). Streaming chat uses named SSE events and therefore must be validated against both the handler/logic implementation and client event unions, not only the JSON schemas in OpenAPI.

### 4.4 Persistence and external dependencies

| Dependency | Primary role |
| --- | --- |
| PostgreSQL/GORM | Users, refresh tokens, conversations/messages, agent/persona/model/MCP/tool/skill/knowledge metadata |
| Redis | Realtime broker and async task queue |
| Elasticsearch | RAG chunk index, vector/BM25 retrieval, reranking input |
| Neo4j | Optional long-term memory graph |
| Object storage | Uploaded documents/images and signed delivery URLs |
| LLM providers | Chat, embeddings, classification, description, and prompt-assisted operations |
| MCP servers | Per-user external tool discovery and invocation |

Infrastructure clients are created in `svc.New`, bound to repository interfaces, and closed through `ServiceContext.Close`. New external dependencies should be introduced through an interface at the consuming layer and wired at the composition root.

## 5. Critical End-to-End Flows

### 5.1 Authentication

```text
Client login/register
  → /api/auth/*
  → auth handler
  → auth logic
  → PostgreSQL repositories + TokenIssuer
  → access/refresh tokens
  → client session storage
  → /api/auth/me hydration
```

- Web/Wails persists the session in localStorage; Expo persists it in SecureStore.
- Authenticated requests attach a bearer access token.
- A 401 triggers at most one refresh and retry; refresh-token failure clears the session.
- Server refresh tokens rotate, so both clients must persist the returned replacement token.

### 5.2 Streaming chat

```text
Chat screen
  → list/create conversation and message history
  → POST /api/chat/stream
  → JWT middleware
  → ChatHandler.ChatStream
  → ChatStreamLogic.ChatStream
  → persist user message + subscribe realtime topic
  → domain/flow/chat.Orchestrator.Run
  → context/history + active persona + agent config
  → built-in tools + enabled knowledge tools + available MCP tools
  → ReAct/LLM stream
  → Redis realtime events
  → Gin SSE response
  → client SSE parser
  → incremental think/tool/token/done UI state
```

The event sequence may include `meta`, `think`, `tool_call`, `tool_result`, `token`, `done`, and `error`. When changing an event, update the server event type/mapper/streaming tests and both client parsers, type unions, reducers, and UI tests.

### 5.3 RAG ingestion and retrieval

```text
Upload or URL import
  → document/image logic
  → object storage + PostgreSQL metadata
  → Redis background task
  → worker parse/crawl/describe/compress/chunk/embed
  → Elasticsearch chunk index
  → enabled knowledge tool during chat
  → hybrid search + normalization/rerank
  → cited context passed to the agent
```

- Reusable RAG primitives live under `internal/core/rag/`.
- Transport- and user-specific decisions belong in logic/domain orchestration, not in core packages.
- Background processing status must remain observable through the document/image API.

### 5.4 MCP tools

```text
User MCP configuration
  → encrypted server configuration in PostgreSQL
  → chat tool registry assembly
  → bounded concurrent MCP discovery
  → tool invocation through core/mcp
  → tool_call/tool_result SSE events
  → per-turn MCP session cleanup
```

An unavailable MCP server should not remove built-in tools or fail the entire chat turn unless the requested behavior explicitly requires strict failure.

## 6. Contract and Generated-Code Rules

- Verify every added or changed API call against `packages/server/docs/openapi.json` before editing a client.
- If implementation and OpenAPI differ, treat server behavior as the runtime truth and update the generated contract through the server codegen workflow.
- Run `make docs` from the `cove/` workspace root to check route, repository, OpenAPI, and prompt generation consistency.
- Do not hand-edit generated Wails bindings or server-generated mapper/OpenAPI output when the source generator should be changed instead.
- Preserve exact trailing-slash behavior for Gin routes where clients depend on it; redirects can break CORS in embedded webviews.
- Treat SSE event contracts as a cross-package interface even when OpenAPI cannot fully describe them.

## 7. Change Routing

Use these ownership rules before editing:

| Change | Start in | Also inspect |
| --- | --- | --- |
| Desktop window or Go bridge | `packages/app/internal/app`, `packages/app/internal/services` | generated bindings and React consumers |
| Web/Wails screen | `packages/app/frontend/src/features` | app shell, API wrapper, tests, native bridge when applicable |
| Native mobile screen/navigation | `packages/app/mobile/src/app` | provider/core API, simulator behavior, navigation lifecycle |
| API endpoint or schema | `packages/server/internal/transport/http` + logic | OpenAPI, web client, Expo client |
| Chat agent behavior | `packages/server/internal/logic/chat`, `internal/domain/flow/chat` | core agent/tools, SSE mapper, both chat UIs |
| RAG primitive | `packages/server/internal/core/rag` | worker pipeline, repositories, chat knowledge tool, tests |
| Persistence change | repository interface + adapter + migration | `ServiceContext`, logic, generated artifacts |
| Async task | `packages/server/internal/worker`, queue types | worker entry, producer, scheduler, status API |

## 8. Validation Boundaries

- End-to-end work must follow `.codex/rules/e2e-testing.md`; do not label unit, component, router, build, or screenshot-only checks as E2E proof.
- App desktop/web: run frontend type/build/tests and the relevant Go tests from `packages/app/`.
- Expo mobile: run lint, typecheck, tests, and validate visual/navigation work in an iOS Simulator.
- Server: run targeted Go tests from `packages/server/`, then broader tests as risk requires; run `make docs` from the workspace root for contract or generated-code changes. Before declaring a backend requirement complete, apply the real migration path and pass its real scenario against a local real database.
- Cross-package API/SSE changes: validate server tests plus both web and Expo client tests/typechecks, then run final frontend/backend E2E against the workspace's isolated local database environment.
- Report app and server validation separately, including any dependency that could not be exercised locally.
