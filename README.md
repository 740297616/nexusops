# nexusops

NexusOps 是一个自托管基础设施管理平台，计划统一查看和管理多台 Linux 主机、Docker 容器、系统资源与服务状态。

## 当前状态

Phase 1.1 提供 Web 工程骨架和 Server 健康检查；尚未实现业务功能、数据库或部署配置。请勿将设计文档中的目标视为已实现功能。

## 技术方向

- Web：Vue 3、TypeScript、Vite、Pinia、Element Plus、ECharts
- Server：Go、Gin、GORM、PostgreSQL
- Agent：Go；主动通过 HTTPS 和 WebSocket 连接 Server
- 部署：Docker Compose；MVP 不引入 Kubernetes 或 Redis

Server 负责认证、API、业务逻辑与数据库；Agent 负责主机信息采集和受限操作，不开放公网管理端口。首次注册使用一次性 Token，成功后使用独立身份凭证；第一版不支持任意 Shell 执行。

## 仓库导航

- [PRD.md](PRD.md)：产品范围、用户流程与验收目标
- [ARCHITECTURE.md](ARCHITECTURE.md)：组件边界、通信与安全设计
- [DATABASE.md](DATABASE.md)：数据模型、迁移与保留策略
- [AGENTS.md](AGENTS.md)：AI 协作与工程规范
- `web/`、`server/`：前端与 Server 工程；`agent/` 为预留目录
- `deploy/`、`docs/`、`scripts/`：部署、补充文档和脚本预留目录

## 开发

分别参见 [Web 开发说明](web/README.md) 和 [Server 开发说明](server/README.md)。本阶段不包含数据库、Agent 通信或 Docker Compose。使用许可尚待项目所有者确定，参见 [LICENSE](LICENSE)。
