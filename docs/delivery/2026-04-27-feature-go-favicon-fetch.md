# 任务交付：网站 icon 自动获取

## 任务背景

支持 admin 在新增和编辑网站时从网站地址获取 icon，同时保留手动上传 icon；游客首页增加进入 admin 的入口。

## 实现方案

1. 后端新增 admin 接口 `POST /api/v1/admin/icons/fetch`，请求体为 `{"url":"https://example.com"}`。
2. 新建网站时，如果未提供 icon，则后端按默认 URL 尝试自动获取 icon；获取失败不阻断网站创建。
3. icon 获取优先解析页面 `<link rel="...icon">`，失败后回退到 `/favicon.ico`。
4. 获取到的 icon 复用上传存储逻辑，保存到 `./data/uploads`，并按 SHA256 去重。
5. 前端网站表单增加“从网站获取”按钮，新增和编辑状态均可使用；原上传 icon 能力保留。
6. 游客首页侧边栏增加 Admin 入口。

## 文件变更

新增文件：

- `internal/service/favicon.go`
- `internal/service/favicon_test.go`
- `docs/delivery/2026-04-27-feature-go-favicon-fetch.md`

修改文件：

- `internal/storage/icon.go`
- `internal/service/icon.go`
- `internal/service/site.go`
- `internal/dto/icon.go`
- `internal/handler/icon.go`
- `web/src/api.ts`
- `web/src/AdminApp.tsx`
- `web/src/App.tsx`
- `web/src/styles.css`

## 配置与依赖变更

无新增配置项。

未新增第三方模块；HTML 解析使用项目已有的 `golang.org/x/net/html` 模块。

## 安全处理

1. 仅允许 `http` / `https` URL。
2. 拒绝带 UserInfo 的 URL。
3. 请求超时限制为 5 秒。
4. DNS 解析后只允许公网 IP，拒绝 loopback、private、link-local、metadata、CGNAT 等地址。
5. 下载内容继续复用 icon 大小和 MIME 校验。

## 测试结果

已执行并通过：

```bash
go test ./...
npm --prefix web run build
go build ./...
docker compose build navbox
docker compose up -d navbox
```

运行态冒烟：

1. `/` 返回嵌入式前端 HTML。
2. `/admin` 返回嵌入式前端 HTML。
3. admin 登录成功。
4. `POST /api/v1/admin/icons/fetch` 使用 `https://www.google.com` 获取 icon 成功。
5. 新建网站未填写 icon 时自动获取 icon 成功，随后已删除冒烟测试网站。

## 风险与后续建议

1. 部分网站只提供 SVG icon，当前 icon 存储仍保持现有图片白名单，可能无法自动保存。
2. 自动获取依赖目标网站可访问性，网络失败时会跳过。
3. 生产环境如需更严格控制，可增加允许域名白名单配置。
