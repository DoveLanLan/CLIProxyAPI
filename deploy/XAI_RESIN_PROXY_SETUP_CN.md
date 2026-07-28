# xAI 接入 Resin 配置指南

本方案让 CLIProxyAPI（CPA）只配置一次 Resin，然后按选中的 xAI 凭据自动生成
稳定、匿名的 Resin Account。无需给数百或数千个 auth 文件逐一写 `proxy_url`。

## 工作方式

CPA 使用稳定 auth ID 和一个仅存于 CPA 的 identity key 计算 HMAC-SHA256，生成：

```text
Default.xai-<32位十六进制摘要>
```

该值作为 Resin V1 HTTP/SOCKS5 正向代理用户名，Resin proxy token 作为密码。
原始 auth ID、邮箱、API Key、OAuth token 和 identity key 都不会发给 Resin。

路由优先级如下：

1. auth 文件中的 `proxy_url` 或 `xai-api-key` 中显式设置的 `proxy-url`；
2. `xai-resin-proxy` 自动路由；
3. 旧 `xai-proxy-pool` 自动路由；
4. CPA 全局 `proxy-url`。

`xai-resin-proxy` 与 `xai-proxy-pool` 不允许同时启用。

Resin 无需预先创建 1000 多个 Account。Account 不是静态配置项；每个 xAI 凭据
第一次经过 Resin 时会自动建立对应的粘性租约，CPA auth 文件不会被修改。

正常请求继续使用这个稳定租约。只有 CPA 在尚未向客户端输出内容前收到精确的
`402 personal-team-blocked:spending-limit` 时，才会调用 Resin 管理 API 删除该
Account 的租约，并用同一 xAI 凭据和 Account 建立新连接。Resin 会重新选路，但不
保证每次一定得到不同的公网 IP。

## 1. Resin 配置

Resin 必须使用 V1 认证。如果使用 Resin 仓库自带的生产 Compose，配置
`/opt/resin/.env`：

```env
COMPOSE_PROJECT_NAME=resin
RESIN_IMAGE=ghcr.io/resinat/resin:latest
TZ=Asia/Shanghai

RESIN_AUTH_VERSION=V1
RESIN_ADMIN_TOKEN=替换为强管理密码
RESIN_PROXY_TOKEN=替换为强代理密码

RESIN_BIND_IP=127.0.0.1
RESIN_HOST_PORT=2260
RESIN_NETWORK=vps-gateway
```

使用自建 Resin 镜像时只需替换 `RESIN_IMAGE`。管理 token 与代理 token 必须不同。

`RESIN_PROXY_TOKEN` 需符合 Resin V1 规则，不能包含
`.:|/\@?#%~` 或空白字符。推荐使用由字母、数字、下划线和连字符组成的强随机值。

先创建共享网络并启动 Resin：

```bash
docker network inspect vps-gateway >/dev/null 2>&1 || docker network create vps-gateway
cd /opt/resin
docker compose -f compose.production.yml up -d
docker compose -f compose.production.yml ps
```

通过 Resin 管理页面导入节点订阅并确认有可路由节点。默认 Platform 为 `Default`，
会使用全部可用节点。如果希望只让 xAI 使用特定地区或订阅节点，可创建名为 `XAI`
的 Platform、配置过滤规则，然后把 CPA 配置中的 `platform` 改为 `XAI`。CPA 启用前，
这个 Platform 必须已经存在且至少有一个可路由节点。

Resin 与 CPA 的生产 Compose 都应加入同一个外部 Docker 网络：

```text
vps-gateway
```

在该网络中，CPA 使用 `http://resin:2260` 访问 Resin，无需发布 Resin 代理端口到公网。

## 2. 准备 CPA secret 文件

CPA 需要三个只读文件：

- Resin proxy token：内容必须与 Resin 的 `RESIN_PROXY_TOKEN` 完全一致；
- Resin identity key：仅 CPA 使用，至少 32 字节，并应长期保持不变；
- Resin admin token：内容必须与 Resin 的 `RESIN_ADMIN_TOKEN` 完全一致，仅在启用
  精确 402 重试时需要。

在 VPS 的可信终端执行：

```bash
install -d -m 700 /opt/resin/secrets /opt/cliproxyapi/secrets
read -rsp 'Resin proxy token: ' RESIN_TOKEN_INPUT
printf '\n'
printf '%s' "$RESIN_TOKEN_INPUT" | install -m 600 /dev/stdin /opt/resin/secrets/proxy-token
unset RESIN_TOKEN_INPUT
read -rsp 'Resin admin token: ' RESIN_ADMIN_TOKEN_INPUT
printf '\n'
printf '%s' "$RESIN_ADMIN_TOKEN_INPUT" | install -m 600 /dev/stdin /opt/resin/secrets/admin-token
unset RESIN_ADMIN_TOKEN_INPUT
openssl rand -hex 32 | install -m 600 /dev/stdin /opt/cliproxyapi/secrets/resin-identity-key
```

