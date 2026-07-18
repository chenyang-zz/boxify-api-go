# Cove

Cove 是一个统一管理桌面端、Web、移动端与后端服务的 monorepo。项目代码托管在 [`chenyang-zz/cove`](https://github.com/chenyang-zz/cove)，项目名称统一为 **Cove**。

## 目录结构

```text
cove/
├── packages/
│   ├── app/       # Wails 桌面端、React/Vite Web 与 Expo 移动端
│   └── server/    # Go API、Worker、Gateway、Scheduler 与迁移工具
├── e2e/           # 跨前后端的本地真实依赖测试编排
├── .github/       # monorepo 级 CI/CD 与协作模板
└── Makefile       # 统一开发入口
```

原前端仓库 [`chenyang-zz/cove-legacy`](https://github.com/chenyang-zz/cove-legacy) 的提交历史已完整导入 `packages/app/`；原 `cove-api` 后端历史保留为当前仓库主线，后端代码位于 `packages/server/`。

## 快速开始

环境要求：Go、Node.js、pnpm、Task，以及运行本地依赖时所需的 OrbStack/Docker CLI。

```bash
make help

# 前端检查
make app-go-test
make app-frontend-test
make app-mobile-typecheck

# 后端开发
make migration
make api
```

前端和后端仍保留各自的依赖清单与构建方式。Git 命令与共享 Make 目标统一从仓库根目录运行；直接的 Go、pnpm 或 Task 命令从对应 package 目录运行。

## 文档

- [架构与开发边界](.codex/rules/architecture.md)
- [前端规则](.codex/rules/frontend.md)
- [后端规则](.codex/rules/backend.md)
- [E2E 测试说明](e2e/README.md)
- [后端 OpenAPI](packages/server/docs/openapi.json)

贡献方式与安全披露流程见 [CONTRIBUTING.md](CONTRIBUTING.md) 和 [SECURITY.md](SECURITY.md)。
