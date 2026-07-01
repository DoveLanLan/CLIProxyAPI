# Refactor: Sync upstream v7.2.48 preserving deploy and CPA-Manager

## 现状

- 本地 `main` 与 `upstream/main`（`router-for-me/CLIProxyAPI`）已显著分叉：
  - 上次同步点：`21fad9db`（2026-05-21，对应 v7.2.x 早期，由任务 `05-22-merge-upstream-non-docker-changes` 落地）。
  - 上游当前 HEAD：`956ce7cf`（2026-06-30，`v7.2.48`）。
  - 待合入上游提交数：`350`（`21fad9db..upstream/main`），改动文件 `664`。
  - 本地落后上游主要体现在新模块：`internal/{pluginhost,pluginstore,signature,safemode,homeplugins,htmlsanitize,httpfetch}`、`sdk/{pluginabi,pluginapi,pluginhost,pluginstore}`、`cmd/fetch_codex_models`、`examples/plugin/**`。
- 本地分叉点 `f1ba6151`（v6.9.36）以来，自有改动 `52` 个提交，分布在：
  - 部署/运维：`.github/workflows/**`、`Dockerfile`、`docker-compose*.yml`、`deploy/**`、`.goreleaser.yml`、`docker-build.sh`。
  - CPA-Manager 集成：`internal/config/config.go`（`DefaultPanelGitHubRepository = seakee/CPA-Manager`）、`internal/managementasset/updater.go`（默认 release/fallback URL）、`config.example.yaml`、`deploy/compose.production.yml`、`docker-compose*.yml`、`.github/workflows/update-cpa-manager-image.yml`。
  - 协议/运行时定制：Codex OAuth 失效 token failover、OpenAI 兼容 xhigh thinking 默认值、OpenAI 流式 null usage chunks、DeepSeek 模型与 reasoning echo 归一化、GPT-5.5 Codex 支持、`host.docker.internal` 网关映射、websocket body log 增长上限、字符串 system prompt 保留。

## 目标架构

- `main` 在功能代码上对齐 `upstream/main` `v7.2.48`（`956ce7cf`），获得插件系统（pluginhost/pluginstore/pluginabi/pluginapi）、signature 校验、safemode、homeplugins、htmlsanitize、httpfetch、新增 translator/runtime/registry/auth 能力、`gpt-image-1.5`、video 处理、`disable-cooling`、`max` reasoning depth、`ResetQuota` 等。
- 部署与 CPA-Manager 定制保持本地所有：上游对 `.github/**`、`Dockerfile`、`docker-*`、`deploy/**`、`.goreleaser.yml` 的改动一律不采用；`internal/config/config.go` 与 `internal/managementasset/updater.go` 中的 CPA-Manager 默认值保持本地。
- 模块路径仍为 `github.com/router-for-me/CLIProxyAPI/v7`，Go 1.26，可编译可测试。

## 影响范围
- 文件:
  - 新增（上游）：`internal/{pluginhost,pluginstore,signature,safemode,homeplugins,htmlsanitize,httpfetch}/**`、`sdk/{pluginabi,pluginapi,pluginhost,pluginstore}/**`、`cmd/fetch_codex_models/**`、`examples/plugin/**`。
  - 冲突高发：`config.example.yaml`、`cmd/server/main.go`、`internal/api/handlers/management/*`、`internal/runtime/executor/*`、`internal/translator/**`、`internal/auth/**`、`internal/registry/*`、`internal/config/config.go`、`internal/managementasset/updater.go`、`README*`、`go.mod/go.sum`。
  - 保护（不采用上游）：`.github/**`、`Dockerfile`、`.dockerignore`、`docker-build.*`、`docker-compose*.yml`、`deploy/**`、`.goreleaser.yml`。
- API:
  - 上游新增管理 API（`ResetQuota`、插件安装/删除/配置、pluginhost 相关端点）随合并进入；公开 `/v1`、`/v1beta` 兼容性由上游 translator/runtime 改动决定。
  - 本地 CPA-Manager 面板默认仓库与 release URL 不变。
- 测试:
  - 上游新增大量 pluginhost/pluginstore/signature/translator 测试随合并进入。
  - 本地关键测试须保留并通过：Codex invalidated OAuth failover（`max-retry-credentials=1`）、OpenAI 兼容 xhigh thinking 默认、OpenAI 流式 null usage、DeepSeek 模型与 reasoning echo、GPT-5.5 Codex、字符串 system prompt。

## 回归范围

- 必过门禁：`go build -o test-output ./cmd/server`、`go test ./...`、`gofmt`、`git diff --check`、无冲突标记。
- 保护路径无 diff：`git status --short -- .github Dockerfile .dockerignore docker-build.sh docker-build.ps1 'docker-compose*.yml' .env.cluster.example deploy .goreleaser.yml`。
- 本地行为保留：Codex 失效 token failover、xhigh thinking 默认、null usage chunk、DeepSeek 模型、GPT-5.5 Codex free-tier 过滤、CPA-Manager 默认仓库/URL。
- 上游新行为冒烟（部署前）：插件安装/删除/热重载、`/v1/responses` Codex WS↔SSE、image/video 路由、`disable-cooling`、`max` reasoning、`ResetQuota`。

## 分步计划
1. 创建任务与 change-workflow 产物（proposal/spec/tasks/regression/rollback/summary）。
2. 抓取上游、确定保护路径集与三向合并策略；先在临时分支验证。
3. 应用上游 v7.2.48（`21fad9db..upstream/main`）非保护路径改动，三向冲突解决，功能冲突优先上游。
4. 还原保护路径为本地 `HEAD`；保留 CPA-Manager 默认值；保留本地行为补丁并适配上游新结构。
5. 运行质量门禁（build/test/gofmt/diff-check/保护路径检查）。
6. 落盘 closure 产物与 quality-gate，本地完成；推送与部署另议。
