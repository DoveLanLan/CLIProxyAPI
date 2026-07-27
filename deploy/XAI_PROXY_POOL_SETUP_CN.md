# xAI 专属代理池部署说明

代理池现在由独立项目 `EgressProxyPool` 维护。CLIProxyAPI 只负责识别 xAI
精确 402、判断请求是否可以安全重放，并调用代理池的私有控制 API。

## 项目边界

```text
CLIProxyAPI
  ├── HMAC(auth ID) -> EgressProxyPool 控制 API
  └── xAI 请求 -> Mihomo lane/probe

EgressProxyPool
  ├── controller：订阅、健康检查、出口 IP、lane、隔离、状态
  └── mihomo：实际代理连接和 provider 节点
```

- EgressProxyPool 使用独立 Compose 项目和持久化目录。
- CLIProxyAPI 不再读取 Mihomo controller secret，也不再写 Mihomo 配置。
- 两个项目只通过名为 `egress-proxy` 的私有 Docker 网络连接。
- 默认不发布控制 API、Mihomo controller 或代理 lane 到宿主机。
- 代理池不可用时，已纳入池的 xAI 请求 fail closed，不回退到 VPS 直连。

## 1. 启动 EgressProxyPool

将新项目部署到 `/opt/egress-proxy-pool`：

```bash
cd /opt/egress-proxy-pool
cp config.example.yaml config.yaml
mkdir -p secrets data/pool data/mihomo
openssl rand -hex 32 > secrets/api-token
openssl rand -hex 32 > secrets/mihomo-controller
chmod 600 config.yaml secrets/api-token secrets/mihomo-controller
docker compose up -d --build
```

controller 会根据私有 registry 生成 `data/mihomo/config.yaml`。首次启动时
registry 为空，所有 lane 指向 `REJECT`，因此不会意外直连。

检查状态：

```bash
docker compose ps
docker compose logs --tail=100 controller mihomo
```

## 2. 连接 CLIProxyAPI

在 `/opt/cliproxyapi/.env` 中设置：

```env
ENABLE_XAI_PROXY_POOL=true
EGRESS_PROXY_NETWORK=egress-proxy
EGRESS_PROXY_API_TOKEN=/opt/egress-proxy-pool/secrets/api-token
```

在 CLIProxyAPI 的 `data/config.yaml` 中设置：

```yaml
xai-proxy-pool:
  enabled: true
  service-url: http://egress-proxy-controller:8080
  service-token-file: /run/secrets/egress-proxy-api-token
```

然后运行 CLIProxyAPI 的生产部署脚本。脚本会确认共享网络和 token 文件
存在，再把 CLIProxyAPI 容器接入该网络。

显式认证 `proxy_url` 始终优先于代理池。CLIProxyAPI 发给控制面的稳定键是
使用 API token 计算的 HMAC-SHA256，不包含原始 auth ID 或 xAI token。

## 3. 管理订阅

推荐直接调用 EgressProxyPool `/v1/subscriptions`。也可以继续调用原有
CLIProxyAPI `/v0/management/xai-proxy-pool/subscriptions`；该接口现在只是兼容
转发层。

先在受控终端读取 token，避免将订阅 URL 写入 shell history：

```bash
TOKEN="$(cat /opt/egress-proxy-pool/secrets/api-token)"
```

查询 registry revision：

```bash
curl -sS \
  -H "Authorization: Bearer $TOKEN" \
  http://127.0.0.1:8080/v1/subscriptions
```

如果控制 API 没有绑定宿主机，请从同一 Docker 网络中的运维容器调用，或
仅通过 Tailscale 私有反向代理暴露 controller。不要公开 Mihomo 端口。

创建订阅时，将请求体保存在权限为 `0600` 的文件中：

```json
{"name":"provider-a","url":"https://subscription.example.invalid/write-only","enabled":true}
```

```bash
curl -sS -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -H 'If-Match: "0"' \
  --data-binary @/path/to/private-request.json \
  http://127.0.0.1:8080/v1/subscriptions
```

更新使用 `PUT /v1/subscriptions/:name`。永久删除前必须先设置
`enabled:false`，确认 lane 不再使用该 provider，再携带最新 `If-Match`
revision 调用 `DELETE`。

订阅 URL 只存在于 EgressProxyPool 的 mode-`0600` registry 和生成的 Mihomo
配置中，不会出现在查询响应、状态、错误或日志中。启用态变更仍执行完整的
下载、Mihomo 热重载、节点发现、lane 协调和失败回滚事务。

## 4. 上线顺序

1. 保持 CLIProxyAPI `xai-proxy-pool.enabled: false`。
2. 启动 EgressProxyPool，添加至少 7 个具有独立出口的可用节点。
3. 在 EgressProxyPool `config.yaml` 中先设置较小的 `rollout-percent`。
4. 启用 CLIProxyAPI 连接配置，验证 HTTP、SSE、WebSocket 和 OAuth 刷新。
5. 查看 CLIProxyAPI 兼容状态接口或 EgressProxyPool `/v1/status`。
6. 在 EgressProxyPool 中逐步扩大 rollout，最终切到 100%。

## 5. 回滚

- 临时停用：设置 CLIProxyAPI `xai-proxy-pool.enabled: false` 并热重载或重启。
- 拆除连接：设置 `ENABLE_XAI_PROXY_POOL=false` 后重新应用 CLIProxyAPI Compose。
- EgressProxyPool 的 state、registry 和 Mihomo provider cache 独立保留，不受
  CLIProxyAPI 容器更新影响。
- 不要在仍有已纳入请求时直接停止 Mihomo；控制面轮换只影响新连接，容器
  重启会中断现有流。
