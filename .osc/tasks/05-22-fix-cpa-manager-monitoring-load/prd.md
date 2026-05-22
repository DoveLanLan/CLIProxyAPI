# Bugfix: Fix CPA-Manager monitoring load

## 问题描述

CPA-Manager VPS monitoring page remains behind `加载中...` and does not render usage rows.

## 复现步骤
1. Open `http://100.67.99.9:18318/management.html#/monitoring`.
2. Observe successful metadata/status requests in browser devtools.
3. Observe no usage data rendered and the loading overlay remains.

## 期望行为

Monitoring data should load or fail visibly within the frontend timeout.

## 实际行为

`/status` returns, but `/v0/management/usage` and `/v0/management/usage/export` time out. Auto-refresh starts new usage requests before older ones settle, so the loading overlay remains active.

## 根因分析

CPA-Manager Usage Service reads a large recent-event window for the dashboard. The deployed service is configured with the default query limit, and the endpoint does not return before the frontend timeout on the VPS.

## 修复方案

Set a small bounded `USAGE_QUERY_LIMIT` for the deployed CPA-Manager container and document the override.

## 回归测试
- [x] `docker compose config` renders the new environment variable.
- [x] `git diff --check`.
- [x] `go build -o test-output ./cmd/server && rm test-output`.
