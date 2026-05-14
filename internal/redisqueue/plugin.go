package redisqueue

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	internallogging "github.com/router-for-me/CLIProxyAPI/v6/internal/logging"
	coreusage "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
)

func init() {
	coreusage.RegisterPlugin(&usageQueuePlugin{})
}

type usageQueuePlugin struct{}

func (p *usageQueuePlugin) HandleUsage(ctx context.Context, record coreusage.Record) {
	if p == nil || !Enabled() || !UsageStatisticsEnabled() {
		return
	}

	timestamp := record.RequestedAt
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	modelName := strings.TrimSpace(record.Model)
	if modelName == "" {
		modelName = "unknown"
	}
	aliasName := strings.TrimSpace(record.Alias)
	if aliasName == "" {
		aliasName = modelName
	}
	provider := strings.TrimSpace(record.Provider)
	if provider == "" {
		provider = "unknown"
	}
	authType := strings.TrimSpace(record.AuthType)
	if authType == "" {
		authType = "unknown"
	}

	tokens := tokenStats{
		InputTokens:     record.Detail.InputTokens,
		OutputTokens:    record.Detail.OutputTokens,
		ReasoningTokens: record.Detail.ReasoningTokens,
		CachedTokens:    record.Detail.CachedTokens,
		TotalTokens:     record.Detail.TotalTokens,
	}
	if tokens.TotalTokens == 0 {
		tokens.TotalTokens = tokens.InputTokens + tokens.OutputTokens + tokens.ReasoningTokens
	}
	if tokens.TotalTokens == 0 {
		tokens.TotalTokens = tokens.InputTokens + tokens.OutputTokens + tokens.ReasoningTokens + tokens.CachedTokens
	}

	failed := record.Failed
	if !failed {
		failed = !resolveSuccess(ctx)
	}

	payload, err := json.Marshal(queuedUsageDetail{
		requestDetail: requestDetail{
			Timestamp: timestamp,
			LatencyMs: record.Latency.Milliseconds(),
			Source:    record.Source,
			AuthIndex: record.AuthIndex,
			Tokens:    tokens,
			Failed:    failed,
		},
		Provider:  provider,
		Model:     modelName,
		Alias:     aliasName,
		Endpoint:  resolveEndpoint(ctx),
		AuthType:  authType,
		APIKey:    strings.TrimSpace(record.APIKey),
		RequestID: resolveRequestID(ctx),
	})
	if err != nil {
		return
	}
	Enqueue(payload)
}

type queuedUsageDetail struct {
	requestDetail
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Alias     string `json:"alias"`
	Endpoint  string `json:"endpoint"`
	AuthType  string `json:"auth_type"`
	APIKey    string `json:"api_key"`
	RequestID string `json:"request_id"`
}

type requestDetail struct {
	Timestamp time.Time  `json:"timestamp"`
	LatencyMs int64      `json:"latency_ms"`
	Source    string     `json:"source"`
	AuthIndex string     `json:"auth_index"`
	Tokens    tokenStats `json:"tokens"`
	Failed    bool       `json:"failed"`
}

type tokenStats struct {
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	TotalTokens     int64 `json:"total_tokens"`
}

func resolveRequestID(ctx context.Context) string {
	if requestID := strings.TrimSpace(internallogging.GetRequestID(ctx)); requestID != "" {
		return requestID
	}
	if ginCtx := ginContext(ctx); ginCtx != nil {
		return strings.TrimSpace(internallogging.GetGinRequestID(ginCtx))
	}
	return ""
}

func resolveSuccess(ctx context.Context) bool {
	if status := internallogging.GetResponseStatus(ctx); status > 0 {
		return status < httpStatusBadRequest
	}
	ginCtx := ginContext(ctx)
	if ginCtx == nil || ginCtx.Writer == nil {
		return true
	}
	status := ginCtx.Writer.Status()
	if status == 0 {
		return true
	}
	return status < httpStatusBadRequest
}

func resolveEndpoint(ctx context.Context) string {
	if endpoint := strings.TrimSpace(internallogging.GetEndpoint(ctx)); endpoint != "" {
		return endpoint
	}
	ginCtx := ginContext(ctx)
	if ginCtx == nil || ginCtx.Request == nil {
		return ""
	}
	path := strings.TrimSpace(ginCtx.FullPath())
	if path == "" && ginCtx.Request.URL != nil {
		path = strings.TrimSpace(ginCtx.Request.URL.Path)
	}
	method := strings.TrimSpace(ginCtx.Request.Method)
	if method != "" && path != "" {
		return method + " " + path
	}
	return path
}

func ginContext(ctx context.Context) *gin.Context {
	if ctx == nil {
		return nil
	}
	ginCtx, _ := ctx.Value("gin").(*gin.Context)
	return ginCtx
}

const httpStatusBadRequest = 400
