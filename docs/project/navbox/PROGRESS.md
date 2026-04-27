# Navbox 项目进度

## 当前状态

阶段：项目规划已完成，等待进入开发。

## 已确认决策

1. 后端使用 Go、Gin、GORM、PostgreSQL、Zerolog、Viper、Uber Fx。
2. 前端使用 React、Vite、TypeScript、lucide-react。
3. 前端构建产物通过 `go:embed` 嵌入后端。
4. 使用 Docker / Docker Compose 部署。
5. icon 文件存储在 `./data/uploads/`。
6. 游客只读，admin 负责管理。
7. admin 使用 Session 权限校验。
8. 首次启动无密码时生成 16 位随机初始密码，并打印英文日志。
9. admin 支持修改密码。
10. 最近访问存储在浏览器本地，不写服务端。
11. 导入导出使用 JSON/zip，支持 icon。
12. 导入冲突时跳过已有数据并报告冲突。

## 任务进度

| 编号 | 任务 | 状态 |
| --- | --- | --- |
| 01 | 项目骨架与 Docker 基础 | 未开始 |
| 02 | 数据模型、数据库连接与迁移 | 未开始 |
| 03 | admin 认证与 Session | 未开始 |
| 04 | Site / Tag 后端 API | 未开始 |
| 05 | icon 存储与访问 | 未开始 |
| 06 | 导入导出 | 未开始 |
| 07 | 前端首页与游客体验 | 未开始 |
| 08 | admin 前端管理能力 | 未开始 |
| 09 | 测试、验收与部署文档 | 未开始 |

## 下一步

从任务 01「项目骨架与 Docker 基础」开始开发。

## 风险与待处理

| 风险 | 状态 |
| --- | --- |
| 导入导出文件与数据库一致性 | 待实现时处理 |
| Session Cookie CSRF 防护 | 待实现时评估 |
| 上传文件安全限制 | 待实现时处理 |
| PostgreSQL 集成测试环境 | 待实现时处理 |
