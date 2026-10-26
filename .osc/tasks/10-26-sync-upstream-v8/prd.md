# Refactor: Sync upstream v8.0.43 preserving deploy, CPA-Manager and local providers

## 现状

本地 `main` 与 `upstream/main`（`router-for-me/CLIProxyAPI`）已显著分叉：

- 上次同步点：`e92bd900`（2026-07-21，合并上游 `01f387f4` = `v7.2.93`，由任务 `07-01-sync-upstream-v7.2.48` 落地）。
- 上游当前 HEAD：`upstream/main` @ `v8.0.43`。
- 待合入上游提交数：`1441`（`v7.2.93..upstream/main`），改动文件 `1810`。
- 本地独有提交：`94`（`upstream/main..HEAD`），触及文件 `552`，其中非 docs/osc 的实际代码与部署文件约 `300`。
- 双方共同修改（潜在冲突源）：`52` 个文件；`git merge-tree` 实测内容冲突 `34` 个。

### 上游关键变化

1. **大版本迁移 v7 → v8**：`31f4cfab feat: migrate CLIProxyAPI to version 8`（912 个文件的 import 路径 `.../v7/...` → `.../v8/...`）。
   - `go.mod`: `module .../v8`，`go 1.26.10`。
   - 新增依赖：`pion/webrtc`、`pion/ice`、`pion/datachannel`、`pion/rtp`、`pion/sdp`、`pion/stun`、`libp2p/zeroconf`、`dlclark/regexp2` 等。
2. **Antigravity 大量演进**：触及 antigravity 文件的提交 797 个（标题含 antigravity 的 155 个）。主要方向：
   - executor 由单文件拆分为 `_auth/_credits/_execute/_request/_stream/_tokens` 六个文件。
   - 新增模块：`internal/signature/claude_antigravity_validation.go`、`internal/translator/common/antigravity_tools.go`、`internal/cache/antigravity_interactions_session.go`、`helps/antigravity_{compaction,connection_trace,interactions_state}.go`、`sdk/cliproxy/antigravity_model_probes.go`。
   - 新能力：conversation compaction + capsule 加密、Claude 5.5 CAQS thinking signature replay、per-account model entitlements、web search 翻译、`video_url`、敏感词混淆。
3. **Home / session affinity 子系统**：Merkle LCP session affinity、session tree、credential 并发与 CAS replay。
4. **其它**：Devin/Cognition provider、pluginhost payload 规则、httptransport 装饰器、payload finalizer 等。

### 本地独有定制（必须保留）

- 部署/运维（protected paths）：`.github/**`、`Dockerfile`、`.dockerignore`、`docker-build.*`、`docker-compose*.yml`、`.env.cluster.example`、`deploy/**`、`.goreleaser.yml`。
- CPA-Manager：`internal/config/config.go`（`DefaultPanelGitHubRepository = seakee/CPA-Manager-Plus`）、`internal/managementasset/updater.go`（release/fallback URL）、`config.example.yaml`。
- 协议/运行时定制：
  - xAI 专属轮换代理池 + Resin（`internal/{api,config,runtime,thinking,auth}/**xai*` 等）。
  - Command Code provider（`commandcode_*`、`sdk/cliproxy/auth/commandcode.go`）。
  - DeepSeek 模型与 reasoning echo（`openai_compat_deepseek.go`、registry overrides）。
  - DeepSeek/OpenAI vision 预检与 tool result image 迁移（`openai_tool_images.go`、`openai_message_links.go`）。
  - OpenAI compat `xhigh` thinking 默认值、流式 null usage 处理。
  - Codex OAuth 失效 token failover、GPT-5.5 Codex 支持、websocket body log 增长上限、字符串 system prompt 保留。
  - Claude 保留上游 error metadata。
  - per-model thinking override（`claude-api-key` 模型）。
  - `host.docker.internal` 网关映射。

## 目标架构

- 本地 `main` 功能代码对齐 `upstream/main` `v8.0.43`，模块路径改为 `github.com/router-for-me/CLIProxyAPI/v8`，Go `1.26.10`。
- 保护路径（deploy/CI/Docker）保持本地内容不变。
- CPA-Manager 默认值保持本地指向 `seakee/CPA-Manager-Plus`。
- 本地 provider/协议定制在 v8 新结构上等价复现；已被上游独立修复的本地补丁直接采用上游实现（例如 antigravity array schema → 上游 `b5cde4ba`）。
- 仓库可编译可测试：`go build ./cmd/server`、`go test ./...`。

## 影响范围

- 文件：
  - 上游新增：约 `1810` 个改动文件中绝大部分为新增/改写，含 v8 迁移的全量 import 改写。
  - 本地新增且需改写 import：`24` 个纯本地 Go 文件仍引用 `/v7`。
  - 冲突高发：`config.example.yaml`、`internal/api/**`、`internal/config/*`、`internal/registry/*`、`internal/runtime/executor/*`（codex/openai_compat/xai）、`internal/watcher/**`、`sdk/**`。
- API：
  - 上游新增管理端点（auth files 刷新、Home/credential 并发、plugin payload 规则等）随合并进入。
  - `/v1`、`/v1beta` 兼容性由上游 translator/runtime 决定；本地 xAI / Command Code / DeepSeek 路由必须保持可用。
- 测试：
  - 上游新增大量 antigravity/translator/signature/session 测试随合并进入。
  - 本地 xai / commandcode / deepseek / thinking 测试需在新结构下通过。
