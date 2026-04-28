# Navbox 部署文档

## 基本信息

- 功能/版本：Navbox MVP
- 发布日期：2026-04-27
- 涉及服务：`navbox`、`postgres`
- 部署环境：Docker Compose

## 一、环境变量变更

| 变量名 | 用途 | 示例值 | 是否敏感 | 影响服务 |
| --- | --- | --- | :---: | --- |
| `NAVBOX_HTTP_ADDR` | HTTP 监听地址 | `:8037` | 否 | navbox |
| `NAVBOX_DATABASE_DSN` | PostgreSQL 连接串 | `host=postgres user=navbox password=navbox dbname=navbox port=5432 sslmode=disable TimeZone=UTC` | 是 | navbox |
| `NAVBOX_UPLOAD_DIR` | icon 上传目录 | `/app/data/uploads` | 否 | navbox |
| `NAVBOX_AUTH_SESSION_TTL` | admin Session 有效期 | `24h` | 否 | navbox |
| `NAVBOX_RESTORE_MODE` | admin 密码恢复模式，临时设置为 `admin-password` | `admin-password` | 否 | navbox |
| `NAVBOX_RESTORE_TOKEN` | admin 密码恢复一次性 Token，启动后 5 分钟内有效 | `openssl rand -hex 32` | 是 | navbox |
| `NAVBOX_ICON_FETCH_ALLOWED_PRIVATE_CIDRS` | 允许获取 icon 的内网 CIDR 白名单 | `10.0.0.0/8,172.16.0.0/12,192.168.0.0/16` | 否 | navbox |
| `POSTGRES_DB` | PostgreSQL 数据库名 | `navbox` | 否 | postgres |
| `POSTGRES_USER` | PostgreSQL 用户名 | `navbox` | 否 | postgres |
| `POSTGRES_PASSWORD` | PostgreSQL 密码 | `navbox` | 是 | postgres |

## 二、配置项变更

| 配置路径 | 默认值/当前值 | 说明 | 是否需按环境调整 |
| --- | --- | --- | :---: |
| `http.addr` | `:8037` | HTTP 监听地址 | 是 |
| `http.read_header_timeout` | `5s` | 请求头读取超时 | 否 |
| `http.shutdown_timeout` | `10s` | 优雅关闭超时 | 否 |
| `database.dsn` | 空字符串 | 数据库连接串，启动必填 | 是 |
| `database.max_open_conns` | `20` | 最大连接数 | 是 |
| `database.max_idle_conns` | `5` | 最大空闲连接数 | 是 |
| `database.conn_max_lifetime` | `1h` | 连接最大生命周期 | 是 |
| `auth.session_ttl` | `24h` | admin Session 有效期 | 是 |
| `auth.cookie_name` | `navbox_admin_session` | Session Cookie 名称 | 否 |
| `auth.cookie_secure` | `false` | Cookie Secure 标记 | 生产 HTTPS 建议开启 |
| `auth.initial_password_length` | `16` | 初始密码长度 | 否 |
| `restore.mode` | 空字符串 | admin 密码恢复模式，忘记密码时临时设置为 `admin-password` | 否 |
| `restore.token` | 空字符串 | admin 密码恢复一次性 Token，启动后 5 分钟内有效 | 是 |
| `upload.dir` | `./data/uploads` | icon 文件目录 | 是 |
| `upload.max_bytes` | `1048576` | 单个 icon 最大字节数 | 是 |
| `icon_fetch.allowed_private_cidrs` | 空字符串 | 允许获取 icon 的内网 CIDR 白名单 | 是 |

## 三、依赖与中间件变更

| 类型 | 名称 | 变更说明 |
| --- | --- | --- |
| 数据库 | PostgreSQL 17 | 存储 Site、Tag、Session、icon 元数据 |
| 文件系统 | `./data/uploads` | 存储上传和导入的 icon 文件 |
| 前端构建 | React + Vite | 构建产物嵌入 Go 二进制 |
| 后端服务 | Go + Gin | 提供 API、静态资源和上传文件访问 |

## 四、数据库/迁移变更

| 变更类型 | 对象 | 说明 | 是否可回滚 |
| :---: | --- | --- | :---: |
| AutoMigrate | `sites` | 网站数据 | 是 |
| AutoMigrate | `tags` | Tag 数据 | 是 |
| AutoMigrate | `site_tags` | 网站与 Tag 关联 | 是 |
| AutoMigrate | `icons` | icon 元数据 | 是 |
| AutoMigrate | `admin_settings` | admin 密码配置 | 是 |
| AutoMigrate | `admin_sessions` | admin Session | 是 |
| Extension | `pgcrypto` | 生成 UUID 所需 | 需人工评估 |

## 五、部署步骤

1. 准备 Docker 和 Docker Compose。
2. 修改 `docker-compose.yml` 中的 PostgreSQL 密码和 `NAVBOX_DATABASE_DSN`。
3. 确认上传目录可写：

```bash
mkdir -p data/uploads
```

4. 构建镜像：

```bash
docker compose build navbox
```

5. 启动服务：

```bash
docker compose up -d
```

6. 查看日志并记录首次启动 admin 密码：

```bash
docker compose logs -f navbox
```

7. 打开 `http://localhost:8037/admin` 登录，并修改 admin 密码。

### admin 密码恢复

忘记 admin 密码时，临时启用恢复模式：

```bash
RESTORE_TOKEN="$(openssl rand -hex 32)"
NAVBOX_RESTORE_MODE=admin-password NAVBOX_RESTORE_TOKEN="${RESTORE_TOKEN}" docker compose up -d
echo "Restore Token: ${RESTORE_TOKEN}"
```

访问 `http://localhost:8037/admin`，输入 Restore Token 和新密码完成重置。重置成功后删除 `NAVBOX_RESTORE_MODE`、`NAVBOX_RESTORE_TOKEN` 并重启服务。

## 六、验证清单

- [ ] `docker compose ps` 显示 `postgres` healthy。
- [ ] `docker compose ps` 显示 `navbox` running。
- [ ] `curl -sS http://localhost:8037/api/v1/health` 返回 `status=ok`。
- [ ] `http://localhost:8037/` 可访问游客首页。
- [ ] `http://localhost:8037/admin` 可访问 admin 页面。
- [ ] admin 可登录、修改密码、创建 Tag、创建网站。
- [ ] icon 上传后 `/uploads/<file>` 可公开访问。
- [ ] 导出 zip 包包含 `navbox.json`。
- [ ] 导入 zip 返回导入报告。

## 七、回滚方案

1. 停止当前服务：

```bash
docker compose down
```

2. 回退镜像或代码版本。
3. 如需回退数据库，恢复发布前 PostgreSQL 备份。
4. 如需回退 icon 文件，恢复发布前 `./data/uploads` 备份。
5. 重新启动服务并执行验证清单。

## 八、注意事项

1. 生产环境必须修改 `POSTGRES_PASSWORD` 和 `NAVBOX_DATABASE_DSN` 中的密码。
2. HTTPS 场景建议设置 `auth.cookie_secure=true`。
3. `./data/uploads` 与 PostgreSQL 数据需要一起备份。
4. 导入不会覆盖已有数据，冲突项会跳过。
5. 当前使用 GORM AutoMigrate，长期生产演进建议引入版本化 migration。
