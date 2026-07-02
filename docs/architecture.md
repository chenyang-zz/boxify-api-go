# Cove Agent Platform

Cove 是一个 AI 助手平台的后端：对话、RAG、Agent、记忆、MCP——全部整合在一个 Go 代码库中。整体采用分层架构，HTTP 入口使用 Gin，但 Gin 被严格限制在 `transport/http` 层，不会泄漏到 domain、repository 或 infrastructure 包中。

## 依赖方向

```
        ┌─────────────────────────────────────────────┐
        │              cmd/ (入口)                      │
        │   api │ worker │ scheduler │ migration │ codegen │
        └────────────────────┬────────────────────────┘
                             │ 初始化
                             ▼
        ┌─────────────────────────────────────────────┐
        │          internal/svc/ (DI 容器)              │
        │   ServiceContext 聚合仓库、配置、基础设施      │
        └────────────────────┬────────────────────────┘
                             │ 注入
         ┌───────────────────┼───────────────────┐
         ▼                   ▼                   ▼
┌────────────────┐ ┌─────────────────┐ ┌─────────────────────┐
│ transport/http │ │     logic/      │ │   internal/core/    │
│   HTTP 传输     │ │   业务编排       │ │    核心业务能力       │
│  Gin/DTO/中间件 │ │ 跨仓库 & domain  │ │ agent/llm/rag/...   │
└────────────────┘ └────────┬────────┘ └─────────────────────┘
                            │
                   ┌────────┴────────┐
                   ▼                 ▼
           ┌──────────────┐ ┌─────────────────┐
           │  repository/ │ │     domain/      │
           │  数据访问     │ │   领域类型/事件   │
           │ GORM/Neo4j/ES│ │   框架无关契约    │
           └──────┬───────┘ └─────────────────┘
                  │
                  ▼
           ┌──────────────────┐
           │  infrastructure/  │
           │  外部适配器        │
           │ db/es │ db/neo4j │
           │ llm   │ queue    │
           │ storage │ realtime│
           └──────────────────┘
```

## 模块职责

### transport/http
- Gin 路由、中间件、请求/响应 DTO。
- 负责 JWT 认证、SSE 流式响应头设置。
- **不引入**任何业务类型。

### logic
- 用例编排，聚合多个 repository 与 domain 交互。
- 持有 `ServiceContext`，通过构造函数注入。

### domain
- 纯领域类型与领域事件（发布订阅）。
- 零外部依赖（framework-free）。

### repository
- 持久化接口 + GORM / Neo4j / Elasticsearch 实现。
- SQL 由 `db/queries/` 生成（sqlc 风格）。

### infrastructure
- 适配外部系统：PostgreSQL、Elasticsearch、Neo4j、Redis、COS、LLM Provider。
- 每个适配器独立子包，互不依赖。

### internal/core
- 横切业务能力（不随 HTTP/schema 变化频繁）：agent、llm、rag、memory、mcp、prompt、security。
- 内部 `rag/` 含 8 个子包，构成完整检索增强生成引擎。

## RAG Pipeline

11 步流水线将原始来源转换为可检索的知识：

```
Source → Crawl → Parse → Describe → Compress → Chunk → Embed
                                                          ↓
  Answer ← Classify ← Rerank ← Search ← Index ◄──────────┘
```

| 步骤 | 包 | 职责 |
|------|------|------|
| 1. Crawl | rag/webcrawl | 抓取 + 重定向跟踪 + SSRF 防护 |
| 2. Parse | rag/documentparse | TXT/MD/HTML/DOCX/PDF 文本提取 |
| 3. Describe | rag/imagedescribe | 视觉模型结构化描述 |
| 4. Compress | rag/imagecompress | 模型输入预处理 |
| 5. Chunk | rag/chunker | 基于 tiktoken 的 parent/child 分块 |
| 6. Embed | llm/ (Provider) | 生成稠密向量 |
| 7. Index | Elasticsearch | Bulk upsert 写入 chunk 索引 |
| 8. Search | rag/search | 向量 + BM25 混合召回 |
| 9. Rerank | rag/search | 分数归一化 + 重排序 |
| 10. Classify | rag/classifier | LLM 自动标签（非阻塞） |
| 11. Answer | agent | 引用参考 + 生成回答 |

所有提示词模板位于 `internal/core/rag/prompt/`，通过 `internal/core/prompt/` 统一渲染。

## 异步任务

通过 asynq + Redis 驱动后台任务：

- `parse:document` — 文档解析与分块
- `parse:image` — 图片内容提取
- `memory:extract` — 记忆提取
- `memory:consolidate` — 每日记忆合并
- `research:run` — 研究任务执行

## 启动接受路径

1. `docker compose -f deployments/docker-compose.yml up -d` 启动基础服务。
2. `make migration` 执行数据库迁移。
3. `make api` 启动 HTTP 服务。
4. 注册/登录 → 配置模型 → 上传内容 → 流式对话。
