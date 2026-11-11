# NexusOps Server

需要 Go 1.24+ 和 PostgreSQL 16+；当前集成测试使用 PostgreSQL 16。Server 启动时必须连接数据库，不能连接时退出。迁移是独立操作，Server 不会自动执行迁移或 GORM AutoMigrate。

## 数据库准备

使用 PostgreSQL 管理员在 `psql` 中创建专用角色和数据库，并以交互方式设置密码：

```sql
CREATE ROLE nexusops LOGIN;
\password nexusops
CREATE DATABASE nexusops OWNER nexusops;
```

为应用和迁移配置所需的最小权限。准备可信 CA 和服务端证书，生产环境使用 `DB_SSLMODE=verify-full`。本地隔离测试可使用 `disable`；不要在不可信网络上关闭 TLS。`.env.example` 只是变量清单，程序不会自动加载它；请用 shell、进程管理器或密钥管理系统注入变量，不要提交密码。示例的密码为空，必须设置。

| 环境变量 | 默认值及含义 |
| --- | --- |
| `NEXUSOPS_SERVER_ADDR` | `:8080`，HTTP 监听地址 |
| `NEXUSOPS_VERSION` | `0.1.0`，健康检查返回的版本 |
| `DB_HOST`、`DB_PORT` | `127.0.0.1`、`5432`，数据库地址和端口 |
| `DB_USER`、`DB_PASSWORD`、`DB_NAME` | 必填，数据库用户、密码和库名 |
| `DB_SSLMODE` | `verify-full`；支持 `disable`、`require`、`verify-ca`、`verify-full` |
| `DB_MAX_OPEN_CONNS`、`DB_MAX_IDLE_CONNS` | `10`、`5`；空闲数须在 0 与最大连接数之间 |
| `DB_CONN_MAX_LIFETIME` | `30m`，连接最大存活时间，Go duration 格式 |
| `DB_CONNECT_TIMEOUT` | `5s`，连接及健康检查超时，Go duration 格式 |

## 迁移与运行

在 `server/` 下设置上述环境变量后执行：

```sh
go run ./cmd/migrate status
go run ./cmd/migrate up
go run ./cmd/server
```

迁移文件随二进制嵌入。`up` 可重复运行，已应用的版本不会再次执行；`status` 显示每个版本状态。回滚最新版本使用 `go run ./cmd/migrate down`。`00002_auth` 的回滚会删除所有用户和会话；操作前必须备份。上线前检查每个迁移的回滚或前向修复方案；同一数据库的迁移命令须串行运行，不能在多个实例并发执行。

`GET /api/v1/health` 始终以 HTTP 200 和 `data.status: "ok"` 表示进程存活，同时在 `data.database.status` 返回 `ok` 或 `unavailable`；`GET /api/v1/ready` 在数据库不可用时返回 503。响应不包含数据库连接详情。生产环境应在可信代理处终止 TLS，并保护到 Server 的链路。

## 服务器资产与 Agent

Phase 3.0 增加了服务器资产和独立 Agent 通信。管理员先通过 `POST /api/v1/servers` 创建资产，再调用 `POST /api/v1/servers/{id}/agent/register-token` 获取短期一次性 Token。Agent 以 `Authorization: Agent <credential>` 调用 `/api/v1/agent/heartbeat` 和 `/api/v1/agent/report`；该凭据与用户 Bearer Token 完全分离，服务端只保存 SHA-256 摘要。Token 使用后不可重放，凭据可轮换或撤销。Agent 请求仅支持协议版本 `1`，并限制请求体大小；正式环境必须使用 HTTPS。心跳超时由 `AGENT_HEARTBEAT_TIMEOUT` 控制，注册 Token 有效期由 `AGENT_REGISTRATION_TTL` 控制。服务器禁用后心跳不会恢复其状态。

## 验证

执行 `go test ./...`、`go vet ./...` 和 `go build ./...`。真实数据库集成测试会修改 Goose 迁移状态，只能在**三个全新、专用、可丢弃**的数据库上运行：通过 `DB_NAME` 指定连接池测试库，以 `NEXUSOPS_AUTH_TEST_DB_NAME`、`NEXUSOPS_RBAC_TEST_DB_NAME` 分别指定认证和 RBAC 测试库，三者不可相同；设置 `NEXUSOPS_DB_INTEGRATION=1` 后执行 `go test -count=1 ./...`。数据库必须尚未应用迁移，测试结束后保留测试数据；再次运行须使用新的隔离数据库。未设置开关时集成测试会明确跳过。

