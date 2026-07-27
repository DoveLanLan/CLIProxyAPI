package management

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
)

type xaiProxyPoolOperator interface {
	XAIProxyPoolStatus(context.Context) helps.XAIProxyPoolStatus
	RefreshXAIProxyProviders(context.Context) error
	RotateXAIProxyLane(context.Context, string) error
	CheckXAIProxyLane(context.Context, string) (bool, error)
	QuarantineXAIProxyIP(context.Context, string) error
	UnquarantineXAIProxyIP(context.Context, string) error
}

func (h *Handler) xaiProxyPoolOperator() (xaiProxyPoolOperator, bool) {
	if h == nil || h.authManager == nil {
		return nil, false
	}
	executor, okExecutor := h.authManager.Executor("xai")
	if !okExecutor || executor == nil {
		return nil, false
	}
	operator, okOperator := executor.(xaiProxyPoolOperator)
	return operator, okOperator && operator != nil
}

func (h *Handler) GetXAIProxyPoolStatus(c *gin.Context) {
	operator, okOperator := h.xaiProxyPoolOperator()
	if !okOperator {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "xai proxy pool executor is unavailable"})
		return
	}
	status := operator.XAIProxyPoolStatus(c.Request.Context())
	if !status.Enabled {
		c.JSON(http.StatusConflict, gin.H{"error": "xai proxy pool is disabled", "status": status})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *Handler) RefreshXAIProxyProviders(c *gin.Context) {
	operator, okOperator := h.requireXAIProxyPool(c)
	if !okOperator {
		return
	}
	if errRefresh := operator.RefreshXAIProxyProviders(c.Request.Context()); errRefresh != nil {
		writeXAIProxyPoolError(c, errRefresh)
		return
	}
	c.JSON(http.StatusOK, operator.XAIProxyPoolStatus(c.Request.Context()))
}

func (h *Handler) RotateXAIProxyLane(c *gin.Context) {
	operator, okOperator := h.requireXAIProxyPool(c)
	if !okOperator {
		return
	}
	lane := strings.TrimSpace(c.Param("lane"))
	if lane == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lane is required"})
		return
	}
	if errRotate := operator.RotateXAIProxyLane(c.Request.Context(), lane); errRotate != nil {
		writeXAIProxyPoolError(c, errRotate)
		return
	}
	c.JSON(http.StatusOK, operator.XAIProxyPoolStatus(c.Request.Context()))
}

func (h *Handler) CheckXAIProxyLane(c *gin.Context) {
	operator, okOperator := h.requireXAIProxyPool(c)
	if !okOperator {
		return
	}
	lane := strings.TrimSpace(c.Param("lane"))
	if lane == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lane is required"})
		return
	}
	healthy, errCheck := operator.CheckXAIProxyLane(c.Request.Context(), lane)
	if errCheck != nil {
		writeXAIProxyPoolError(c, errCheck)
		return
	}
	c.JSON(http.StatusOK, gin.H{"lane": lane, "healthy": healthy, "status": operator.XAIProxyPoolStatus(c.Request.Context())})
}

func (h *Handler) QuarantineXAIProxyIP(c *gin.Context) {
	operator, okOperator := h.requireXAIProxyPool(c)
	if !okOperator {
		return
	}
	var body struct {
		IP string `json:"ip"`
	}
	if errBind := c.ShouldBindJSON(&body); errBind != nil || strings.TrimSpace(body.IP) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid ip is required"})
		return
	}
	if errQuarantine := operator.QuarantineXAIProxyIP(c.Request.Context(), body.IP); errQuarantine != nil {
		writeXAIProxyPoolError(c, errQuarantine)
		return
	}
	c.JSON(http.StatusOK, operator.XAIProxyPoolStatus(c.Request.Context()))
}

func (h *Handler) UnquarantineXAIProxyIP(c *gin.Context) {
	operator, okOperator := h.requireXAIProxyPool(c)
	if !okOperator {
		return
	}
	ip := strings.TrimSpace(c.Param("ip"))
	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ip is required"})
		return
	}
	if errUnquarantine := operator.UnquarantineXAIProxyIP(c.Request.Context(), ip); errUnquarantine != nil {
		writeXAIProxyPoolError(c, errUnquarantine)
		return
	}
	c.JSON(http.StatusOK, operator.XAIProxyPoolStatus(c.Request.Context()))
}

func (h *Handler) requireXAIProxyPool(c *gin.Context) (xaiProxyPoolOperator, bool) {
	operator, okOperator := h.xaiProxyPoolOperator()
	if !okOperator {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "xai proxy pool executor is unavailable"})
		return nil, false
	}
	if !operator.XAIProxyPoolStatus(c.Request.Context()).Enabled {
		c.JSON(http.StatusConflict, gin.H{"error": "xai proxy pool is disabled"})
		return nil, false
	}
	return operator, true
}

func writeXAIProxyPoolError(c *gin.Context, err error) {
	status := xaiProxyPoolHTTPStatus(err, http.StatusBadGateway)
	c.JSON(status, gin.H{"error": err.Error()})
}

func xaiProxyPoolHTTPStatus(err error, fallback int) int {
	type statusCoder interface{ StatusCode() int }
	var statusError statusCoder
	if errors.As(err, &statusError) && statusError != nil {
		if candidate := statusError.StatusCode(); candidate >= 400 && candidate <= 599 {
			return candidate
		}
	}
	return fallback
}
