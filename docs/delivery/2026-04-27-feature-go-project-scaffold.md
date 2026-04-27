# 任务交付：项目骨架与 Docker 基础

## 任务背景

根据 `docs/project/navbox/tasks/task-01-project-scaffold.md`，建立 Navbox 的 Go 后端、React 前端和 Docker 部署基础，使项目具备可运行的最小骨架。

## 实现方案

1. 初始化 Go module。
2. 创建 Go 后端启动入口和分层目录。
3. 使用 Gin 提供 HTTP 服务。
4. 使用 Viper 加载配置。
5. 使用 Zerolog 输出结构化日志。
6. 使用 Uber Fx 管理依赖和生命周期。
7. 添加健康检查接口。
8. 创建 React + Vite + TypeScript 前端骨架。
9. 配置 `go:embed` 静态资源占位。
10. 添加 Dockerfile 和 Docker Compose。

## 文件变更

新增后端文件：

- `cmd/navbox/main.go`
- `internal/app/app.go`
- `internal/config/config.go`
- `internal/handler/health.go`
- `internal/logging/logger.go`
- `internal/middleware/request_logger.go`
- `internal/response/response.go`
- `internal/server/router.go`
- `internal/server/server.go`
- `internal/web/assets.go`
- `internal/web/dist/index.html`

新增前端文件：

- `web/package.json`
- `web/package-lock.json`
- `web/index.html`
- `web/tsconfig.json`
- `web/tsconfig.node.json`
- `web/vite.config.ts`
- `web/src/App.tsx`
- `web/src/main.tsx`
- `web/src/styles.css`
- `web/src/vite-env.d.ts`

新增部署与配置文件：

- `config/config.toml`
- `Dockerfile`
- `docker-compose.yml`
- `.dockerignore`

修改文件：

- `.gitignore`
- `docs/project/navbox/PROGRESS.md`

## 配置与依赖变更

Go 依赖：

- `github.com/gin-gonic/gin`
- `github.com/rs/zerolog`
- `github.com/spf13/viper`
- `go.uber.org/fx`

前端依赖：

- `react`
- `react-dom`
- `lucide-react`
- `vite`
- `typescript`
- `@vitejs/plugin-react`

## 新增接口

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/health` | 健康检查 |

## 测试结果

已执行：

```text
go build ./...
npm --prefix web run build
go test ./...
docker compose config
docker compose build navbox
docker compose up -d postgres navbox
curl -sS http://127.0.0.1:8080/api/v1/health
docker compose down
```

健康检查返回：

```json
{"code":200,"data":{"status":"ok"},"message":"success"}
```

## 风险与后续建议

1. 当前仅完成项目骨架，尚未接入 PostgreSQL 数据访问。
2. Go embed 当前有占位静态文件，生产镜像构建时会使用前端构建产物覆盖。
3. admin 认证、Session、数据模型将在后续任务实现。
