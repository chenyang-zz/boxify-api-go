# Cove Server Rules

These rules apply to work under `packages/server/`. Read `packages/server/AGENTS.md` as the authoritative package instruction file and read `.codex/rules/architecture.md` before making changes. Run Git and shared Make targets from the `cove/` monorepo root; run direct Go commands from `packages/server/`.

## 1. Architecture and Dependency Direction

- Keep process entry points under `cmd/` thin: load configuration, construct `svc.ServiceContext`, register routes or tasks, run the process, and close resources.
- Preserve the dependency direction:
  - `transport/http/routes` registers endpoints and generation annotations
  - `transport/http/handler` binds transport input, reads authenticated context, invokes logic, and writes responses
  - `logic` orchestrates application use cases
  - `domain/flow` owns multi-step domain workflows
  - `core` owns reusable business-neutral engines
  - `repository` defines persistence contracts
  - `infrastructure` implements databases, queues, storage, realtime, channels, security, and provider adapters
  - `svc.ServiceContext` is the composition root
- Do not call concrete infrastructure directly from handlers or reusable core packages.
- Define small interfaces at the consuming layer. Add a concrete adapter under `internal/infrastructure/`, then wire it in `svc.New` or a focused builder.
- Avoid mutable package globals. Inject dependencies so tests can use local fakes.
- Prefer pointer constructors and pointer receivers wherever pointers are reasonable, consistent with `packages/server/AGENTS.md`.

## 2. HTTP, Authentication, and API Contracts

- Register routes only under `internal/transport/http/routes/`. Keep the exact Gin path and trailing-slash behavior stable because redirects can break embedded clients and CORS.
- Use request DTOs from `internal/transport/http/request/` and response DTOs from `internal/transport/http/response/`. Do not expose GORM models directly.
- Keep handlers transport-only: bind JSON, query, URI, or multipart input; convert binding failures with `xerr.Validation`; obtain authenticated identity from request context; call one logic entry point; return through the response package.
- Never trust a client-supplied `user_id` for authorization. Derive user identity from the authenticated context and pass it explicitly into logic and repository operations.
- Use `response.OK` and `response.FromError` so normal JSON responses retain the `{code, message, data?, errors?}` envelope.
- Use `xerr` kinds and safe messages for caller-visible failures. Never return a raw database, provider, encryption, or internal error string to the client.
- Add or update route annotations and DTO tags whenever externally visible request, response, authentication, or SSE behavior changes.
- Treat `docs/openapi.json` as generated output. Change routes, annotations, DTOs, or generators first, then run `make docs` from the workspace root; do not hand-edit the contract.
- After a requirement introduces a new HTTP endpoint and `make docs` has regenerated the contract, add a preset request to the matching existing module collection in the Postman `boxify-go` workspace. Include the exact method, a `{{base_url}}`-based URL, path/query variables, a representative request body and content type when applicable, and the correct authentication mode. Do not create a duplicate collection or add automated Postman tests unless the task explicitly asks for them. Verify the request is visible in Postman before marking the endpoint work complete; otherwise report Postman as an incomplete validation item.
- Update the server contract before app consumers. Preserve existing web and mobile compatibility unless the task explicitly coordinates a breaking change across packages.
- Treat SSE names and payloads as a cross-package contract even when OpenAPI cannot fully express them. Update event types, mappers, stream tests, and both app parsers together.

## 3. Logic and Domain Behavior

- Put use-case orchestration in `internal/logic/<domain>/`. Keep handlers free of persistence decisions, provider calls, and business branching.
- Use `internal/domain/flow/` for workflows that coordinate several domain steps, especially chat and agent execution. Do not hide a second workflow inside a handler or repository.
- Keep exported logic methods easy to scan. Move parsing, normalization, validation, selection, sorting, fallback, and mapping details into focused private helpers.
- Make defaults promised by API descriptions, tool schemas, prompts, or documentation explicit in code and tests.
- Do not silently override explicit user input. Any automatic correction or fallback must have a clear guard, a documented tolerance, and tests for the explicit-input case.
- When fixing a timezone, UUID, ownership, status, or contract bug, audit all affected layers: request/response DTOs, logic, mapper, repository, task payload, prompt/tool schema, SSE mapping, and app consumers.

## 4. Persistence, Ownership, and Transactions

- Define persistence interfaces under `internal/repository/` and PostgreSQL implementations under `internal/repository/postgres/`. Always propagate `context.Context` with `WithContext(ctx)` or the pgx context API.
- Scope user-owned reads, updates, and deletes by authenticated user ID at the repository boundary. Do not fetch by object ID alone and check ownership only after loading.
- For a model without a direct `UserID`, use the code generator's parent-scope definition so generated queries join through the owning record. Add tests for cross-user access denial.
- Use generated repository field selectors for partial updates. Do not pass arbitrary column names from request data.
- Use `svc.ServiceContext.WithTx` for GORM transactions that must share repository bindings. Keep the transaction callback focused on database work.
- Never perform slow external HTTP, LLM, MCP, object-storage, queue-wait, or other network I/O while holding a database transaction or row lock. Perform required external work before opening the transaction or after committing whenever semantics allow.
- Keep transactions short and never make serial external calls inside a database loop. Use a provider batch API or bounded concurrency outside the transaction.
- For multi-system writes such as storage → database → queue, define the failure state and compensation path. If enqueue fails after persistence, mark the record failed or use a durable outbox; never leave a silent pending record.
- Validate external identifiers before forwarding them. Accept only the documented type and format; reject local paths, `undefined`, mock IDs, malformed UUIDs, and unsupported URL schemes at the boundary.

