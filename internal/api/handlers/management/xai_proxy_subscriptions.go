package management

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
)

const xaiProxySubscriptionRequestMaxBytes = 16 << 10

type xaiProxySubscriptionOperator interface {
	XAIProxySubscriptions(context.Context) helps.XAIProxySubscriptionList
	CreateXAIProxySubscription(context.Context, uint64, helps.XAIProxySubscriptionCreate) (helps.XAIProxySubscriptionList, error)
	UpdateXAIProxySubscription(context.Context, uint64, string, helps.XAIProxySubscriptionUpdate) (helps.XAIProxySubscriptionList, error)
	DeleteXAIProxySubscription(context.Context, uint64, string) (helps.XAIProxySubscriptionList, error)
	CheckXAIProxySubscription(context.Context, string) (helps.XAIProxySubscriptionStatus, error)
}

func (h *Handler) xaiProxySubscriptionOperator() (xaiProxySubscriptionOperator, bool) {
	if h == nil || h.authManager == nil {
		return nil, false
	}
	executor, okExecutor := h.authManager.Executor("xai")
	if !okExecutor || executor == nil {
		return nil, false
	}
	operator, okOperator := executor.(xaiProxySubscriptionOperator)
	return operator, okOperator && operator != nil
}

func (h *Handler) GetXAIProxySubscriptions(c *gin.Context) {
	operator, okOperator := h.requireXAIProxySubscriptions(c)
	if !okOperator {
		return
	}
	writeXAIProxySubscriptionList(c, http.StatusOK, operator.XAIProxySubscriptions(c.Request.Context()))
}

func (h *Handler) CreateXAIProxySubscription(c *gin.Context) {
	operator, okOperator := h.requireXAIProxySubscriptions(c)
	if !okOperator {
		return
	}
	revision, okRevision := requireXAIProxySubscriptionRevision(c)
	if !okRevision {
		return
	}
	var body struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Enabled *bool  `json:"enabled"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, xaiProxySubscriptionRequestMaxBytes)
	if errBind := c.ShouldBindJSON(&body); errBind != nil || strings.TrimSpace(body.Name) == "" || strings.TrimSpace(body.URL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "name and write-only url are required"})
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	result, errCreate := operator.CreateXAIProxySubscription(c.Request.Context(), revision, helps.XAIProxySubscriptionCreate{
		Name: body.Name, URL: body.URL, Enabled: enabled,
	})
	if errCreate != nil {
		writeXAIProxySubscriptionError(c, errCreate, &result)
		return
	}
	writeXAIProxySubscriptionList(c, http.StatusCreated, result)
}

func (h *Handler) UpdateXAIProxySubscription(c *gin.Context) {
	operator, okOperator := h.requireXAIProxySubscriptions(c)
	if !okOperator {
		return
	}
	revision, okRevision := requireXAIProxySubscriptionRevision(c)
	if !okRevision {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_provider_name", "error": "subscription provider name is required"})
		return
	}
	var body struct {
		URL     *string `json:"url"`
		Enabled *bool   `json:"enabled"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, xaiProxySubscriptionRequestMaxBytes)
	if errBind := c.ShouldBindJSON(&body); errBind != nil || (body.URL == nil && body.Enabled == nil) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_request", "error": "url or enabled is required"})
		return
	}
	result, errUpdate := operator.UpdateXAIProxySubscription(c.Request.Context(), revision, name, helps.XAIProxySubscriptionUpdate{
		URL: body.URL, Enabled: body.Enabled,
	})
	if errUpdate != nil {
		writeXAIProxySubscriptionError(c, errUpdate, &result)
		return
	}
	writeXAIProxySubscriptionList(c, http.StatusOK, result)
}

func (h *Handler) DeleteXAIProxySubscription(c *gin.Context) {
	operator, okOperator := h.requireXAIProxySubscriptions(c)
	if !okOperator {
		return
	}
	revision, okRevision := requireXAIProxySubscriptionRevision(c)
	if !okRevision {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_provider_name", "error": "subscription provider name is required"})
		return
	}
	result, errDelete := operator.DeleteXAIProxySubscription(c.Request.Context(), revision, name)
	if errDelete != nil {
		writeXAIProxySubscriptionError(c, errDelete, &result)
		return
	}
	writeXAIProxySubscriptionList(c, http.StatusOK, result)
}

func (h *Handler) CheckXAIProxySubscription(c *gin.Context) {
	operator, okOperator := h.requireXAIProxySubscriptions(c)
	if !okOperator {
		return
	}
	name := strings.TrimSpace(c.Param("name"))
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_provider_name", "error": "subscription provider name is required"})
		return
	}
	result, errCheck := operator.CheckXAIProxySubscription(c.Request.Context(), name)
	if errCheck != nil {
		writeXAIProxySubscriptionError(c, errCheck)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) requireXAIProxySubscriptions(c *gin.Context) (xaiProxySubscriptionOperator, bool) {
	operator, okOperator := h.xaiProxySubscriptionOperator()
	if !okOperator {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "subscription_management_unavailable", "error": "xAI subscription management executor is unavailable"})
		return nil, false
	}
	status := operator.XAIProxySubscriptions(c.Request.Context())
	if !status.Enabled {
		c.JSON(http.StatusConflict, gin.H{"code": "subscription_management_disabled", "error": "xAI subscription management is disabled"})
		return nil, false
	}
	return operator, true
}

func requireXAIProxySubscriptionRevision(c *gin.Context) (uint64, bool) {
	raw := strings.TrimSpace(c.GetHeader("If-Match"))
	if raw == "" {
		c.JSON(http.StatusPreconditionRequired, gin.H{"code": "revision_required", "error": "If-Match registry revision is required"})
		return 0, false
	}
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' || strings.Contains(raw[1:len(raw)-1], `"`) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_revision", "error": "If-Match registry revision is invalid"})
		return 0, false
	}
	revision, errParse := strconv.ParseUint(raw[1:len(raw)-1], 10, 64)
	if errParse != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_revision", "error": "If-Match registry revision is invalid"})
		return 0, false
	}
	return revision, true
}

func writeXAIProxySubscriptionList(c *gin.Context, status int, result helps.XAIProxySubscriptionList) {
	c.Header("ETag", `"`+strconv.FormatUint(result.Revision, 10)+`"`)
	c.JSON(status, result)
}

func writeXAIProxySubscriptionError(c *gin.Context, err error, result ...*helps.XAIProxySubscriptionList) {
	status := xaiProxyPoolHTTPStatus(err, http.StatusBadGateway)
	code := "subscription_operation_failed"
	message := "xAI subscription operation failed"
	var subscriptionErr *helps.XAIProxySubscriptionError
	if errors.As(err, &subscriptionErr) && subscriptionErr != nil {
		if strings.TrimSpace(subscriptionErr.Code) != "" {
			code = subscriptionErr.Code
		}
		if strings.TrimSpace(subscriptionErr.Message) != "" {
			message = subscriptionErr.Message
		}
	}
	if len(result) > 0 && result[0] != nil {
		c.Header("ETag", `"`+strconv.FormatUint(result[0].Revision, 10)+`"`)
	}
	c.JSON(status, gin.H{"code": code, "error": message})
}
