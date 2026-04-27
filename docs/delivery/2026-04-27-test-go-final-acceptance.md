# 任务交付：测试、验收与部署文档

## 测试目标

根据 `docs/project/navbox/tasks/task-09-test-deploy.md`，完成后端测试、前端构建验证、Docker 部署验证、README、部署文档和验收文档。

## 用例设计

| 用例编号 | 目标 | 场景分类 | 描述 | 预期 |
| --- | --- | --- | --- | --- |
| T09-01 | Go 单元测试 | 自动化 | 执行 `go test ./...` | 全部通过 |
| T09-02 | 前端构建 | 自动化 | 执行 `npm --prefix web run build` | TypeScript 和 Vite 构建通过 |
| T09-03 | Go 构建 | 自动化 | 执行 `go build ./...` | 构建通过 |
| T09-04 | Docker 构建 | 冒烟 | 执行 `docker compose build navbox` | 镜像构建通过 |
| T09-05 | Docker 启动 | 冒烟 | 执行 `docker compose up -d` | 服务启动，PostgreSQL healthy |
| T09-06 | 首次启动密码 | 冒烟 | 使用独立空数据库 Compose project 启动 navbox | 日志输出 `Admin initial password generated` |
| T09-07 | 首页访问 | 冒烟 | 请求 `/` 和 `/admin` | 返回嵌入式前端 HTML |
| T09-08 | admin API | 冒烟 | 登录、创建 Tag、创建 Site、导出、清理数据 | 接口返回成功 |

## 文件变更

新增文件：

- `internal/server/router_test.go`
- `docs/deploy/2026-04-27-deploy-go-navbox.md`
- `docs/acceptance/2026-04-27-navbox-acceptance.md`
- `docs/delivery/2026-04-27-test-go-final-acceptance.md`

修改文件：

- `README.md`
- `docs/project/navbox/PROGRESS.md`

## 覆盖统计

已覆盖测试包：

- `internal/repo`
- `internal/server`
- `internal/service`
- `internal/storage`

新增测试：

- `TestRegisterWebRoutes`：覆盖 `/`、SPA fallback、API 404。

## 运行方式

```bash
go test ./...
npm --prefix web run build
go build ./...
docker compose build navbox
docker compose up -d
curl -sS http://localhost:8037/api/v1/health
```

## 风险与覆盖盲区

1. 未接入 Playwright 等浏览器自动化测试。
2. 未引入独立 PostgreSQL 集成测试框架。
3. 未覆盖高并发、压力和长时间运行场景。
4. 生产环境仍需替换默认 PostgreSQL 密码并按需启用 Cookie Secure。
