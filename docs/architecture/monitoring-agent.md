# NexusOps Agent 监控采集设计

## 模块

```text
runtime scheduler
  ├── cpu collector
  ├── memory collector
  ├── filesystem collector
  └── network collector + counter state
          |
       sample batch
          |
       bounded reporter queue -> HTTPS client
```

采集器只返回结构化样本和错误，不直接写网络或数据库。调度器在固定间隔触发采集，报告器独立发送，保证网络阻塞不会停止心跳。

优先使用成熟的 `gopsutil` 采集 CPU、内存、文件系统和网络信息。依赖保持在 Agent 模块内，不引入 Docker SDK。gopsutil 的平台抽象允许后续扩展 Windows；Linux amd64/arm64 是当前验收目标。

## 采集规则

- CPU：两次采样计算使用率；同时采集核心数和 1/5/15 分钟负载。首次采样只建立基线。
- 内存：报告总量、已用、可用和使用率，统一使用 bytes 和 percent。
- 文件系统：遍历可见挂载点，忽略伪文件系统和明确配置的排除路径；单个挂载点失败不影响其他挂载点。
- 网络：读取每个网卡累计接收/发送字节数。保存上一次计数和时间；时间差小于零、计数回退或网卡变化时返回无速率样本并重置基线。

所有采集时间使用 UTC。数值必须经过有限性、非负性和合理上限校验；采集异常记录不含凭据的结构化日志，并继续下一类采集。

## 调度与资源控制

默认 30 秒采集一次，配置范围 10 秒至 5 分钟。报告队列有界，建议容量 4；队列满时丢弃最旧批次并累计 `metrics_dropped_total`。HTTP 请求超时不超过 10 秒，重试最多 3 次，退避上限 30 秒。心跳使用独立 ticker 和客户端，不与指标报告共享阻塞通道。

Agent 重启后网络基线丢失，第一次网络样本没有速率是预期行为。凭据读取仍使用 0600 文件，任何日志和错误响应都不得包含注册 Token 或 Agent credential。

## 报告契约

指标报告使用现有 Agent 凭据和协议版本，带请求 ID、schema 版本、UTC 采样时间和有限样本列表。Server 返回接受数量；部分样本校验失败时返回明确的逐项错误或整批拒绝策略，Phase 3.1.1 应选择一种并固定。

