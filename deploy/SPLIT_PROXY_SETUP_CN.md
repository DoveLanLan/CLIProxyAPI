# Split Proxy 服务器操作说明

这份文档用于在服务器上启用 `split-proxy` 方案，让 `CLIProxyAPI` 保留全局代理，同时对本地 Claude 兼容服务走直连。

适用场景：

- 全局上游代理可以访问公网
- 但全局上游代理拒绝访问 `localhost`、`127.0.0.1`、Docker 服务名或私网 IP
- 你希望 `CLIProxyAPI` 访问本地 `kirors-kiro` 之类的服务时不再经过外部代理

## 方案说明

最方便的做法是把这些代理参数放到服务器上的 `/opt/cliproxyapi/.env`，不要放进 GitHub Actions。

当前仓库已经按这种方式接好了：

- `deploy-production` workflow 不需要改
- 工作流会照常上传 `deploy/` 目录
- 服务器上执行 [scripts/remote-deploy.sh](/root/Projects/Go/src/CLIProxyAPI/deploy/scripts/remote-deploy.sh)
- 脚本会读取服务器上的 `.env`
- 当 `ENABLE_SPLIT_PROXY=true` 时，脚本会自动追加 [compose.production.split-proxy.yml](/root/Projects/Go/src/CLIProxyAPI/deploy/compose.production.split-proxy.yml)

这样做的好处：

- 不用每次手动 `export`
- 不用把代理账号密码塞进 GitHub Actions secrets
- GitHub Actions 继续只负责构建镜像和触发部署

## 服务器目录

服务器上的部署目录建议为：

```text
/opt/cliproxyapi/
  .env
  compose.production.yml
  compose.production.split-proxy.yml
  split-proxy/start.sh
  data/config.yaml
  data/auths/
  data/logs/
  data/logs/split-proxy/
  scripts/remote-deploy.sh
```

## 步骤 1：登录服务器

```bash
ssh root@你的服务器IP
cd /opt/cliproxyapi
```

## 步骤 2：编辑服务器上的 `.env`

把 `/opt/cliproxyapi/.env` 改成类似下面这样：

```env
COMPOSE_PROJECT_NAME=cliproxyapi
CLI_PROXY_IMAGE=ghcr.io/dovelanlan/cliproxyapi:main
TZ=Asia/Shanghai
PUBLIC_BIND_IP=23.175.201.12
TAILSCALE_BIND_IP=100.67.99.9
TAILSCALE_MANAGEMENT_PORT=18317

ENABLE_SPLIT_PROXY=true
UPSTREAM_PROXY_HOST=proxy.example.com
UPSTREAM_PROXY_PORT=3128
UPSTREAM_PROXY_LOGIN=your-user:your-password
DIRECT_DOMAINS="localhost host.docker.internal kirors-kiro"
```

说明：

- `ENABLE_SPLIT_PROXY=true` 表示启用本地分流代理 sidecar
- `UPSTREAM_PROXY_HOST` / `UPSTREAM_PROXY_PORT` / `UPSTREAM_PROXY_LOGIN` 是你的外部 HTTP 代理信息
- `DIRECT_DOMAINS` 表示这些目标不走外部代理，直接连接

如果你想编辑：

```bash
vi /opt/cliproxyapi/.env
```

或者直接覆盖写入：

```bash
cat > /opt/cliproxyapi/.env <<'EOF'
COMPOSE_PROJECT_NAME=cliproxyapi
CLI_PROXY_IMAGE=ghcr.io/dovelanlan/cliproxyapi:main
TZ=Asia/Shanghai
PUBLIC_BIND_IP=23.175.201.12
TAILSCALE_BIND_IP=100.67.99.9
TAILSCALE_MANAGEMENT_PORT=18317

ENABLE_SPLIT_PROXY=true
UPSTREAM_PROXY_HOST=proxy.example.com
UPSTREAM_PROXY_PORT=3128
UPSTREAM_PROXY_LOGIN=your-user:your-password
DIRECT_DOMAINS="localhost host.docker.internal kirors-kiro"
EOF
```

## 步骤 3：修改运行中的 `data/config.yaml`

编辑服务器上的：

```bash
vi /opt/cliproxyapi/data/config.yaml
```

把全局代理改成：

```yaml
proxy-url: "http://split-proxy:3128"
```

这是关键步骤。启用后，`CLIProxyAPI` 会先把请求发给本地 `split-proxy` 容器：

- 本地目标直连
- 其他目标继续转发到你的外部代理

## 步骤 4：修改本地 Claude 上游地址

在 `data/config.yaml` 里，把本地 Claude 兼容服务的 `base-url` 改成：

```yaml
base-url: "http://host.docker.internal:8990"
```

如果你的 `kirors-kiro` 与 `split-proxy` 在同一个 Docker 网络里，也可以使用：

```yaml
base-url: "http://kirors-kiro:8990"
```

不要再使用：

```yaml
base-url: "http://localhost:8990"
```

原因：

- 启用 `split-proxy` 后，请求会先进入 `split-proxy` 容器
- 此时 `localhost` 指的是 `split-proxy` 容器自己
- 它不再表示宿主机

## 步骤 5：执行部署

如果你只是想在服务器上手动重启当前部署：

```bash
cd /opt/cliproxyapi
CLI_PROXY_IMAGE=ghcr.io/dovelanlan/cliproxyapi:main bash scripts/remote-deploy.sh
```

如果你使用 GitHub Actions 自动部署，也不需要额外改 workflow。
下次 Actions 触发部署时，会自动读取服务器上的 `.env` 并带上 `split-proxy` 配置。

## 步骤 6：检查容器是否启动成功

```bash
cd /opt/cliproxyapi
docker compose -f compose.production.yml -f compose.production.split-proxy.yml ps
```

重点确认这几个容器：

- `cli-proxy-api`
- `cli-proxy-api-nginx`
- `cli-proxy-split-proxy`

查看日志：

```bash
docker compose -f compose.production.yml -f compose.production.split-proxy.yml logs -f split-proxy
docker compose -f compose.production.yml -f compose.production.split-proxy.yml logs -f cli-proxy-api
tail -f /opt/cliproxyapi/data/logs/split-proxy/access.log
tail -f /opt/cliproxyapi/data/logs/split-proxy/cache.log
```

说明：

- `cli-proxy-api` 仍然可以直接用 `docker compose logs -f cli-proxy-api`
- `split-proxy` 的详细 Squid 日志现在持久化在 `/opt/cliproxyapi/data/logs/split-proxy/`
- 这样可以避免 Squid 因为写 `/dev/stdout`、`/dev/stderr` 权限不足而反复重启

## 常见结论

如果之前出现下面这种情况：

- `base-url: http://localhost:8990` 在宿主机代理下能通
- 换成远程外部代理后报 `403 client_connect_invalid_ip`

那通常说明：

- 外部代理拒绝访问回环地址或私网地址
- 远程代理把 `localhost` 视为“代理服务器自己”
- `split-proxy` 的作用就是在本机先做一次分流，再决定哪些请求交给远程代理

## 明天操作时最重要的三点

1. 服务器 `.env` 里开启 `ENABLE_SPLIT_PROXY=true`
2. `data/config.yaml` 里把全局 `proxy-url` 改成 `http://split-proxy:3128`
3. 本地 Claude 上游不要再写 `localhost:8990`，改成 `host.docker.internal:8990`
