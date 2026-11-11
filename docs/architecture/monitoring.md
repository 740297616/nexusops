# NexusOps 主机监控架构

## 范围

Phase 3.1 监控主机 CPU、内存、文件系统和网络累计计数。Agent 主动采集并通过现有 `Authorization: Agent` HTTPS 通道上报，Server 根据凭据反查 `server_id`，校验后写入 PostgreSQL。Web 只通过 Server 查询指标，不直接访问 Agent 或数据库。

现有注册、心跳、凭据轮换、撤销、审计和 Agent 限流链路保持不变。指标上报与心跳分离：心跳表示连接存活，指标报告表示一批采样数据可用，二者失败状态分别展示。

## 数据流

```text
Linux Agent collectors
        |
        | HTTPS POST /api/v1/agent/metrics
        v
Agent auth + body/version/time validation
        |
        v
Server metric service -> PostgreSQL metric_samples
        |
        +--> latest snapshot query
        +--> bounded time-range query / aggregation
        v
Web monitoring API -> charts and stale/offline state
```

Agent 使用单个采集调度器，采集器之间互不阻塞；网络请求通过有限队列发送。发送失败不阻塞下一次心跳，队列满时丢弃最旧的指标批次并记录本地计数。

## 指标模型

每条样本包含 `server_id`、`agent_id`、`metric_type`、`metric_name`、`labels`、`value`、`unit`、`sampled_at`、`received_at` 和 `schema_version`。服务器身份来自 Agent 凭据，客户端不能提交目标服务器 ID 作为授权依据。

指标类型：

- `cpu`: `usage_percent`、`load_1m`、`load_5m`、`load_15m`、`cores`
- `memory`: `total_bytes`、`used_bytes`、`available_bytes`、`usage_percent`
- `filesystem`: `total_bytes`、`used_bytes`、`available_bytes`、`usage_percent`，标签包含挂载点和文件系统类型
- `network`: `rx_bytes_total`、`tx_bytes_total`、`rx_bytes_per_second`、`tx_bytes_per_second`，标签包含网卡名

CPU 和内存使用率由 Agent 在采样时计算。磁盘按挂载点采集。网络速率由相邻累计计数和采样时间差计算；首次采样、计数器回退和网卡重新出现时速率为 `null`，下一次采样重新建立基线。

## 采集与传输

默认采集间隔建议为 30 秒，可通过 Agent 配置调整，允许范围 10 秒至 5 分钟。每个报告最多 500 条样本，单请求体不超过 256 KiB。样本时间必须是 UTC，允许的客户端时钟偏差建议为 5 分钟；Server 以接收时间判断新鲜度。

Agent 采用有限指数退避，最大重试间隔 30 秒；非关键采集失败只影响对应指标，Agent 继续心跳。报告请求使用请求 ID，Server 对重复请求可安全忽略或按样本唯一键去重。

## 存储与查询

Phase 3.1.1 建议使用以下逻辑表，具体列类型和约束通过新的 Goose Migration 确定：

- `metric_samples`：样本批次、服务器、Agent、请求 ID、采样时间、接收时间和 schema 版本；`(server_id, request_id)` 唯一。
- `metric_values`：样本中的 CPU/内存数值，按 `sample_id, metric_name` 唯一。
- `metric_filesystems`：样本中的挂载点、文件系统和容量数值，按 `sample_id, mountpoint` 唯一。
- `metric_networks`：样本中的网卡、累计计数和速率，按 `sample_id, interface` 唯一。
- `metric_rollups`：按服务器、指标、标签和时间桶保存的聚合值，区分 `avg`、`min`、`max`、`last`。

原始样本按 `server_id, sampled_at`、指标类型和时间建立索引，子表通过 `sample_id` 关联。查询必须有服务器、时间范围和最大点数，Server 负责按时间桶聚合，避免 Web 拉取无界原始数据。最新值查询从每类指标最近有效样本读取，并返回 `sampled_at`、`received_at` 和 `stale`。

按 30 秒采样估算，每台主机每天约 2,880 个批次；100 台主机约 288,000 个批次/天。容量预算应同时计算磁盘和网卡子行，超过约千万级原始样本后优先启用时间分区和聚合表，避免依赖单表无限增长。

建议同时维护每台服务器每类指标的最新快照表，减少 Dashboard 读取历史表的压力；快照不是唯一事实，原始样本仍用于有限历史查询。

## 保留与清理

建议默认策略：原始样本保留 7 天，5 分钟聚合保留 30 天，1 小时聚合保留 180 天；具体周期通过环境变量配置。清理任务按时间分区或有限批次执行，先删除过期原始样本，再删除过期聚合数据，不删除最新快照。生产规模较大时优先采用按月分区并在保留期结束后删除分区，避免长事务。

指标数据不是审计事实。采集失败、离线和过期状态由 Server 根据最后接收时间计算并返回，前端不得把旧样本标记为实时数据。