## 5. Models and Migrations

- Keep persistent models under `internal/models/`. Add every new persistent model to `models.MigrationModels()` and add or update migration tests.
- Use `make migration` from the workspace root or `go run ./cmd/migration` from `packages/server/` for the current GORM `AutoMigrate` workflow.
- Do not assume `AutoMigrate` safely performs destructive changes, data rewrites, constraint tightening, or zero-downtime type changes. Use an explicit reviewed expand–migrate–contract plan for those changes.
- Preserve UUID initialization through the existing model hooks and generator. Do not create alternate ID assignment conventions per model.
- Treat JSON/JSONB scanner compatibility as persisted data behavior. When changing a scanner or stored shape, test nil, current, malformed, and supported legacy representations.
- Test database constraints and ownership behavior with a role and configuration representative of production whenever privilege differences could hide a failure.

## 6. External Calls and Resource Safety

- Before adding synchronous external work to a request path, decide:
  1. If equivalent requests can share the result, cache it with an explicit TTL and invalidation strategy.
  2. If the response does not require completion, enqueue an asynq task.
  3. If the response requires the result, use a bounded `context` deadline and a configured client timeout.
- Do not use an unbounded `http.DefaultClient`. Construct clients in infrastructure or the composition root with explicit timeouts and close any owned resources during `ServiceContext.Close`.
- A synchronous provider call should normally target ten seconds or less. Document and test any longer timeout required by a specific protocol; never combine it with an open transaction.
- Do not issue one external HTTP request per item in an unbounded loop. Prefer provider batch endpoints, caching, or bounded concurrency with cancellation.
- Use negative caching only when repeated failure retries would amplify an outage, and keep the negative TTL short enough to recover automatically.
- Make fail-open versus fail-closed behavior explicit. Optional MCP or reranking failure may degrade only where product semantics allow it; authentication, ownership, encryption, and required persistence must fail closed.
- Never log or return API keys, JWTs, refresh tokens, passwords, encrypted secret material, full private documents, or sensitive provider payloads.

## 7. Queues, Workers, and Scheduling

- Define task names, queue names, typed payloads, and validating constructors in `internal/domain/types/task.go`.
- When adding a task, update `TaskNames()`, map it to a real handler in `internal/worker/tasks/registry.go`, add the handler and tests, and register scheduling only when required. Never let an implemented task fall through to the placeholder handler.
- Keep task payloads minimal and stable. Prefer durable IDs over embedding large records or secrets.
- Make handlers safe for retry. Re-read current state, make repeated execution idempotent, and avoid duplicating external side effects.
- Distinguish permanent failures from transient failures. Skip retries for invalid payloads or missing terminal resources; return transient provider, queue, or infrastructure errors so asynq can retry.
- Persist observable progress and terminal failure state for long-running document, image, RAG, gateway, or agent tasks.
- Keep the scheduler responsible for enqueueing, not executing business logic. Deployment must provide one scheduling authority or task-level deduplication/idempotency when multiple scheduler replicas are possible.
- Use a durable outbox or an equivalent recoverable state machine for side effects that must survive a process crash between database commit and enqueue.

## 8. Agent, RAG, MCP, and Gateway Rules

- Use `internal/core/rag/search` as the package template for new `internal/core` work: focused files, small consumer-owned interfaces, complete defaults, functional options, request-level overrides, private helpers, and local fake tests.
- Keep core behavior business-neutral. Inject user filters, source decoders, permissions, and business adapters from domain or logic layers.
- Do not implement a second business calculation solely for an agent tool. Reuse the same repository or logic semantics as the corresponding API path, or perform and test a field-by-field comparison.
- Keep tool names, JSON schemas, descriptions, defaults, permission policy, execution behavior, and result formatting synchronized.
- Preserve built-in tools when an optional MCP server is unavailable unless the request explicitly requires strict failure. Bound MCP discovery and execution with context deadlines and clean up per-turn sessions.
- Make retrieval thresholds, rerank fail-open behavior, fallback sources, and scoring basis explicit through typed options. Do not silently mix incompatible score scales.
- For gateway ingestion and delivery, preserve idempotency keys, state transitions, recoverable enqueue errors, and outbox delivery semantics. Test duplicate events and retries.

## 9. Errors, Logging, and Observability

