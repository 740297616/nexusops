# NexusOps 监控 API 契约

以下为 Phase 3.1.1 实现前的公共契约草案，统一使用现有 `{data,error}` 响应封装和 `/api/v1` 前缀。

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
{"data":{"server_id":"...","status":"online","observed_at":"...","stale":false,"metrics":[]},"error":null}
```

### `GET /api/v1/servers/{id}/metrics/history`

需要相同权限。参数：`metric_type`、`metric_name`、`interface`/`mountpoint`、`from`、`to`、`step`、`limit`。时间范围和点数有服务端上限；`step=raw` 只允许短时间范围，较长范围返回时间桶聚合。

响应包含 `series`、`sampled_at`、`received_at`、`aggregation` 和 `stale`。聚合数据明确标注 `avg`、`min`、`max` 或 `last`，Web 不得把聚合值显示为单次实时采样。

## 权限与错误

Agent 接口只接受 Agent 凭据，拒绝用户 Bearer Token。查询接口只返回用户有权访问的服务器；未知服务器和无权限资源使用安全一致的错误响应。数据库故障返回 `service_unavailable`，不泄露 SQL、地址或凭据。

