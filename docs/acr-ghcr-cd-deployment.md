# 阿里云 ACR + GHCR CD 部署指南

生产 compose 只部署 API 与 worker。GitHub Actions 对每个服务构建一次，并将同一提交 SHA 的镜像同时推送到阿里云容器镜像服务（ACR）和私有 GitHub Container Registry（GHCR）；服务器只从 ACR 拉取。

## 1. 阿里云 ACR 准备

在 ACR 创建一个命名空间，例如 `cove`，并创建 `cove-api` 和 `cove-worker` 两个私有仓库。GitHub Actions 使用有推送权限的账号或访问凭证；部署服务器使用独立的只读拉取账号或访问凭证。

ACR 镜像地址格式为：

```text
<ALIYUN_REGISTRY>/<ALIYUN_NAMESPACE>/cove-api:<git-sha>
<ALIYUN_REGISTRY>/<ALIYUN_NAMESPACE>/cove-worker:<git-sha>
```

`ALIYUN_REGISTRY` 使用 ACR 实例实际提供的 Docker 登录域名，不包含 `https://`。个人版与企业版均支持该格式。

## 2. GitHub 配置

在仓库 Variables 中配置：

| 名称 | 示例 | 说明 |
| --- | --- | --- |
| `ALIYUN_REGISTRY` | `registry.cn-hangzhou.aliyuncs.com` | ACR Docker 登录域名和可选端口，不含协议。 |
| `ALIYUN_NAMESPACE` | `cove` | ACR 命名空间。 |

在 `production` Environment Secrets 中配置：

| 名称 | 说明 |
| --- | --- |
| `ALIYUN_USERNAME` / `ALIYUN_PASSWORD` | GitHub Actions 使用的 ACR 推送账号或访问凭证。 |
| `DEPLOY_ALIYUN_USERNAME` / `DEPLOY_ALIYUN_PASSWORD` | 服务器使用的 ACR 只读拉取账号或访问凭证。 |
| `DEPLOY_HOST` / `DEPLOY_USER` / `DEPLOY_PORT` | 服务器 SSH 地址、用户和端口。 |
| `DEPLOY_SSH_KEY` | 部署私钥。 |
| `DEPLOY_SSH_FINGERPRINT` | SSH 主机 Ed25519 公钥的 SHA256 指纹。 |
| `DEPLOY_COMPOSE_DIR` | 服务器部署目录；未设置时为 `/opt/cove`。 |

工作流使用 `GITHUB_TOKEN` 推送 GHCR，不需要额外的 GitHub 镜像 secret。首次发布后，在 GitHub Packages 页面确认 `cove-api`、`cove-worker` 两个包保持私有，并关联到 `chenyang-zz/cove-api` 仓库。

镜像始终生成不可变的完整提交 SHA tag；从 `main` 发布时额外生成 `main` tag，从语义版本 tag 发布时额外生成对应版本 tag。部署始终使用完整 SHA，任一 ACR 或 GHCR 推送失败都会阻止部署。

## 3. 服务器初始化与验收

在 `DEPLOY_COMPOSE_DIR` 中准备：

```text
deployments/docker-compose.production.yml
configs/config.production.yml
```

PostgreSQL、Redis、Elasticsearch 和 Neo4j 不由本生产 compose 管理；它们必须已在外部 Docker 网络 `boxify_default` 中运行。生产 compose 会将 API/worker 加入该网络，`configs/config.production.yml` 应使用这些容器在该网络中的实际 DNS 名称。本服务器当前使用 `boxify-postgresql-1`、`redis-server`、`elasticsearch-server`、`neo4j-server`。

首次部署前确认网络和服务别名存在：

```bash
docker network inspect boxify_default
```

首次部署前，以服务器的 ACR 只读账号验证：

```bash
docker login <ALIYUN_REGISTRY>
docker pull <ALIYUN_REGISTRY>/<ALIYUN_NAMESPACE>/cove-api:<full-git-sha>
docker pull <ALIYUN_REGISTRY>/<ALIYUN_NAMESPACE>/cove-worker:<full-git-sha>

export ALIYUN_REGISTRY=<ALIYUN_REGISTRY>
export ALIYUN_NAMESPACE=<ALIYUN_NAMESPACE>
export IMAGE_TAG=<full-git-sha>
docker compose -f deployments/docker-compose.production.yml config
```

推送 `main`、推送语义版本 tag 或手动运行 CD 后，确认日志出现 `Deployment completed from Alibaba Cloud: <sha>`，并在服务器执行：

```bash
docker compose -f deployments/docker-compose.production.yml ps
curl -fsS http://127.0.0.1:8000/api/health
```