不要把 identity key 复制到 Resin。只旋转 proxy token 不会改变 Resin Account，但需
同时更新 Resin `.env` 和 CPA token 文件，并重启 Resin、CPA 才会生效。旋转 identity
key 会为全部 xAI 凭据生成新的 Account，并丢失原有粘性租约关联；更新该文件后也要
重启 CPA，单独修改 secret 文件不会触发配置热更新。

## 3. CPA 配置

在 `/opt/cliproxyapi/data/config.yaml` 中设置：

```yaml
# CPA 顶层 proxy-url 保持现有值即可；启用后它不会用于无显式代理的 xAI 凭据。

xai-proxy-pool:
  enabled: false

xai-resin-proxy:
  enabled: true
  proxy-url: "http://resin:2260"
  platform: "Default"
  proxy-token-file: "/run/secrets/resin-proxy-token"
  identity-key-file: "/run/secrets/resin-identity-key"
  admin-url: "http://resin:2260"
  admin-token-file: "/run/secrets/resin-admin-token"
  max-402-retries: 2
```

`max-402-retries: 2` 表示首次请求之后最多再试 2 次，总计最多 3 次上游请求；允许值
为 0–5。设为 0 时不调用 Resin 管理 API，也不要求 admin token，行为与旧版本一致。
只有完全匹配上述状态码和错误 code 才会触发，不会把普通 402、401、403 或 429
当成 IP 拉黑。

不要给普通 xAI auth 文件添加 `proxy_url`，也不要给 `xai-api-key` 条目设置
`proxy-url`；显式值会按设计绕过 Resin 自动派生。如果旧凭据已经有显式代理，需要
先删除或清空该字段，才能统一进入 Resin。

在 `/opt/cliproxyapi/.env` 中设置：

```env
GATEWAY_NETWORK=vps-gateway
ENABLE_XAI_PROXY_POOL=false
ENABLE_XAI_RESIN_PROXY=true
XAI_RESIN_PROXY_TOKEN_FILE=/opt/resin/secrets/proxy-token
XAI_RESIN_IDENTITY_KEY_FILE=/opt/cliproxyapi/secrets/resin-identity-key
XAI_RESIN_ADMIN_TOKEN_FILE=/opt/resin/secrets/admin-token
XAI_RESIN_MAX_402_RETRIES=2
```

`XAI_RESIN_MAX_402_RETRIES` 只用于部署前校验 admin token 挂载，必须与
`config.yaml` 的 `max-402-retries` 保持一致。然后运行 CPA 的生产部署脚本。脚本会
校验三个 secret 文件，挂载到 CPA 容器并拒绝同时启用旧 EgressProxyPool overlay。

```bash
cd /opt/cliproxyapi
bash scripts/remote-deploy.sh
```

## 4. 验证

先确认 Resin 和 CPA 都在共享网络中：

```bash
docker network inspect vps-gateway --format '{{range .Containers}}{{println .Name}}{{end}}'
```

输出应同时包含 `resin` 和 `cli-proxy-api`。再从 VPS 验证 Resin 自身：

```bash
TOKEN="$(cat /opt/resin/secrets/proxy-token)"
curl -x http://127.0.0.1:2260 \
  -U "Default.manual-test:${TOKEN}" \
  https://api.ipify.org
unset TOKEN
```

再通过 CPA 连续调用同一个 xAI 凭据。在 Resin 管理页面确认：

- Platform 为 `Default`（或配置的自定义 Platform）；
- Account 形如 `xai-0123456789abcdef0123456789abcdef`；
- 同一 CPA auth 的 Account 保持不变；
- 不同 CPA auth 的 Account 不同；
- CPA auth 文件本身没有新增动态 `proxy_url`。

HTTP、SSE、WebSocket、Management API 代发请求以及已有 auth 的 OAuth 刷新都使用
同一派生身份。首次 OAuth 登录发生在稳定 auth ID 创建之前，不属于动态身份路由。
Resin 连接故障会作为单次请求级代理错误返回，不会把 xAI 凭据本身标记为失效，也
不会回退到 CPA 顶层 `proxy-url`。

验证精确 402 轮换时，还应确认 Resin 管理 API 中该 Account 的 lease 被删除后重新
建立，并且 CPA 最多只发送配置规定的额外请求。SSE 只会在首个有效 payload 前重试；
WebSocket 只会在握手或首个 payload 前重试。已经开始输出的流即使随后断开也不会
重放。

## 5. 重要限制

Resin 的标准 HTTP CONNECT 隧道本身看不到 TLS 内部的 xAI HTTP 状态码；状态判断由
CPA 完成，再通过仅内网可达且带 Bearer 认证的 Resin 管理 API 删除具体 Account 的
lease。网络连接失败仍由 Resin 自身的节点健康、熔断和切换机制处理，CPA 不会在流
中途自动重放请求。

## 6. 回滚

1. 若只回滚 402 重试，设置 `max-402-retries: 0`，并把
   `XAI_RESIN_MAX_402_RETRIES=0`；
2. 若完全关闭 Resin，设置 CPA `xai-resin-proxy.enabled: false`；
3. 设置 `.env` 中 `ENABLE_XAI_RESIN_PROXY=false`；
4. 重新运行部署脚本或热重载配置；
5. secret 文件可保留以便恢复，auth 文件无需回滚。
