# xAI 专属代理池部署说明

本文只描述仓库已提供的部署形态。真实订阅地址和控制密钥必须在生产
主机上创建，不能提交到 Git。

## 边界

- Mihomo 以普通 Docker bridge sidecar 运行。
- 不使用 TUN、host network、privileged 或额外 Linux capability。
- 端口只通过 Compose `expose` 提供给同一 Docker 网络，不绑定宿主机。
- xAI 流量使用 6 条 lane 和 1 条内部 probe 路径；其他 provider 保持原路径。
- 代理池启用但不可用时 fail closed，不回退到 VPS 直连或全局代理。

## 生产私密文件

在服务器 `/opt/cliproxyapi` 下准备：

```text
secrets/xai-proxy/controller-secret
data/mihomo/config.yaml
data/xai-proxy-pool/
```

要求：

- `controller-secret` 只包含一行随机密钥，权限 `0600`。
- `data/mihomo/config.yaml` 只在首次安装时从
  `mihomo/config.example.yaml` 创建，权限 `0600`。
- bootstrap 中的 `secret` 必须与 `controller-secret` 内容相同；初始
  `proxy-providers` 为空并使用 `REJECT`，因此不会意外直连。
- 启用订阅管理后，不再手工编辑 provider；Management API 是唯一数据源。
- `deploy/secrets/` 和 `deploy/data/` 均已被 `.gitignore` 排除。

## CPA 配置

从仓库 `config.example.yaml` 的 `xai-proxy-pool` 段复制到服务器
`data/config.yaml`，设置 `enabled: true`。首轮旁路验证可先设置较小的
`rollout-percent`，验证完成后再逐步调到 `100`。

同时启用：

```yaml
xai-proxy-pool:
  subscription-management:
    enabled: true
    registry-file: /var/lib/cliproxyapi/xai-proxy-pool/subscriptions.json
    generated-config-file: /var/lib/cliproxyapi/mihomo/config.yaml
```

订阅 URL 只会写入权限为 `0600` 的私密 registry 和 Mihomo 生成配置，查询
接口、状态接口和错误响应不会返回原始地址。

显式认证 `proxy_url` 始终优先于 xAI 代理池。普通 xAI 认证进入池后不会
再经过全局 `proxy-url`。

## Compose 开关

在服务器 `.env` 中设置：

```env
ENABLE_XAI_PROXY_POOL=true
MIHOMO_IMAGE=docker.io/metacubex/mihomo:v1.19.28@sha256:e6acd921addecfd59a8e2d38203f88356d635b54de6c0673db0e015139989312
```

部署脚本会追加 `compose.production.xai-proxy.yml`。CPA 和 Mihomo 共享
`data/mihomo/` 私密目录，以便 API 原子更新配置；该目录不发布到宿主机端口。

## 通过 API 管理订阅

先查询 revision。默认部署端口是 `18317`；如果生产 `.env` 覆盖了
`TAILSCALE_MANAGEMENT_PORT`，请替换成实际端口：

```bash
curl -sS -H "X-Management-Key: $KEY" -H "Authorization: Bearer $KEY" \
  http://100.67.99.9:18317/v0/management/xai-proxy-pool/subscriptions
```

响应体中的 `revision` 和响应头 `ETag` 表示当前版本，但响应不包含订阅
URL。创建或修改前，将请求体写入本机权限为 `0600` 的临时 JSON 文件，
避免 URL 进入 shell history：

```json
{"name":"provider-a","url":"https://subscription.example.invalid/write-only","enabled":true}
```

```bash
curl -sS -X POST \
  -H "X-Management-Key: $KEY" -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' -H 'If-Match: "0"' \
  --data-binary @/path/to/private-request.json \
  http://100.67.99.9:18317/v0/management/xai-proxy-pool/subscriptions
```

每次成功变更都会增加 revision；添加多个订阅时，重复 `POST` 并将下一次
`If-Match` 更新为上一次响应中的新 `ETag`。修改 URL 或启用状态使用：

```bash
curl -sS -X PUT \
  -H "X-Management-Key: $KEY" -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' -H 'If-Match: "1"' \
  --data-binary @/path/to/private-update.json \
  http://100.67.99.9:18317/v0/management/xai-proxy-pool/subscriptions/provider-a
```

`private-update.json` 可只包含要修改的字段，例如
`{"enabled":false}`。检查已启用 provider 使用：

```bash
curl -sS -X POST \
  -H "X-Management-Key: $KEY" -H "Authorization: Bearer $KEY" \
  http://100.67.99.9:18317/v0/management/xai-proxy-pool/subscriptions/provider-a/check
```

永久删除前必须先用 `PUT` 停用并确认 lane 已排空，然后携带最新 revision：

```bash
curl -sS -X DELETE \
  -H "X-Management-Key: $KEY" -H "Authorization: Bearer $KEY" \
  -H 'If-Match: "2"' \
  http://100.67.99.9:18317/v0/management/xai-proxy-pool/subscriptions/provider-a
```

启用态新增/更新只有在下载、解析、Mihomo 热重载、provider 节点发现和 lane
协调全部成功后才提交；失败会恢复旧配置。删除必须先通过 `PUT` 设置
`enabled:false`，待 lane 排空后再调用 `DELETE`。

## 上线顺序

1. 保持 CPA `xai-proxy-pool.enabled: false`，启动空 provider 的 Mihomo sidecar。
2. 确认控制 API 未暴露到宿主机，并通过订阅 API 添加至少 7 个独立出口节点。
3. 开启代理池并设置小比例 `rollout-percent`，验证 HTTP、SSE、WebSocket 和 OAuth 刷新。
4. 查看 `/v0/management/xai-proxy-pool/status` 中的 lane、出口 IP 和隔离状态。
5. 逐步扩大到 50%，最后切换到 100%。

管理 API 还提供 provider 刷新、lane 轮换/检查和 IP 隔离操作。所有接口
继续受现有 Management API密钥与内网访问策略保护。

## 回滚

- 自动故障：代理池保持 fail closed，不会泄漏到旧出口。
- 人工回滚：显式设置 `xai-proxy-pool.enabled: false`，CPA 恢复原有
  `auth proxy -> global proxy -> direct` 行为。
- 移除 sidecar：设置 `ENABLE_XAI_PROXY_POOL=false` 后重新应用 Compose。
- `data/xai-proxy-pool/state.json` 不含订阅 URL、代理密码或 xAI 凭证，可独立归档。
- `data/xai-proxy-pool/subscriptions.json` 和 `data/mihomo/config.yaml` 含有
  write-only 订阅地址，回滚和备份时必须继续按 secret 文件处理。
