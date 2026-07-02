# Cove API — Go

<p align="center">
  <b>Cove is an AI assistant platform backend</b><br/>
  Conversations, RAG, agents, memory, MCP — all in one Go codebase.
</p>

<p align="center">
  <!-- Badges -->
  <img src="https://img.shields.io/github/go-mod/go-version/chenyang-zz/cove-api?logo=go&logoColor=white&style=flat" alt="Go version" />
  <img src="https://img.shields.io/github/v/release/chenyang-zz/cove-api?style=flat&color=blue" alt="Release" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=flat" alt="License" />
</p>

<p align="center">
  <a href="#features">Features</a> ·
  <a href="#quick-start">Quick Start</a> ·
  <a href="#architecture">Architecture</a> ·
  <a href="#rag-pipeline">RAG Pipeline</a> ·
  <a href="#configuration">Configuration</a> ·
  <a href="#development">Development</a> ·
  <a href="https://github.com/chenyang-zz/cove-api/blob/main/docs/architecture.md">Docs</a>
</p>

---

## Features

- **Conversations** — Streaming chat with SSE, multi-turn context management
- **RAG Engine** — Full retrieval-augmented generation: crawl, parse, chunk, embed, search, rank
- **Agent Orchestration** — Tool-calling agents with persona and memory
- **Memory** — Long-term memory extraction, consolidation, and recall
- **MCP Integration** — Connect external tools via Model Context Protocol
- **Real-time** — Event streaming over Redis
- **Document Processing** — Multi-format parsing: TXT, Markdown, HTML, DOCX, PDF
- **Content Classification** — LLM-powered auto-tagging with graceful degradation

## Tech Stack

| Layer | Technology |
|---|---|
| **Language** | Go 1.25 |
| **HTTP** | Gin (transport only — no leakage into domain) |
| **Database** | PostgreSQL (pgx + GORM) |
| **Search** | Elasticsearch 8.x (hybrid vector + BM25) |
| **Graph** | Neo4j 5.x |
| **Queue** | Redis + asynq |
| **LLM** | Anthropic / OpenAI |
| **Auth** | JWT |
| **Storage** | Tencent Cloud COS (local fallback) |
| **Observability** | slog + OpenTelemetry |

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose

### 1. Start dependencies

```bash
docker compose -f deployments/docker-compose.yml up -d
```

| Service | Port |
|---|---|
| PostgreSQL | 5432 |
| Elasticsearch | 9200 |
| Neo4j | 7474 (HTTP), 7687 (Bolt) |
| Redis | 6379 |

### 2. Configure

```bash
cp configs/config.yml.example configs/config.yml
# Edit configs/config.yml with your LLM keys and connection strings
```

### 3. Migrate database

```bash
make migration
```

### 4. Run

```bash
make api      # API server on :8000
make worker   # Background worker (separate terminal)
make scheduler # Cron jobs (optional)
```

## Architecture

```
transport/http/    →  Gin routing, middleware, request/response DTOs
    ↓
logic/             →  Business orchestration across repositories & domain
    ↓
repository/        →  Data access (GORM / Neo4j / Elasticsearch)
    ↓
domain/            →  Domain types, events, interfaces
```

Cross-cutting concerns (LLM, memory, RAG, MCP, security) live in `internal/core/` and are wired through a single `ServiceContext` — see `internal/svc/context.go`.

### Core packages

```
internal/core/
├── agent/          # Agent orchestration & tool dispatch
├── llm/            # LLM provider abstraction
├── rag/            # Retrieval-augmented generation engine
│   ├── chunker/        # Token-aware parent/child chunking
│   ├── classifier/     # LLM content classification
│   ├── documentparse/  # Multi-format text extraction
│   ├── imagecompress/  # Model-ready image preprocessing
│   ├── imagedescribe/  # Vision model structured descriptions
│   ├── prompt/         # RAG prompt templates (embedded)
│   ├── search/         # Hybrid vector + BM25 search
│   └── webcrawl/       # Web fetching with SSRF guard
├── memory/         # Long-term memory extraction & consolidation
├── mcp/            # Model Context Protocol integration
├── prompt/         # Template rendering (FS, memory, legacy fallback)
└── security/       # JWT, encryption, secret management
```