## 首次管理员

先应用迁移。通过安全方式向进程环境设置 `NEXUSOPS_ADMIN_USER`、`NEXUSOPS_ADMIN_DISPLAY_NAME`、`NEXUSOPS_ADMIN_PASSWORD`，然后运行 `go run ./cmd/init-admin`。用户名为 3-64 位 ASCII 字母、数字、下划线、点或连字符，并规范化为小写；密码至少 12 字节。已有管理员时命令失败，且不提供公开注册接口。环境变量可能被同一主机的高权限用户或容器管理者读取，使用后应从当前环境移除，不要将密码写进命令历史、仓库或日志。Compose 调用方式见部署说明。

## 认证接口

- `POST /api/v1/auth/login`：JSON `username`、`password`；返回 Access 和 Refresh Token 及到期时间。
- `POST /api/v1/auth/refresh`：JSON `refresh_token`；原子轮换两个 Token，旧 Access Token 立即失效。重放已消费且未过期的 Refresh Token 将撤销整个会话，包括新 Token；并发刷新同一 Token 也会触发此保护。
- `GET /api/v1/auth/me`：`Authorization: Bearer <access_token>` 返回当前用户及有效的 `roles`、`permissions` 字符串数组，不包含密码哈希。原 `role` 字段仅为兼容保留，可能过时，不得用于授权。
- `POST /api/v1/auth/logout`：同样使用 Bearer Access Token，撤销当前会话及其刷新能力；Access Token 已过期时须先刷新。

Access Token 有效期 15 分钟；Refresh Token 随会话在首次登录后最多有效 7 天，刷新不延长会话。Access Token 每次请求都验证数据库会话和用户状态；认证失败使用统一响应，不泄露原因。响应 `Cache-Control: no-store`。登录限流使用 PostgreSQL 原子 UPSERT，在 `AUTH_RATE_LIMIT_WINDOW` 窗口内同时按来源 IP 和已存在账户计数，默认每 IP 20 次、每账户 10 次；未知或无效账户统一计入 `account:unknown`。限流表是所有 Server 实例共享的，存储不可用时返回 503，不绕过保护；触发限制返回 429。Token 通过 JSON 响应体和 Authorization Header 传输，不使用 Cookie、浏览器自动凭证或 CORS；浏览器客户端应只在内存保管两个 Token，禁止写入 localStorage、日志或 URL，页面刷新后需重新登录。必须使用 HTTPS，保护反向代理到 Server 的链路；XSS 仍可窃取内存中的 Token。刷新响应丢失时客户端可能需要重新登录；不应自动并发刷新或重试已消费的 Token。

认证审计写入 `auth_audit_events`，记录结果、来源 IP、截断后的 User-Agent、用户/会话 ID和结构化元数据，不记录密码、Token 或 Authorization Header。只有可信代理网段（`NEXUSOPS_TRUSTED_PROXY_CIDRS`）连接时才解析 `X-Forwarded-For`/`X-Real-IP`；其他请求使用直接连接地址。审计写入失败记录服务端错误日志，认证主流程继续完成，避免审计存储故障造成不可控登录中断；生产环境应监控该错误。后台清理每 `AUTH_CLEANUP_INTERVAL` 运行一次，按 100 条批次删除过期 Token、过期或超过 `AUTH_SESSION_RETENTION` 的撤销/失效会话，以及超过 `AUTH_AUDIT_RETENTION` 的审计记录。清理失败只记录日志，不影响认证；多个实例可重复执行，删除条件安全幂等。

`GET /api/v1/auth/sessions` 返回当前用户未撤销且未过期的会话 ID、创建时间、有效期和 `current` 标记，不返回任何 Token 或摘要。`DELETE /api/v1/auth/sessions/{id}` 只能撤销当前用户的其他会话；当前会话使用 logout。`DELETE /api/v1/auth/sessions` 撤销当前用户的其他全部会话。撤销写入审计，数据库认证查询立即失效旧 Access Token。

## RBAC 接口

以下接口均需 Bearer 认证以及指定权限，缺少认证返回 401，无权限返回 403；授权查询失败返回 503。