- Wrap internal causes with `%w`, `xerr.Wrapf`, or a typed sentinel so `errors.Is` and `errors.As` continue to work.
- Keep raw Go error strings lower-case and without trailing punctuation. Use `xerr` safe Chinese messages for user-visible API errors where the existing domain does so.
- Use `slog` through the existing `xlog` component pattern. Prefer `InfoContext`, `WarnContext`, and `ErrorContext` with the request or task context.
- Use stable structured keys such as `user_id`, resource IDs, task name, queue, provider, status, duration, and count. Do not encode searchable fields only inside a prose message.
- Log state transitions and failure causes once at the owning layer. Avoid duplicate handler, logic, and repository logs for the same error unless each adds distinct context.
- Sanitize logs before including URLs, headers, prompts, request bodies, or third-party responses.

## 10. Go Documentation and Code Style

- Follow `gofmt` and the Go version declared in `packages/server/go.mod`. Keep imports grouped and let tooling format source.
- Accept `context.Context` as the first parameter for request-scoped, I/O, cancellation-aware, or long-running work.
- Use short idiomatic names, standard initialisms, `New`/`New<Type>` constructors, and `-er` names for focused one-method interfaces.
- Prefer explicit errors to panic for normal failures. Reserve panic for impossible initialization invariants where process startup cannot continue.
- Under `internal/core`, follow the stricter documentation requirements in `packages/server/AGENTS.md`: package comments, Chinese complete-sentence Go docs for exported declarations, and Chinese comments for non-obvious algorithms and fallback behavior.
- Keep comments focused on behavior and invariants rather than restating syntax. Update stale comments when behavior changes.
- Do not add a dependency, abstraction, option, or generic type until it serves a concrete current use case or test seam.

## 11. Generated Code and Contract Checks

- Never hand-edit a file headed `// Code generated by codegen; DO NOT EDIT.`. Change the model, route annotation, DTO, template, or generator and regenerate.
- Handler and logic skeletons do not carry the generated header and may be completed by hand, but route generation must remain idempotent.
- Prefer the current `@` route annotation protocol for new endpoints. Declare authentication, user ID extraction, request DTO, response DTO, and SSE event type accurately.
- Run `make docs` from the workspace root after route, DTO, model/repository, prompt, or OpenAPI-related changes.
- Run `go run ./cmd/codegen doctor --format json` when changing routes, DTOs, repository scopes, or generators.
- Key steps in generated code and generator templates must include Chinese comments as required by `packages/server/AGENTS.md`.

## 12. Tests and Validation

- Place a Chinese comment immediately above every added or modified Go test function explaining the behavior it verifies.
- Prefer table-driven tests and local fakes. Cover success, invalid input, nil dependencies, ownership isolation, dependency errors, transaction rollback, retries, idempotency, timeout/cancellation, fallback, and persisted result shape as applicable.
- Keep test failure messages actionable: name the operation, actual value, and expected value. Use focused diffs for large structures.
- Do not make unit tests depend on live external providers. Test adapters with controlled servers or fakes and keep credentials out of fixtures.
- A backend requirement is not complete until the changed behavior has passed a real scenario against a local real database. Start the local database through the project-owned OrbStack workflow, apply the real server migration path, exercise the public API, worker, scheduler, gateway, or repository boundary that owns the behavior, and verify the persisted outcome.
- Unit tests, fake repositories, `httptest` routers, successful builds, and code-generation checks remain required where applicable, but none of them replaces the local-database scenario. If the requirement changes Redis, Elasticsearch, or Neo4j persistence semantics, exercise the corresponding local real dependency as part of the scenario.
- Use synthetic run-unique data in local real-database validation and clean up only resources created by that run. Never reuse a developer's normal database or copy remote credentials into a local fixture.
- If the local real-database scenario cannot run, report it as an explicit completion blocker with the missing dependency or configuration; do not mark the backend requirement complete based only on mocked tests.
- Run focused package tests first. As risk requires, run:
  - `gofmt` on changed Go files
  - `go vet ./...`
  - `go build ./...`
  - `go test -count=1 ./...` or `go test -count=1 -race <changed packages>`
  - workspace-root `make docs` and codegen doctor for generated or contract changes
- Report server validation separately from app validation. Include the local database used, migration result, real scenario and observable persisted result, cleanup status, and any Redis, Elasticsearch, Neo4j, object-storage, model-provider, or MCP dependency that was not exercised.

## 13. Applying External Backend Notes

- Treat external backend notes as candidate failure patterns, not as Cove implementation facts.
- Do not copy Python, hatchling, uv, Alembic, asyncpg, arq, tenant-RLS, Doc Center, GeTui, or paths from a foreign backend into Cove rules or code unless Cove intentionally adopts that technology.
- Translate reusable lessons into the current Go architecture: strict boundary validation, short transactions, bounded external I/O, batch or queued work, user-scoped repositories, idempotent tasks, safe migrations, and end-to-end contract tests.
- Promote a workaround to a permanent Cove rule only after the relevant dependency exists here or Cove reproduces the failure.
