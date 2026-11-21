# 网站启用 / 禁用功能交付

## 使用方式

后台「网站」→「编辑网站」→ 切换「启用网站」→「保存网站」。新建网站默认启用，后台列表在网站名称旁显示「启用 / 禁用」，手机卡片也显示状态。

禁用的网站保留在后台，可继续编辑、导出及重新启用；首页的全部网站、常用工具、搜索、Tag 筛选与最近访问均不显示。已打开的首页刷新或重新进入对应视图后读取最新状态。

## 影响范围与接口

- `internal/model/site.go`、`internal/dto/site.go`：新增网站 `is_enabled`。
- `internal/repo/site.go`、`internal/service/site.go`、`internal/handler/site.go`：持久化状态、区分公开和后台列表。
- `internal/dto/import_export.go`、`internal/service/import_export.go`：备份包含并恢复状态。
- `web/src/AdminApp.tsx`、`api.ts`、`types.ts`、`preferences.tsx`、`styles.css`：编辑开关、中英文提示、桌面和手机状态标记。
- `web/src/App.tsx`：最近访问缓存与最新公开网站列表按 ID 取交集，使用最新网站内容，保留访问顺序和时间。
- `internal/web/dist`：同步生产构建资源，版本号 `v0.0.36`。

| 方法 | 路径 | 行为 |
| --- | --- | --- |
| GET | `/api/v1/sites` | 仅返回启用网站，搜索、收藏及 Tag 等条件继续叠加；客户端参数无法包含禁用网站 |
| GET | `/api/v1/admin/sites` | 新增受现有 Session 鉴权保护的完整网站列表，包含禁用项 |
| POST | `/api/v1/admin/sites` | 请求新增可选 `is_enabled`，省略时默认 `true` |
| PUT | `/api/v1/admin/sites/:id` | 请求新增可选 `is_enabled`，显式 `false` 可禁用，省略或 `null` 保留原状态 |

网站响应包含布尔字段 `is_enabled`，更新沿用原有管理员审计并记录状态。列表过滤由数据库执行，首页不依赖前端隐藏来隔离禁用项。

## 数据迁移与兼容

- 服务启动时沿用 GORM AutoMigrate，为 `sites` 添加 `is_enabled boolean NOT NULL DEFAULT true` 及普通索引；已有记录默认启用。
- Model 使用 `*bool`，确保创建和批量导入时的显式 `false` 不会被 GORM 的 `default:true` 替换。
- 旧客户端更新省略状态时，更新 SQL 不包含该列，避免意外重新启用及读取后覆盖的并发窗口。
- 备份版本保持 1，新增可选字段；旧备份缺失字段默认启用，新备份保留禁用状态。
- 不新增配置、环境变量或依赖。回退旧版本会忽略状态字段，不能继续保证禁用项隐藏；备份应使用本版本恢复以保留该状态。

## 验证结果

- `go test ./...` 通过。
- `VITE_NAVBOX_VERSION=v0.0.36 npm run build` 通过。
- `go build -o work/icon-design/navbox-check ./cmd/navbox` 通过。
- `git diff --check` 通过。
- Go 测试覆盖：创建默认启用及显式禁用、更新省略状态保留原值、启用与禁用的 SQL 参数、导入禁用项、旧备份和新备份的状态往返、公开列表的组合过滤、Service 过滤传递、公开请求无法绕过状态限制、后台列表登录保护。
- Tabbit 隔离页面使用内存 API 夹具验证：编辑取消启用并保存、后台仍显示禁用项、首页及常用工具隐藏、旧最近访问缓存隐藏禁用项、最近访问搜索、手机端重新启用、最近访问刷新后恢复；检查两套布局及明暗主题切换。
- 浏览器测试没有修改线上业务数据；临时页面及服务在验证后关闭。
- 本轮未连接真实 PostgreSQL 执行迁移，发布后由现有启动迁移机制添加字段。

## ChangeLog

### v0.0.36 - 2026-11-21

- Added：网站启用 / 禁用开关及后台状态标记。
- Changed：首页接口只返回启用网站，后台使用独立鉴权列表。
- Fixed：最近访问旧缓存不再继续显示已禁用或删除的网站。
- Compatibility：现有网站及旧备份默认启用，导入导出保留网站状态。
