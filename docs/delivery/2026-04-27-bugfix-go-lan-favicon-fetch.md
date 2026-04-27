# Bug 修复：LAN 网站 icon 获取失败

## 问题背景

`Open Speedtest` 的网站地址为 `http://10.1.1.99:3000/`，admin 页面从网站获取 icon 时返回未找到。

## 根因分析

Open Speedtest 页面中存在可用 icon：

- `assets/images/icons/apple-touch-icon.png`
- `assets/images/icons/favicon-32x32.png`
- `assets/images/icons/favicon-16x16.png`

失败原因不是目标站点缺少 icon，而是后端 icon 抓取的 SSRF 防护默认拒绝私网 IP。`10.1.1.99` 属于 RFC1918 内网地址，因此请求在 Dial 阶段被拦截。

## 修复方案

1. 新增 `icon_fetch.allowed_private_cidrs` 配置项。
2. Docker Compose 默认允许常见 LAN 网段：
   - `10.0.0.0/8`
   - `172.16.0.0/12`
   - `192.168.0.0/16`
3. icon 抓取仍然拒绝 loopback、link-local、metadata、multicast、unspecified 和未白名单私网地址。
4. 补充单元测试覆盖私网默认拒绝和 CIDR 白名单允许。

## 文件变更

- `config/config.toml`
- `docker-compose.yml`
- `internal/config/config.go`
- `internal/service/icon.go`
- `internal/service/favicon.go`
- `internal/service/favicon_test.go`
- `README.md`
- `docs/deploy/2026-04-27-deploy-go-navbox.md`

## 验证结果

已执行：

```bash
go test ./...
go build ./...
docker compose build navbox
docker compose up -d navbox
```

Open Speedtest 验证：

1. `POST /api/v1/admin/icons/fetch` 传入 `http://10.1.1.99:3000/` 返回 200。
2. 返回的 icon 文件为 `/uploads/7e02e40c7b6e825b2bdc00f07dee7d00383d5384c3cde6058217e06ff95313ec.png`。
3. 已将 `Open Speedtest` 的 `icon_type` 更新为 `image`，`icon_value` 更新为上述上传文件路径。
4. `GET /api/v1/sites?search=Open%20Speedtest` 确认可读列表返回上述 icon。
5. `GET /uploads/7e02e40c7b6e825b2bdc00f07dee7d00383d5384c3cde6058217e06ff95313ec.png` 返回 200，`Content-Type` 为 `image/png`。

## 风险与后续建议

1. LAN 获取 icon 属于服务端主动访问内网资源，必须通过 CIDR 白名单控制。
2. 不建议在公网部署时放开过宽的内网 CIDR。
3. metadata 地址和 loopback 地址仍保持拒绝。