| 接口 | 权限 | 说明 |
| --- | --- | --- |
| `GET /api/v1/roles` | `system:role:read` | 列出角色，包括禁用状态 |
| `GET /api/v1/permissions` | `system:permission:read` | 列出权限 |
| `GET /api/v1/users/{id}/roles` | `system:user:read` | 查询用户角色 |
| `PUT /api/v1/users/{id}/roles` | `system:role:manage` | 用 `{"role_ids":[1,2]}` 原子替换角色；空数组撤销全部 |

角色 ID 必须存在且处于启用状态，重复 ID 或不存在、禁用的角色返回 400，不存在的用户返回 404。分配操作写入数据库审计；没有角色或角色被禁用时不会获得权限。每次授权均查询数据库，已提交的撤权会影响后续检查；已经通过检查的请求仍可能继续执行。`system:role:manage` 允许分配任意角色，因此相当于可提升权限，初始只授予 `super_admin`。初始化管理员命令会关联 `super_admin`；旧 `users.role` 不随角色分配更新。事务层禁止撤销或替换最后一名有效 `super_admin` 的管理角色，并使用数据库 advisory lock 串行化并发变更。尚未提供角色 CRUD、资源级权限或审计查询接口。

在新增路由时依次组合认证和授权：

```go
group.GET("/example", auth.RequireAuthentication(authService), rbac.RequirePermission(rbacService, rbac.PermissionRoleRead), handler)
```

## 系统管理接口

系统管理 API 只使用现有认证和 RBAC Middleware。用户列表 `GET /api/v1/users` 支持 `page`、`page_size`、`q` 和 `status`；用户详情、创建、更新、状态切换和密码重置分别使用 `/users/{id}` 资源。禁用用户或重置密码会撤销其所有 Session。用户接口始终返回脱敏用户视图。

角色接口扩展了角色详情、创建、修改、状态切换、删除和角色权限批量设置。系统内置角色带 `is_system` 标志，不能修改或删除；仍被用户使用的自定义角色不能删除。权限定义仍由数据库维护，批量授权在事务中执行并即时生效。

审计查询包括 `GET /api/v1/audit/auth` 和 `GET /api/v1/audit/roles`，按时间倒序分页；可按事件类型、结果和用户筛选。审计只读，接口不提供删除或修改。跨用户 Session 查询和撤销使用 `/api/v1/admin/sessions` 及 `/api/v1/admin/users/{id}/sessions`，分别需要 `system:session:read` 和 `system:session:manage`，响应不含 Token。系统配置接口 `GET/PUT /api/v1/system/config` 只返回非敏感项，部署级配置仍只能通过环境变量设置。

新增权限包括 `system:role:create/update/delete`、`system:permission:manage`、`system:audit:read`、`system:session:read/manage` 和 `system:config:read/update`。角色、用户、审计、Session 和配置管理操作写入认证审计；所有列表接口限制页大小为 100。当前尚未提供用户删除、审计全文关键词查询、配置项动态注册或资源级权限。

Agent 安全配置：`AGENT_HEARTBEAT_TIMEOUT`（默认 2m）、`AGENT_REGISTRATION_TTL`（默认 15m）、`AGENT_RATE_LIMIT_WINDOW`（默认 1m）、`AGENT_REGISTER_RATE_LIMIT`（默认每 IP 10 次）、`AGENT_HEARTBEAT_RATE_LIMIT`（默认每 Agent 120 次）和 `AGENT_REPORT_RATE_LIMIT`（默认每 Agent 30 次）。限流状态存储在 PostgreSQL，所有 Server 实例共享。

指标配置：`AGENT_METRICS_RATE_LIMIT`（默认每 Agent 60 次/窗口）、`METRICS_CLEANUP_INTERVAL`（默认 1h）和 `METRICS_RAW_RETENTION`（默认 168h）。当前实现保存原始指标，5 分钟和 1 小时聚合留待后续阶段。

Phase 3.1.1 提供 Agent 指标上报 `POST /api/v1/agent/metrics`，以及管理员查询 `/api/v1/servers/{id}/metrics/latest` 和 `/metrics/history`。当前历史查询只支持 `step=raw`，原始指标默认保留 7 天；聚合数据和监控 Web 页面留待后续阶段。
