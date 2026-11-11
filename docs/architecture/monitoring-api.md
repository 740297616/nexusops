# NexusOps 监控 API 契约

统一使用现有 `{data,error}` 响应封装和 `/api/v1` 前缀。

## Agent 上报

### `POST /api/v1/agent/metrics`

需要 `Authorization: Agent <credential>`。凭据绑定的 `server_id` 是唯一目标；请求体不能通过字段覆盖身份。

请求：

```json
{
  "protocol_version": "1",
  "schema_version": 1,
  "request_id": "uuid",
  "sampled_at": "2026-11-11T12:00:00Z",
  "samples": [
    {
      "metric_type": "cpu",
      "metric_name": "usage_percent",
      "labels": {},
      "value": 37.4,
      "unit": "percent"
    },
    {
      "metric_type": "network",
      "metric_name": "rx_bytes_total",
      "labels": {"interface": "eth0"},
      "value": 123456,
      "unit": "bytes"
    }
  ]
}
```

响应 `200`：

```json
{"data":{"accepted":2,"received_at":"2026-11-11T12:00:01Z","request_id":"uuid"},"error":null}
```

请求大小、样本数量、协议版本、时间偏差、数值范围和标签长度必须校验。错误码包括 `unauthorized`、`agent_rejected`、`invalid_request`、`rate_limited`、`service_unavailable`。

## Server 查询

### `GET /api/v1/servers/{id}/metrics/latest`

需要用户 Bearer 认证和 `infrastructure:server:read`。返回每类指标的最近有效值及新鲜度：

```json
{"data":{"server_id":"...","status":"online","observed_at":"...","last_sampled_at":"...","metrics_status":"fresh","stale":false,"metrics":[]},"error":null}
```

`status` 是现有心跳状态，`metrics_status` 独立根据最后采样时间与 `METRICS_STALE_AFTER` 判断为 `fresh`、`stale` 或 `missing`；无数据时 `last_sampled_at` 为 `null`。`observed_at` 保持原来的接收时间语义。兼容字段 `stale` 在指标不新鲜或主机不在线时为 true；新客户端应分别使用 `status` 和 `metrics_status`。

### `GET /api/v1/servers/{id}/metrics/history`

需要相同权限。必填 `metric_type`、`metric_name`，文件系统和网络还须分别提供 `mountpoint`、`interface`；`from`、`to` 为 RFC3339 时间，默认最近一小时。`step` 默认 `raw`，可选 `raw`、`5m`、`1h`，最大范围分别为 7、30、180 天；`limit` 默认及上限均为 2000。结果按时间升序，超出上限的点不会自动分页。

响应的 `aggregation` 标识所选步长。聚合点的 `sampled_at` 为 UTC 桶起点，`last_sampled_at` 和 `received_at` 为桶内最后采样及其接收时间，`sample_count`、`min_value`、`max_value` 是有效样本的统计；`value` 对累计网络字节数为桶内最后计数，对其他指标为算术平均值。累计计数是计数器读数，不是桶内流量增量，不能跨重启/计数器回退直接相减。空桶不会生成 0 或伪造点；`missing_windows` 列出缺失的 UTC 桶起点，结果或检查窗口达到 `limit` 时 `truncated=true`，客户端需缩小时间范围继续查询。未结束或尚未聚合的桶也会显示为缺失。整个范围无数据时 `series` 为空且 `metrics_status=missing`。历史响应状态针对返回的范围，不能用作主机当前状态，应查询 latest。

## 权限与错误

Agent 接口只接受 Agent 凭据，拒绝用户 Bearer Token。查询接口只返回用户有权访问的服务器；未知服务器和无权限资源使用安全一致的错误响应。数据库故障返回 `service_unavailable`，不泄露 SQL、地址或凭据。

