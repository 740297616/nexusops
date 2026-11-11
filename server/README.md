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

## 验证

执行 `go test ./...`、`go vet ./...` 和 `go build ./...`。真实数据库集成测试会修改 Goose 迁移状态，只能在**三个全新、专用、可丢弃**的数据库上运行：通过 `DB_NAME` 指定连接池测试库，以 `NEXUSOPS_AUTH_TEST_DB_NAME`、`NEXUSOPS_RBAC_TEST_DB_NAME` 分别指定认证和 RBAC 测试库，三者不可相同；设置 `NEXUSOPS_DB_INTEGRATION=1` 后执行 `go test -count=1 ./...`。数据库必须尚未应用迁移，测试结束后保留测试数据；再次运行须使用新的隔离数据库。未设置开关时集成测试会明确跳过。

## 首次管理员

先应用迁移。通过安全方式向进程环境设置 `NEXUSOPS_ADMIN_USER`、`NEXUSOPS_ADMIN_DISPLAY_NAME`、`NEXUSOPS_ADMIN_PASSWORD`，然后运行 `go run ./cmd/init-admin`。用户名为 3-64 位 ASCII 字母、数字、下划线、点或连字符，并规范化为小写；密码至少 12 字节。已有管理员时命令失败，且不提供公开注册接口。环境变量可能被同一主机的高权限用户或容器管理者读取，使用后应从当前环境移除，不要将密码写进命令历史、仓库或日志。Compose 调用方式见部署说明。

## 认证接口

- `POST /api/v1/auth/login`：JSON `username`、`password`；返回 Access 和 Refresh Token 及到期时间。
- `POST /api/v1/auth/refresh`：JSON `refresh_token`；原子轮换两个 Token，旧 Access Token 立即失效。重放已消费且未过期的 Refresh Token 将撤销整个会话，包括新 Token；并发刷新同一 Token 也会触发此保护。
- `GET /api/v1/auth/me`：`Authorization: Bearer <access_token>` 返回当前用户及有效的 `roles`、`permissions` 字符串数组，不包含密码哈希。原 `role` 字段仅为兼容保留，可能过时，不得用于授权。
- `POST /api/v1/auth/logout`：同样使用 Bearer Access Token，撤销当前会话及其刷新能力；Access Token 已过期时须先刷新。

Access Token 有效期 15 分钟；Refresh Token 随会话在首次登录后最多有效 7 天，刷新不延长会话。Access Token 每次请求都验证数据库会话和用户状态；认证失败使用统一响应，不泄露原因。响应 `Cache-Control: no-store`。登录全局限速为每个 Server 进程每分钟 20 次，重启重置，多实例不共享；这是基础保护，正式部署应在可信入口增加集中限流及监控。Token 通过 JSON 响应体和 Authorization Header 传输，不使用 Cookie、浏览器自动凭证或 CORS；浏览器客户端应只在内存保管两个 Token，禁止写入 localStorage、日志或 URL，页面刷新后需重新登录。必须使用 HTTPS，保护反向代理到 Server 的链路；XSS 仍可窃取内存中的 Token。刷新响应丢失时客户端可能需要重新登录；不应自动并发刷新或重试已消费的 Token。当前没有认证审计记录或会话清理任务。

## RBAC 接口

以下接口均需 Bearer 认证以及指定权限，缺少认证返回 401，无权限返回 403；授权查询失败返回 503。

| 接口 | 权限 | 说明 |
| --- | --- | --- |
| `GET /api/v1/roles` | `system:role:read` | 列出角色，包括禁用状态 |
| `GET /api/v1/permissions` | `system:permission:read` | 列出权限 |
| `GET /api/v1/users/{id}/roles` | `system:user:read` | 查询用户角色 |
| `PUT /api/v1/users/{id}/roles` | `system:role:manage` | 用 `{"role_ids":[1,2]}` 原子替换角色；空数组撤销全部 |

角色 ID 必须存在且处于启用状态，重复 ID 或不存在、禁用的角色返回 400，不存在的用户返回 404。分配操作写入数据库审计；没有角色或角色被禁用时不会获得权限。每次授权均查询数据库，已提交的撤权会影响后续检查；已经通过检查的请求仍可能继续执行。`system:role:manage` 允许分配任意角色，因此相当于可提升权限，初始只授予 `super_admin`。初始化管理员命令会关联 `super_admin`；旧 `users.role` 不随角色分配更新。当前未阻止撤销最后一名有角色管理权限的用户，应在操作前确认另有管理员。尚未提供角色 CRUD、资源级权限或审计查询接口。

在新增路由时依次组合认证和授权：

```go
group.GET("/example", auth.RequireAuthentication(authService), rbac.RequirePermission(rbacService, rbac.PermissionRoleRead), handler)
```