## RAG Pipeline

Cove's 11-step ingestion pipeline turns raw sources into retrievable knowledge:

```
Source
  │
  ▼
1. Crawl       ──── webcrawl/     Fetch with retry, redirect tracking, SSRF guard
  │
  ▼
2. Parse       ──── documentparse/ Extract text from TXT/MD/HTML/DOCX/PDF
  │
  ▼
3. Describe    ──── imagedescribe/ Vision model → description, OCR, objects, scene
  │
  ▼
4. Compress    ──── imagecompress/ Downscale & re-encode for model input
  │
  ▼
5. Chunk       ──── chunker/       Parent/child token chunks via tiktoken
  │
  ▼
6. Embed       ──── (provider)     Generate dense vectors via LLM provider
  │
  ▼
7. Index       ──── Elasticsearch Bulk upsert into chunk index
  │
  ▼
8. Search      ──── search/        Hybrid vector + BM25 recall
  │
  ▼
9. Rerank      ──── search/        Score normalization & reranker
  │
  ▼
10. Classify   ──── classifier/    LLM auto-tagging (non-blocking)
  │
  ▼
11. Answer     ──── agent/         Cite references, generate response
```

All prompt templates live in `internal/core/rag/prompt/` and are rendered through `internal/core/prompt/`.

## Configuration

Key sections in `configs/config.yml`:

```yaml
database:
  postgres: "postgres://user:pass@localhost:5432/cove"

redis:
  addr: "localhost:6379"

elasticsearch:
  addresses: ["http://localhost:9200"]

rag:
  chunk_index: "cove_chunks"
  embedding_dim: 1024

llm:
  provider: "anthropic"   # or "openai"
  api_key: "${LLM_API_KEY}"

jwt:
  secret: "${JWT_SECRET}"

storage:
  driver: "cos"           # or "local"
```

## Development

### Code generation

Cove ships a built-in codegen (`cmd/codegen/`) that scans Go annotations to produce:

| Command | Output |
|---|---|
| `make gen-route` | Router registration |
| `make gen-repository MODEL=User LABEL=用户` | Type-safe repository |
| `make gen-docs` | OpenAPI 3.0 spec |

### API routes

All routes are mounted under `/api/`:

| Domain | Description |
|---|---|
| `/api/health` | Health check (public) |
| `/api/auth` | Registration / login |
| `/api/models` | Model configuration |
| `/api/chat` | Streaming conversation |
| `/api/conversations` | Conversation management |
| `/api/documents` | Document CRUD |
| `/api/knowledge-bases` | Knowledge base management |
| `/api/agents` | Agent configuration |
| `/api/mcp-servers` | MCP server integration |

Authenticated routes are protected by JWT middleware.

### Async tasks

Powered by asynq + Redis:

| Task | Description |
|---|---|
| `parse:document` | Document parsing & chunking |
| `parse:image` | Image content extraction |
| `memory:extract` | Memory extraction |
| `memory:consolidate` | Daily memory consolidation |
| `research:run` | Research task execution |

## Project structure

```
.
├── cmd/                # Entrypoints
│   ├── api/            # HTTP server
│   ├── worker/         # Background processor
│   ├── scheduler/      # Cron scheduler
│   ├── migration/      # Database migrations
│   └── codegen/        # Code generation tool
├── configs/            # Configuration files
├── deployments/        # Docker Compose
├── db/                 # Migrations & queries
├── docs/               # Architecture & OpenAPI
├── internal/
│   ├── config/         # Config loader
│   ├── core/           # Core business capabilities
│   ├── domain/         # Domain types & events
│   ├── infrastructure/ # External adapters
│   ├── logic/          # Business logic layer
│   ├── models/         # GORM models
│   ├── repository/     # Data access
│   ├── svc/            # Service context (DI)
│   └── transport/http/ # HTTP transport layer
├── Makefile
└── README.md
```

---

<p align="center">
  Built with Go · LLM-powered · Open for contributions
</p>
