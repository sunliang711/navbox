# Navbox 验收文档

## 验收范围

本次验收覆盖 Navbox MVP：

1. Docker Compose 部署。
2. PostgreSQL 数据持久化。
3. admin 初始密码、登录、Session、修改密码。
4. 游客首页。
5. Site / Tag 管理。
6. icon 上传和访问。
7. 导入导出。

## 验收命令

```bash
go test ./...
npm --prefix web run build
go build ./...
docker compose build navbox
docker compose up -d
curl -sS http://localhost:8080/api/v1/health
```

## 自动化验证结果

| 项目 | 命令 | 结果 |
| --- | --- | --- |
| Go 测试 | `go test ./...` | 通过 |
| Go 构建 | `go build ./...` | 通过 |
| 前端构建 | `npm --prefix web run build` | 通过 |
| Docker 构建 | `docker compose build navbox` | 通过 |
| Docker 启动 | `docker compose up -d` | 通过 |
| 健康检查 | `curl /api/v1/health` | 通过 |
| 首次启动密码日志 | 独立空数据库 Compose project | 通过 |

## 功能验收清单

### 游客

- [x] 可访问 `/`。
- [x] 可查看网站卡片。
- [x] 可查看 Tag 列表。
- [x] 支持默认 Tag 视图。
- [x] 支持搜索。
- [x] 支持多 Tag 筛选。
- [x] 支持常用、最近、未分类视图。
- [x] 最近访问保存在浏览器本地。
- [x] 游客首页不展示 admin 管理入口。

### admin

- [x] 可访问 `/admin`。
- [x] 未登录访问 admin Session 返回 401。
- [x] 可登录并创建 Session。
- [x] 可退出登录。
- [x] 可修改密码。
- [x] 可新增、编辑、删除网站。
- [x] 可新增、编辑、删除 Tag。
- [x] 可设置默认 Tag。
- [x] 可上传 icon。
- [x] 可批量删除网站。
- [x] 可批量添加 Tag。
- [x] 可批量移除 Tag。
- [x] 可调整网站排序。
- [x] 可调整 Tag 排序。
- [x] 可导出全部、指定网站、指定 Tag。
- [x] 可导入 zip，并返回报告。

## 冒烟测试记录

1. 使用默认 Docker project 验证 `/`、`/admin` 和 `/api/v1/health`。
2. 使用 admin Session 验证 Tag 创建、Site 创建、导出、删除 Site、删除 Tag。
3. 使用独立 `navbox-task09` Compose project 和空 PostgreSQL volume 验证首次启动日志输出初始 admin 密码。
4. 独立验证完成后已清理 `navbox-task09` 项目和临时 volume。

## 覆盖盲区

1. 未接入浏览器自动化截图测试。
2. 未引入真实 PostgreSQL 集成测试框架；数据库行为通过 Docker 冒烟测试验证。
3. 未覆盖高并发上传、超大导入包压力测试。
4. 未覆盖 HTTPS Cookie Secure 场景。

## 验收结论

Navbox MVP 已满足需求文档和任务拆分中的核心验收标准，可进入部署试用阶段。
