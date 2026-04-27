# Navbox 编码规范

## 一、通用规范

1. 回复、文档和代码注释说明使用简体中文。
2. 日志信息统一使用英文。
3. 只修改任务范围内的文件和代码。
4. 禁止自动格式化未修改的代码。
5. 敏感信息禁止写入代码、配置样例和日志。

## 二、Go 后端规范

### 2.1 技术栈

后端默认使用：

- Gin
- GORM
- PostgreSQL
- Zerolog
- Viper
- Uber Fx
- Validator

禁止在无明确理由时引入同类替代库。

### 2.2 分层结构

必须遵循：

```text
Handler -> Service -> Repository -> Model/DTO
```

约束：

1. `cmd/navbox/main.go` 只负责启动。
2. Handler 只做参数绑定、校验、响应封装。
3. Service 承载业务逻辑和事务。
4. Repository 只做数据访问。
5. DTO 和 Model 严格分离。
6. Handler 不能直接调用 Repository。
7. Service 不能直接写 HTTP 响应。

### 2.3 依赖注入

1. 使用 Uber Fx 注入 Handler、Service、Repository、配置、日志和数据库连接。
2. 禁止使用全局变量保存业务依赖。
3. 资源启动和释放通过 `fx.Lifecycle` 管理。

### 2.4 配置

1. 配置加载统一放在 `internal/config`。
2. 使用强类型配置模型。
3. 默认加载顺序：默认值 < 配置文件 < 环境变量。
4. 新增配置必须补默认值和校验逻辑。

### 2.5 日志

1. 使用 Zerolog。
2. 日志内容使用英文。
3. 错误日志必须带 `err` 字段。
4. 禁止打印密码、Session token、数据库连接串。
5. 初始 admin 密码只允许在首次初始化时打印一次。

### 2.6 API

1. API 路径使用 `/api/v1` 前缀。
2. 资源名使用复数。
3. 多单词路径使用连字符。
4. 所有响应使用统一格式：

```json
{
  "code": 200,
  "data": {},
  "message": "success"
}
```

5. 参数校验错误由 Handler 转换为友好提示。
6. 未知错误返回泛化提示，不能暴露内部实现细节。

### 2.7 数据库

1. 所有数据库操作必须带 `context.Context`。
2. 必须配置连接池参数。
3. 批量写入使用批量能力，禁止一次性无控制写入大量数据。
4. 批量删除、导入等高风险操作必须使用事务。
5. 禁止拼接 SQL。

### 2.8 认证与权限

1. admin 密码使用 bcrypt 存储。
2. Session token 只保存 hash。
3. admin 写接口必须通过 Session Middleware。
4. 游客接口和 admin 接口集中注册，避免零散放行逻辑。
5. 游客模式不能产生服务端写入。

### 2.9 文件与 icon

1. icon 文件只保存到配置的上传目录。
2. 禁止接受路径穿越文件名。
3. 文件名使用 hash 或 UUID。
4. 限制 MIME、扩展名和大小。
5. 导入 icon 时先写临时目录，确认成功后再提交。

### 2.10 导入导出

1. 导出包使用 zip。
2. zip 内包含 `navbox.json` 和 `icons/`。
3. 导入前必须校验结构。
4. 冲突时跳过并报告。
5. 导入失败不能产生半成功数据。

## 三、前端规范

### 3.1 技术栈

前端使用：

- React
- Vite
- TypeScript
- lucide-react

不引入 Ant Design、MUI 等大型 UI 框架，除非后续明确确认。

### 3.2 结构建议

```text
web/src/api/          API Client
web/src/components/   通用组件
web/src/features/     业务模块
web/src/hooks/        React Hooks
web/src/styles/       样式
web/src/types/        类型定义
```

### 3.3 UI 规则

1. 首页直接展示可用导航体验，不做营销落地页。
2. 游客界面不展示 admin 管理入口。
3. admin 管理入口只在登录后展示。
4. 卡片布局需要适配桌面和移动端。
5. Tag 较多时支持滚动或折叠。
6. 图标按钮优先使用 lucide-react。

### 3.4 状态与数据

1. 最近访问使用 `localStorage`。
2. 最近访问不调用服务端写接口。
3. API 调用统一封装。
4. Session 状态通过 `/api/v1/admin/session` 检查。

## 四、Docker 与部署规范

1. 使用多阶段 Docker 构建。
2. 前端先构建，Go 再 embed 构建产物。
3. Runtime 镜像只包含运行二进制和必要目录。
4. `./data/uploads` 需要挂载到容器内上传目录。
5. PostgreSQL 使用 Compose 服务和持久化 volume。

## 五、测试规范

1. 后端 Service 层优先写单元测试。
2. Repository 集成测试需要独立测试数据库。
3. Handler 测试覆盖认证、游客拒绝写接口、核心错误响应。
4. 前端至少保证 TypeScript 检查和构建通过。
5. Docker Compose 需要做启动冒烟验证。
