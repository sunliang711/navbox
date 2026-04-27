# 任务 01：项目骨架与 Docker 基础

## 目标

建立 Go 后端、React 前端和 Docker 部署基础，使项目具备可运行的最小骨架。

## 技术方案

1. 初始化 Go module。
2. 创建 `cmd/navbox/main.go` 启动入口。
3. 创建 Go 分层目录：`internal/config`、`internal/handler`、`internal/service`、`internal/repo`、`internal/model`、`internal/dto`、`internal/middleware`。
4. 初始化 React + Vite + TypeScript 前端目录 `web/`。
5. 配置 `go:embed` 静态资源目录。
6. 添加 `Dockerfile` 和 `docker-compose.yml`。
7. 提供基础健康检查接口。

## 验收标准

1. `go build ./...` 可以通过。
2. 前端可以执行构建命令。
3. Docker Compose 可以启动 PostgreSQL 和 navbox 服务。
4. 访问健康检查接口返回成功。

## 实现步骤

1. 初始化 `go.mod`。
2. 引入 Gin、Zerolog、Viper、Fx 等基础依赖。
3. 创建配置模型和默认配置。
4. 创建 Gin Server 启动和优雅关停。
5. 创建前端 Vite 项目结构。
6. 编写 Docker 多阶段构建。
7. 编写 Compose 配置和默认环境变量。

## 前置依赖

无。

## 风险

1. 前端构建产物在开发初期可能不存在，需要 Go embed 使用占位文件。
2. Docker 构建需要同时处理 Node 和 Go 环境。
