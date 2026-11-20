# Navbox

Navbox 是一个自托管导航站。后端使用 Go + Gin + GORM + PostgreSQL，前端使用 React + Vite，前端构建产物通过 `go:embed` 嵌入后端。系统支持游客只读访问和 admin 管理访问。

## 功能

- 网站卡片展示、搜索、Tag 多选筛选。
- 默认 Tag、常用、最近访问、未分类视图。
- 最近访问仅保存在浏览器 `localStorage`。
- admin 登录、退出、修改密码。
- 网站和 Tag 的新增、编辑、删除、排序。
- 网站批量删除、批量添加 Tag、批量移除 Tag。
- icon 上传，文件保存到 `./data/uploads/`。
- 配置 zip 导入导出，包含 `navbox.json` 和 `icons/`。
- 定时任务调度：支持在线配置 Cron 周期、预览未来触发时间、手动立即执行、查看执行历史，内置 **SafeKeep 全量数据定时备份**（默认每周一 04:00 导出全部数据并通过 SafeKeep API 上传备份，可自定义周期、路径、目标上游与同步/异步模式）、过期审计日志清理与任务记录清理。
- 审计日志：记录管理员登录/改密、网站与 Tag 变更、导入导出、定时任务执行等操作轨迹，支持多条件筛选、查看 JSON 详情及 CSV / JSON 导出。

## 快速启动

使用 Docker Hub 镜像一键部署：

```bash
mkdir -p navbox/data/uploads && cd navbox
cat > docker-compose.yml <<'EOF'
services:
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_DB: navbox
      POSTGRES_USER: navbox
      POSTGRES_PASSWORD: navbox
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U navbox -d navbox"]
      interval: 10s
      timeout: 5s
      retries: 5

  navbox:
    image: sunliang711/navbox:latest
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      NAVBOX_HTTP_ADDR: ":8037"
      NAVBOX_DATABASE_DSN: "host=postgres user=navbox password=navbox dbname=navbox port=5432 sslmode=disable TimeZone=UTC"
      NAVBOX_UPLOAD_DIR: "/app/data/uploads"
      NAVBOX_AUTH_SESSION_TTL: "168h"
      NAVBOX_ICON_FETCH_ALLOWED_PRIVATE_CIDRS: "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"
      NAVBOX_ICON_FETCH_SKIP_TLS_VERIFY: ${NAVBOX_ICON_FETCH_SKIP_TLS_VERIFY:-false}
    ports:
      - "8037:8037"
    volumes:
      - ./data/uploads:/app/data/uploads

volumes:
  postgres_data:
EOF
docker compose up -d
docker compose logs -f navbox
```

在源码目录本地构建并启动：

```bash
docker compose up -d
docker compose logs -f navbox
```

首次启动且数据库中没有 admin 密码时，日志会输出初始密码，日志消息为：

```text
Admin initial password generated
```

日志字段 `password` 即初始密码。登录后请尽快在 admin 页面修改密码。

访问地址：

- 游客首页：`http://localhost:8037/`
- admin 管理：`http://localhost:8037/admin`
- 健康检查：`http://localhost:8037/api/v1/health`

## 本地开发

后端：

```bash
go test ./...
go run ./cmd/navbox
```

前端：

```bash
npm --prefix web install
npm --prefix web run dev
```

构建：

```bash
npm --prefix web run build
go build ./...
docker compose build navbox
```

## 主要配置

配置文件示例位于 `config/config.toml`，Docker 部署主要使用环境变量覆盖。

| 环境变量 | 说明 | 示例 |
| --- | --- | --- |
| `NAVBOX_HTTP_ADDR` | HTTP 监听地址 | `:8037` |
| `NAVBOX_DATABASE_DSN` | PostgreSQL DSN | `host=postgres user=navbox password=navbox dbname=navbox port=5432 sslmode=disable TimeZone=UTC` |
| `NAVBOX_UPLOAD_DIR` | icon 上传目录 | `/app/data/uploads` |
| `NAVBOX_AUTH_SESSION_TTL` | admin Session 有效期（默认 7 天） | `168h` |
| `NAVBOX_RESTORE_MODE` | 忘记 admin 密码时临时设置为 `admin-password` | `admin-password` |
| `NAVBOX_RESTORE_TOKEN` | 恢复模式使用的一次性 Token，启动后 5 分钟内有效 | `openssl rand -hex 32` |
| `NAVBOX_ICON_FETCH_ALLOWED_PRIVATE_CIDRS` | 允许获取 icon 的内网 CIDR 白名单 | `10.0.0.0/8,172.16.0.0/12,192.168.0.0/16` |
| `NAVBOX_ICON_FETCH_SKIP_TLS_VERIFY` | 获取 icon 时是否跳过 HTTPS 证书校验，内网自签证书站点可临时开启 | `false` |
| `NAVBOX_SECURITY_MASTER_KEY` | 可选 32 字节加密主密钥（64 位十六进制或 Base64），用于加密保存 SafeKeep API Key；留空时自动生成并持久化到数据库 | `openssl rand -hex 32` |

运行时目录：

```text
./data/uploads:/app/data/uploads
postgres_data:/var/lib/postgresql/data
```

## 导入导出

admin 页面支持导出全部、导出选中网站、导出选中 Tag。导出文件为 zip 包：

```text
navbox.json
icons/
```

导入时遇到冲突数据会跳过并返回报告，不覆盖已有数据。

## admin 密码恢复

忘记 admin 密码时，可以临时启用恢复模式：

```bash
RESTORE_TOKEN="$(openssl rand -hex 32)"
NAVBOX_RESTORE_MODE=admin-password NAVBOX_RESTORE_TOKEN="${RESTORE_TOKEN}" ./navbox
```

Docker Compose 场景可把 `NAVBOX_RESTORE_MODE=admin-password` 和 `NAVBOX_RESTORE_TOKEN=<token>` 临时加入 `navbox` 服务环境变量后重启。然后访问 `/admin`，页面会切换为密码恢复表单。重置成功后删除这两个恢复配置并重启服务。

## 验证命令

```bash
go test ./...
npm --prefix web run build
go build ./...
docker compose build navbox
docker compose up -d
curl -sS http://localhost:8037/api/v1/health
```

## 文档

- 需求文档：`docs/tag-navigation-requirements.md`
- 架构方案：`docs/project/navbox/architecture.md`
- 项目进度：`docs/project/navbox/PROGRESS.md`
- 部署文档：`docs/deploy/2026-04-27-deploy-go-navbox.md`
- 验收文档：`docs/acceptance/2026-04-27-navbox-acceptance.md`
