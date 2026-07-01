<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **boxify-api-go** (6276 symbols, 24945 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> If any GitNexus tool warns the index is stale, run `npx gitnexus analyze` in terminal first.

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `gitnexus_impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `gitnexus_detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `gitnexus_query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `gitnexus_context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `gitnexus_impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `gitnexus_rename` which understands the call graph.
- NEVER commit changes without running `gitnexus_detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/boxify-api-go/context` | Codebase overview, check index freshness |
| `gitnexus://repo/boxify-api-go/clusters` | All functional areas |
| `gitnexus://repo/boxify-api-go/processes` | All execution flows |
| `gitnexus://repo/boxify-api-go/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

## Testing

- When adding or modifying test functions, you must use Chinese comments to explain what the test verifies.

## Generated Code

- For generated code, key steps must include Chinese comments explaining what they do.

## Go Style

- Prefer using pointers wherever pointers can reasonably be used.

## Core Package Template

Use `internal/core/rag/search` as the implementation template for new packages under `internal/core`.

- Split package responsibilities into focused files: `types.go` for interfaces/request/result types, `options.go` for defaults and functional options, one main implementation file for behavior, and `*_test.go` for package-level behavior tests.
- Keep core packages business-neutral. Define small dependency interfaces inside the core package, accept dependencies through constructors, and let callers provide business filters, decoders, or adapters.
- Constructors should return pointers, initialize complete defaults first, then apply functional options. Options should ignore invalid zero values when preserving defaults is safer.
- Separate package-level options from per-call request options when both long-lived configuration and request-specific overrides exist.
- Prefer generic result/source types when the core behavior is reusable but callers need typed metadata.
- Keep exported methods small and orchestration-focused. Move query building, normalization, filtering, sorting, decoding, and fallback behavior into private helper functions.
- Key implementation steps must include Chinese comments explaining what they do, especially non-obvious algorithms, fallback behavior, score normalization, filtering, and external-service query construction.
- Tests should use local fakes for dependencies, cover defaults/options, dependency errors, request overrides, helper edge cases, fallback paths, and result shaping. Every test function must include a Chinese comment explaining what the test verifies.
