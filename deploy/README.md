# Docker Compose

需要 Docker Engine、Compose v2 和可用的镜像仓库网络。配置在 `deploy/`；以下命令从仓库根目录执行。复制 `deploy/.env.example` 为 `deploy/.env`，设置非空且唯一的 `DB_PASSWORD`，并限制文件读取权限；不要提交该文件。Compose 读取此文件用于变量替换，不会把它打包进镜像。密码包含 `$` 等 Compose 特殊字符时，应按 Compose 环境文件规则引用或转义。

首次使用生产式镜像：

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.yml config --quiet
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --build
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm --build migrate up
```

也可在 `deploy/` 目录直接运行 `docker compose up -d --build`，Compose 会自动读取该目录的 `.env`。Web 默认地址为 `http://127.0.0.1:8080`，`/api/` 由 Nginx 转发至 Server。`WEB_BIND_ADDRESS` 和 `WEB_PORT` 可调整监听；不要在没有访问控制的情况下向公网开放，当前应用没有认证。

后续发布如果包含结构变更，应先备份数据库，启动 PostgreSQL、执行一次迁移并检查结果，再启动或更新依赖新结构的 Server：

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d postgres
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm --build migrate status
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm --build migrate up
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up -d --build
```

回滚最近一次迁移使用相同 `run` 命令加 `down`。迁移不会随 Server 自动运行；同一数据库必须由一个操作者串行执行迁移。回滚前检查数据影响并备份。首次启动已存在的 PostgreSQL 数据卷后，修改 `DB_USER`、`DB_PASSWORD` 或 `DB_NAME` 不会自动修改数据库内的角色或库；请通过 PostgreSQL 管理流程修改并同步配置。

应用迁移后，在当前 shell 安全设置 `NEXUSOPS_ADMIN_USER`、`NEXUSOPS_ADMIN_DISPLAY_NAME`、`NEXUSOPS_ADMIN_PASSWORD`，运行以下一次性初始化命令，再从 shell 环境移除这些变量：

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.yml run --rm --no-deps -e NEXUSOPS_ADMIN_USER -e NEXUSOPS_ADMIN_DISPLAY_NAME -e NEXUSOPS_ADMIN_PASSWORD --entrypoint /usr/local/bin/nexusops-init-admin server
```

不要将初始密码写入 Compose 文件或命令参数；容器运行时管理员环境仍可被 Docker 管理者读取。已存在管理员时初始化会失败。

## 开发

开发覆盖文件以 Vite 替换 Nginx，绑定本地 `web/` 并启用文件轮询和热更新；Server 和 PostgreSQL 仍使用容器。两个配置使用相同项目和数据库卷，不应同时作为不同环境运行。

```sh
docker compose --env-file deploy/.env -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up -d --build
docker compose --env-file deploy/.env -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml logs -f web
```

也可用 `make dev`；无 Make 时直接使用上述 Compose 命令。需要单独操作服务时，使用相应配置的 `up -d --build web`、`up -d --build server`、`stop web` 或 `restart server`。使用 `--no-deps` 可在已有依赖独立运行时仅启动指定服务；Vite 的 `/api` 代理依赖可用的 Server。宿主机单独运行前端或 Server 的方法见各自 README；默认 Compose 数据库不发布宿主机端口。

## 网络、数据与安全

Web 和 Server 连接 `frontend` 网络；PostgreSQL 仅连接 `internal` 的 `database` 网络，Server 同时加入两个网络，数据库不发布端口。`postgres_data` 是持久卷，普通 `down`、容器重建和重启不会删除数据。`docker compose down -v` 会删除卷及数据，使用前必须确认备份。开发覆盖文件还使用 `web_node_modules` 卷。

示例配置的 `DB_SSLMODE=disable` 只适合受信任的单机开发环境，Docker 内部网络不是数据库 TLS 的替代品。正式部署须配置 PostgreSQL TLS、为 Server 和迁移容器提供可信 CA，并设置 `DB_SSLMODE=verify-full`；证书名称须与 `postgres` 主机名匹配。容器内的数据库帐号在初始化时由官方镜像创建，生产环境应另外规划最小权限、秘密注入、备份和恢复。镜像提供 CA 证书，但此示例不配置数据库服务器证书。外部 HTTPS 由可信反向代理终止，并应保护代理至 Web 的链路。

Server 镜像以非 root 用户运行，健康检查使用 readiness；Web 健康检查只检查静态入口。Nginx 的 `index.html` 不做长期缓存，带哈希的 `/assets/` 长期缓存，API 不缓存；SPA 路由回退到 `index.html`。查看状态和日志可使用 `docker compose ... ps`、`docker compose ... logs -f`。普通关闭使用 `docker compose ... down`。
