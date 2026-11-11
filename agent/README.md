# NexusOps Agent

独立运行在 Linux 主机上的基础信息采集 Agent。Agent 主动通过 HTTPS 注册和发送心跳，不开放管理端口，不执行任意 Shell。

环境变量：`NEXUSOPS_SERVER_URL`、`NEXUSOPS_AGENT_REGISTRATION_TOKEN`（首次注册后移除）、`NEXUSOPS_AGENT_CREDENTIAL_FILE`、`NEXUSOPS_AGENT_HEARTBEAT_INTERVAL`。

指标采集默认每 30 秒执行一次，可通过 `NEXUSOPS_AGENT_METRICS_INTERVAL`（10 秒至 5 分钟）和 `NEXUSOPS_AGENT_METRICS_QUEUE_SIZE` 配置。指标报告失败不会停止心跳；指标队列有界，网络中断时会丢弃旧批次并重试。

构建：

```sh
GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=0.1.0" -o nexusops-agent ./cmd/agent
GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=0.1.0" -o nexusops-agent ./cmd/agent
```

凭据文件默认 `/var/lib/nexusops-agent/credential`，权限为 0600。生产环境必须使用 HTTPS，并通过 systemd 的 `EnvironmentFile` 或受保护的密钥文件注入配置。
