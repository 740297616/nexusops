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

迁移文件随二进制嵌入。`up` 可重复运行，已应用的版本不会再次执行；`status` 显示每个版本状态。回滚最新版本使用 `go run ./cmd/migrate down`。初始迁移只执行 `SELECT 1`，Goose 自身创建 `goose_db_version` 表，不创建业务表。上线前备份并检查每个迁移的回滚或前向修复方案；同一数据库的迁移命令须串行运行，不能在多个实例并发执行。

`GET /api/v1/health` 始终以 HTTP 200 和 `data.status: "ok"` 表示进程存活，同时在 `data.database.status` 返回 `ok` 或 `unavailable`；`GET /api/v1/ready` 在数据库不可用时返回 503。响应不包含数据库连接详情。生产环境应在可信代理处终止 TLS，并保护到 Server 的链路。

## 验证

执行 `go test ./...`、`go vet ./...` 和 `go build ./...`。真实数据库集成测试会修改 Goose 迁移状态，只能在**全新、专用、可丢弃**的数据库上运行：设置上述数据库环境变量及 `NEXUSOPS_DB_INTEGRATION=1`，然后执行 `go test -count=1 ./...`。该测试要求初始迁移尚未应用，结束时保留已应用状态；再次运行前请使用新的隔离数据库。未设置开关时集成测试会明确跳过。
