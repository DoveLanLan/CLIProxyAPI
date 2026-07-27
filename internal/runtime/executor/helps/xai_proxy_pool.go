package helps

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	log "github.com/sirupsen/logrus"
)

const xaiProxyPoolResponseMaxBytes = 1 << 20

// XAIProxyPoolError is availability-neutral for credential scheduling. It
// represents a request-scoped proxy-pool condition, not an auth failure.
type XAIProxyPoolError struct {
	Message    string
	Retry      time.Duration
	HTTPStatus int
}

func (e *XAIProxyPoolError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *XAIProxyPoolError) StatusCode() int {
	if e != nil && e.HTTPStatus >= 400 && e.HTTPStatus <= 599 {
		return e.HTTPStatus
	}
	return http.StatusServiceUnavailable
}

func (e *XAIProxyPoolError) IsRequestScoped() bool { return true }

func (e *XAIProxyPoolError) RetryAfter() *time.Duration {
	if e == nil || e.Retry <= 0 {
		return nil
	}
	retry := e.Retry
	return &retry
}

func (e *XAIProxyPoolError) Headers() http.Header {
	if e == nil || e.Retry <= 0 {
		return nil
	}
	seconds := int64((e.Retry-1)/time.Second) + 1
	headers := make(http.Header)
	headers.Set("Retry-After", strconv.FormatInt(seconds, 10))
	return headers
}

func (e *XAIProxyPoolError) ManagedHeaders() http.Header { return e.Headers() }

// XAIProxyRoute is the non-sensitive routing decision used by the xAI executor.
type XAIProxyRoute struct {
	LaneName string `json:"lane"`
	ProxyURL string `json:"-"`
	Selector string `json:"-"`
	Node     string `json:"node"`
	Provider string `json:"provider"`
	EgressIP string `json:"egress_ip"`
	Probe    bool   `json:"probe"`
}

type XAIProxyPoolStatus struct {
	Enabled          bool                           `json:"enabled"`
	Ready            bool                           `json:"ready"`
	RolloutPercent   int                            `json:"rollout_percent"`
	LastError        string                         `json:"last_error,omitempty"`
	LastRefresh      time.Time                      `json:"last_refresh,omitempty"`
	Lanes            []XAIProxyPoolLaneStatus       `json:"lanes"`
	Providers        []XAIProxyPoolProviderStatus   `json:"providers"`
	IPQuarantines    []XAIProxyPoolQuarantineStatus `json:"ip_quarantines"`
	NodeQuarantines  []XAIProxyPoolQuarantineStatus `json:"node_quarantines"`
	Counters         XAIProxyPoolCounters           `json:"counters"`
	ConfiguredLanes  int                            `json:"configured_lanes"`
	AvailableNodes   int                            `json:"available_nodes"`
	StateFileEnabled bool                           `json:"state_file_enabled"`
}

type XAIProxyPoolLaneStatus struct {
	Name          string    `json:"name"`
	ProxyEndpoint string    `json:"proxy_endpoint"`
	Selector      string    `json:"selector"`
	Node          string    `json:"node,omitempty"`
	Provider      string    `json:"provider,omitempty"`
	EgressIP      string    `json:"egress_ip,omitempty"`
	Ready         bool      `json:"ready"`
	LastChanged   time.Time `json:"last_changed,omitempty"`
}

type XAIProxyPoolProviderStatus struct {
	Name           string `json:"name"`
	AvailableNodes int    `json:"available_nodes"`
}

type XAIProxyPoolQuarantineStatus struct {
	Value     string    `json:"value"`
	Reason    string    `json:"reason"`
	ExpiresAt time.Time `json:"expires_at"`
}

type XAIProxyPoolCounters struct {
	Requests              uint64 `json:"requests"`
	QueueRejected         uint64 `json:"queue_rejected"`
	Exact402              uint64 `json:"exact_402"`
	ABSuccess             uint64 `json:"ab_success"`
	ABCredentialFailure   uint64 `json:"ab_credential_failure"`
	ABUnavailable         uint64 `json:"ab_unavailable"`
	Rotations             uint64 `json:"rotations"`
	PreconnectFailures    uint64 `json:"preconnect_failures"`
	MidResponseFailures   uint64 `json:"mid_response_failures"`
	ProviderRefreshErrors uint64 `json:"provider_refresh_errors"`
}

type XAIProxySubscriptionError struct {
	Code    string
	Message string
	Status  int
}

func (e *XAIProxySubscriptionError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *XAIProxySubscriptionError) StatusCode() int {
	if e != nil && e.Status >= 400 && e.Status <= 599 {
		return e.Status
	}
	return http.StatusBadGateway
}

func (e *XAIProxySubscriptionError) IsRequestScoped() bool { return true }

type XAIProxySubscriptionList struct {
	Enabled       bool                         `json:"enabled"`
	Ready         bool                         `json:"ready"`
	Revision      uint64                       `json:"revision"`
	LastErrorCode string                       `json:"last_error_code,omitempty"`
	Subscriptions []XAIProxySubscriptionStatus `json:"subscriptions"`
}

type XAIProxySubscriptionStatus struct {
	Name          string    `json:"name"`
	Enabled       bool      `json:"enabled"`
	Fingerprint   string    `json:"fingerprint"`
	SourceHost    string    `json:"source_host"`
	State         string    `json:"state"`
	NodeCount     int       `json:"node_count"`
	LastChecked   time.Time `json:"last_checked,omitempty"`
	LastErrorCode string    `json:"last_error_code,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type XAIProxySubscriptionCreate struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type XAIProxySubscriptionUpdate struct {
	URL     *string `json:"url,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

// XAIProxyProbeLease represents a single-use alternate-egress reservation.
type XAIProxyProbeLease interface {
	CurrentRoute() XAIProxyRoute
	AlternateRoute() XAIProxyRoute
	ConfirmIPBlock(context.Context) error
	CredentialFailure()
	Unavailable()
	Release()
}

// XAIProxyPool is a client for the standalone EgressProxyPool control API.
type XAIProxyPool struct {
	enabled    bool
	baseURL    string
	token      string
	signingKey []byte
	client     *http.Client
	transport  *http.Transport
	initErr    error
}

type xaiProxyRouteWire struct {
	LaneName string `json:"lane"`
	ProxyURL string `json:"proxy_url"`
	Selector string `json:"selector"`
	Node     string `json:"node"`
	Provider string `json:"provider"`
	EgressIP string `json:"egress_ip"`
	Probe    bool   `json:"probe"`
}

func (route xaiProxyRouteWire) public() XAIProxyRoute {
	return XAIProxyRoute{
		LaneName: route.LaneName, ProxyURL: route.ProxyURL, Selector: route.Selector,
		Node: route.Node, Provider: route.Provider, EgressIP: route.EgressIP, Probe: route.Probe,
	}
}

func routeWire(route XAIProxyRoute) xaiProxyRouteWire {
	return xaiProxyRouteWire{
		LaneName: route.LaneName, ProxyURL: route.ProxyURL, Selector: route.Selector,
		Node: route.Node, Provider: route.Provider, EgressIP: route.EgressIP, Probe: route.Probe,
	}
}

type remoteProbeLease struct {
	pool      *XAIProxyPool
	ctx       context.Context
	id        string
	current   XAIProxyRoute
	alternate XAIProxyRoute
	once      sync.Once
}

// NewXAIProxyPool builds a remote control-plane client. Invalid enabled
// settings remain fail-closed and are surfaced when the first route is needed.
func NewXAIProxyPool(cfg config.XAIProxyPoolConfig) *XAIProxyPool {
	client := &XAIProxyPool{enabled: cfg.Enabled}
	if !cfg.Enabled {
		return client
	}
	parsed, errURL := url.Parse(strings.TrimSpace(cfg.ServiceURL))
	if errURL != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		client.initErr = errors.New("standalone proxy-pool service URL is invalid")
		return client
	}
	tokenBytes, errToken := os.ReadFile(strings.TrimSpace(cfg.ServiceTokenFile))
	if errToken != nil || strings.TrimSpace(string(tokenBytes)) == "" {
		client.initErr = errors.New("standalone proxy-pool service token is unavailable")
		return client
	}
	transport, okTransport := http.DefaultTransport.(*http.Transport)
	if !okTransport || transport == nil {
		client.initErr = errors.New("default HTTP transport is unavailable")
		return client
	}
	client.baseURL = strings.TrimRight(parsed.String(), "/")
	client.token = strings.TrimSpace(string(tokenBytes))
	client.signingKey = []byte(client.token)
	client.transport = transport.Clone()
	client.transport.Proxy = nil
	client.client = &http.Client{Transport: client.transport}
	return client
}

func (p *XAIProxyPool) Route(ctx context.Context, authID string) (XAIProxyRoute, bool, error) {
	if p == nil || !p.enabled {
		return XAIProxyRoute{}, false, nil
	}
	if strings.TrimSpace(authID) == "" {
		return XAIProxyRoute{}, true, poolUnavailable("xai proxy pool requires a stable auth ID")
	}
	if errReady := p.readyError(); errReady != nil {
		return XAIProxyRoute{}, true, errReady
	}
	mac := hmac.New(sha256.New, p.signingKey)
	_, _ = mac.Write([]byte(authID))
	var result struct {
		Enrolled bool              `json:"enrolled"`
		Route    xaiProxyRouteWire `json:"route"`
	}
	if errRequest := p.doJSON(ctx, http.MethodPost, "/v1/routes", map[string]string{
		"key": hex.EncodeToString(mac.Sum(nil)),
	}, &result, false, 0); errRequest != nil {
		return XAIProxyRoute{}, true, errRequest
	}
	return result.Route.public(), result.Enrolled, nil
}

func (p *XAIProxyPool) AcquireProbe(ctx context.Context, current XAIProxyRoute) (XAIProxyProbeLease, error) {
	if errReady := p.readyError(); errReady != nil {
		return nil, errReady
	}
	var result struct {
		LeaseID string            `json:"lease_id"`
		Current xaiProxyRouteWire `json:"current"`
		Route   xaiProxyRouteWire `json:"route"`
	}
	if errRequest := p.doJSON(ctx, http.MethodPost, "/v1/probes", map[string]any{
		"current": routeWire(current),
	}, &result, false, 0); errRequest != nil {
		return nil, errRequest
	}
	if strings.TrimSpace(result.LeaseID) == "" {
		return nil, poolUnavailable("xai proxy pool returned an invalid probe lease")
	}
	return &remoteProbeLease{
		pool: p, ctx: ctx, id: result.LeaseID,
		current: result.Current.public(), alternate: result.Route.public(),
	}, nil
}

func (l *remoteProbeLease) CurrentRoute() XAIProxyRoute {
	if l == nil {
		return XAIProxyRoute{}
	}
	return l.current
}

func (l *remoteProbeLease) AlternateRoute() XAIProxyRoute {
	if l == nil {
		return XAIProxyRoute{}
	}
	return l.alternate
}

func (l *remoteProbeLease) ConfirmIPBlock(ctx context.Context) (resultErr error) {
	if l == nil || l.pool == nil {
		return poolUnavailable("xai proxy pool probe lease is unavailable")
	}
	run := false
	l.once.Do(func() {
		run = true
		resultErr = l.pool.doJSON(ctx, http.MethodPost, "/v1/probes/"+url.PathEscape(l.id)+"/confirm-ip-block", nil, nil, false, 0)
	})
	if !run && resultErr == nil {
		return poolUnavailable("xai proxy pool probe lease was already completed")
	}
	return resultErr
}

func (l *remoteProbeLease) CredentialFailure() {
	l.finish("credential-failure", http.MethodPost)
}

func (l *remoteProbeLease) Unavailable() {
	l.finish("unavailable", http.MethodPost)
}

func (l *remoteProbeLease) Release() {
	l.finish("", http.MethodDelete)
}

func (l *remoteProbeLease) finish(action string, method string) {
	if l == nil || l.pool == nil {
		return
	}
	l.once.Do(func() {
		path := "/v1/probes/" + url.PathEscape(l.id)
		if action != "" {
			path += "/" + action
		}
		if errFinish := l.pool.doJSON(l.ctx, method, path, nil, nil, false, 0); errFinish != nil {
			log.WithError(errFinish).Debug("xai proxy-pool probe outcome could not be reported")
		}
	})
}

func (p *XAIProxyPool) HandlePreconnectFailure(ctx context.Context, route XAIProxyRoute) (XAIProxyRoute, bool, error) {
	if errReady := p.readyError(); errReady != nil {
		return XAIProxyRoute{}, false, errReady
	}
	var result struct {
		Retry bool              `json:"retry"`
		Route xaiProxyRouteWire `json:"route"`
	}
	if errRequest := p.doJSON(ctx, http.MethodPost, "/v1/events/preconnect-failure", map[string]any{
		"route": routeWire(route),
	}, &result, false, 0); errRequest != nil {
		return XAIProxyRoute{}, false, errRequest
	}
	return result.Route.public(), result.Retry, nil
}

func (p *XAIProxyPool) ObserveMidResponseFailure(ctx context.Context, route XAIProxyRoute) {
	if p == nil || p.readyError() != nil {
		return
	}
	if errObserve := p.doJSON(ctx, http.MethodPost, "/v1/events/mid-response-failure", map[string]any{
		"route": routeWire(route),
	}, nil, false, 0); errObserve != nil {
		log.WithError(errObserve).Debug("xai proxy-pool mid-response failure could not be reported")
	}
}

func (p *XAIProxyPool) RecordExact402(ctx context.Context) {
	if p == nil || p.readyError() != nil {
		return
	}
	if errRecord := p.doJSON(ctx, http.MethodPost, "/v1/events/exact-402", nil, nil, false, 0); errRecord != nil {
		log.WithError(errRecord).Debug("xai proxy-pool exact 402 counter could not be reported")
	}
}

func (p *XAIProxyPool) Status(ctx context.Context) XAIProxyPoolStatus {
	if p == nil || !p.enabled {
		return XAIProxyPoolStatus{Lanes: []XAIProxyPoolLaneStatus{}, Providers: []XAIProxyPoolProviderStatus{}, IPQuarantines: []XAIProxyPoolQuarantineStatus{}, NodeQuarantines: []XAIProxyPoolQuarantineStatus{}}
	}
	if errReady := p.readyError(); errReady != nil {
		return unavailableStatus(errReady.Error())
	}
	var result XAIProxyPoolStatus
	if errRequest := p.doJSON(ctx, http.MethodGet, "/v1/status", nil, &result, false, 0); errRequest != nil {
		return unavailableStatus(errRequest.Error())
	}
	return result
}

func (p *XAIProxyPool) RefreshProviders(ctx context.Context) error {
	return p.control(ctx, http.MethodPost, "/v1/providers/refresh", nil)
}

func (p *XAIProxyPool) RotateLane(ctx context.Context, lane string) error {
	return p.control(ctx, http.MethodPost, "/v1/lanes/"+url.PathEscape(strings.TrimSpace(lane))+"/rotate", nil)
}

func (p *XAIProxyPool) CheckLane(ctx context.Context, lane string) (bool, error) {
	if errReady := p.readyError(); errReady != nil {
		return false, errReady
	}
	var result struct {
		Healthy bool `json:"healthy"`
	}
	errRequest := p.doJSON(ctx, http.MethodPost, "/v1/lanes/"+url.PathEscape(strings.TrimSpace(lane))+"/check", nil, &result, false, 0)
	return result.Healthy, errRequest
}

func (p *XAIProxyPool) QuarantineIP(ctx context.Context, ip string) error {
	return p.control(ctx, http.MethodPost, "/v1/quarantines", map[string]string{"ip": strings.TrimSpace(ip)})
}

func (p *XAIProxyPool) UnquarantineIP(ctx context.Context, ip string) error {
	return p.control(ctx, http.MethodDelete, "/v1/quarantines/"+url.PathEscape(strings.TrimSpace(ip)), nil)
}

func (p *XAIProxyPool) XAIProxySubscriptions(ctx context.Context) XAIProxySubscriptionList {
	if p == nil || !p.enabled {
		return XAIProxySubscriptionList{Subscriptions: []XAIProxySubscriptionStatus{}}
	}
	if errReady := p.readyError(); errReady != nil {
		return unavailableSubscriptions()
	}
	var result XAIProxySubscriptionList
	if errRequest := p.doJSON(ctx, http.MethodGet, "/v1/subscriptions", nil, &result, true, 0); errRequest != nil {
		return unavailableSubscriptions()
	}
	return result
}

func (p *XAIProxyPool) CreateXAIProxySubscription(ctx context.Context, revision uint64, input XAIProxySubscriptionCreate) (XAIProxySubscriptionList, error) {
	var result XAIProxySubscriptionList
	errRequest := p.doJSON(ctx, http.MethodPost, "/v1/subscriptions", input, &result, true, revision)
	return result, errRequest
}

func (p *XAIProxyPool) UpdateXAIProxySubscription(ctx context.Context, revision uint64, name string, input XAIProxySubscriptionUpdate) (XAIProxySubscriptionList, error) {
	var result XAIProxySubscriptionList
	errRequest := p.doJSON(ctx, http.MethodPut, "/v1/subscriptions/"+url.PathEscape(strings.TrimSpace(name)), input, &result, true, revision)
	return result, errRequest
}

func (p *XAIProxyPool) DeleteXAIProxySubscription(ctx context.Context, revision uint64, name string) (XAIProxySubscriptionList, error) {
	var result XAIProxySubscriptionList
	errRequest := p.doJSON(ctx, http.MethodDelete, "/v1/subscriptions/"+url.PathEscape(strings.TrimSpace(name)), nil, &result, true, revision)
	return result, errRequest
}

func (p *XAIProxyPool) CheckXAIProxySubscription(ctx context.Context, name string) (XAIProxySubscriptionStatus, error) {
	var result XAIProxySubscriptionStatus
	errRequest := p.doJSON(ctx, http.MethodPost, "/v1/subscriptions/"+url.PathEscape(strings.TrimSpace(name))+"/check", nil, &result, true, 0)
	return result, errRequest
}

func (p *XAIProxyPool) control(ctx context.Context, method string, path string, body any) error {
	if errReady := p.readyError(); errReady != nil {
		return errReady
	}
	return p.doJSON(ctx, method, path, body, nil, false, 0)
}

func (p *XAIProxyPool) readyError() error {
	if p == nil || !p.enabled {
		return poolUnavailable("xai proxy pool is unavailable")
	}
	if p.initErr != nil || p.client == nil {
		return poolUnavailable("standalone xai proxy pool is unavailable")
	}
	return nil
}

func (p *XAIProxyPool) doJSON(ctx context.Context, method string, path string, input any, output any, subscription bool, revision uint64) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var body io.Reader
	if input != nil {
		payload, errMarshal := json.Marshal(input)
		if errMarshal != nil {
			return p.remoteError(subscription, http.StatusInternalServerError, "request_encode_failed", 0)
		}
		body = bytes.NewReader(payload)
	}
	request, errRequest := http.NewRequestWithContext(ctx, method, p.baseURL+path, body)
	if errRequest != nil {
		return p.remoteError(subscription, http.StatusServiceUnavailable, "service_unavailable", 30*time.Second)
	}
	request.Header.Set("Authorization", "Bearer "+p.token)
	if input != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if revision > 0 || strings.HasPrefix(path, "/v1/subscriptions") && method != http.MethodGet && !strings.HasSuffix(path, "/check") {
		request.Header.Set("If-Match", `"`+strconv.FormatUint(revision, 10)+`"`)
	}
	response, errDo := p.client.Do(request)
	if errDo != nil {
		return p.remoteError(subscription, http.StatusServiceUnavailable, "service_unavailable", 30*time.Second)
	}
	defer func() {
		if errClose := response.Body.Close(); errClose != nil {
			log.WithError(errClose).Debug("close standalone proxy-pool response")
		}
	}()
	limited := io.LimitReader(response.Body, xaiProxyPoolResponseMaxBytes)
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if subscription {
			if list, okList := output.(*XAIProxySubscriptionList); okList && list != nil {
				if revisionValue, okRevision := parseRemoteRevision(response.Header.Get("ETag")); okRevision {
					list.Revision = revisionValue
				}
			}
		}
		var remote struct {
			Code              string `json:"code"`
			Error             string `json:"error"`
			RetryAfterSeconds int64  `json:"retry_after_seconds"`
		}
		_ = json.NewDecoder(limited).Decode(&remote)
		code := strings.TrimSpace(remote.Code)
		if code == "" {
			code = "remote_operation_failed"
		}
		message := strings.TrimSpace(remote.Error)
		if message == "" {
			message = "standalone proxy-pool operation failed"
		}
		retry := time.Duration(remote.RetryAfterSeconds) * time.Second
		return p.remoteResponseError(subscription, response.StatusCode, code, message, retry)
	}
	if output == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if errDecode := json.NewDecoder(limited).Decode(output); errDecode != nil {
		return p.remoteError(subscription, http.StatusBadGateway, "invalid_service_response", 30*time.Second)
	}
	return nil
}

func parseRemoteRevision(raw string) (uint64, bool) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '"' || raw[len(raw)-1] != '"' {
		return 0, false
	}
	value, errParse := strconv.ParseUint(raw[1:len(raw)-1], 10, 64)
	return value, errParse == nil
}

func (p *XAIProxyPool) remoteError(subscription bool, status int, code string, retry time.Duration) error {
	message := "standalone xai proxy pool is unavailable"
	if subscription {
		return &XAIProxySubscriptionError{Code: code, Message: "xAI subscription service is unavailable", Status: status}
	}
	return &XAIProxyPoolError{Message: message, Retry: retry, HTTPStatus: status}
}

func (p *XAIProxyPool) remoteResponseError(subscription bool, status int, code string, message string, retry time.Duration) error {
	if subscription {
		return &XAIProxySubscriptionError{Code: code, Message: message, Status: status}
	}
	return &XAIProxyPoolError{Message: message, Retry: retry, HTTPStatus: status}
}

func (p *XAIProxyPool) Close() {
	if p != nil && p.transport != nil {
		p.transport.CloseIdleConnections()
	}
}

func unavailableStatus(message string) XAIProxyPoolStatus {
	return XAIProxyPoolStatus{
		Enabled: true, Ready: false, LastError: message,
		Lanes: []XAIProxyPoolLaneStatus{}, Providers: []XAIProxyPoolProviderStatus{},
		IPQuarantines: []XAIProxyPoolQuarantineStatus{}, NodeQuarantines: []XAIProxyPoolQuarantineStatus{},
	}
}

func unavailableSubscriptions() XAIProxySubscriptionList {
	return XAIProxySubscriptionList{
		Enabled: true, Ready: false, LastErrorCode: "service_unavailable",
		Subscriptions: []XAIProxySubscriptionStatus{},
	}
}

func poolUnavailable(message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "xai proxy pool is unavailable"
	}
	return &XAIProxyPoolError{Message: message, Retry: 30 * time.Second}
}

func wrapXAIProxyNetworkError(cause error) error {
	if cause == nil {
		return poolUnavailable("xai proxy request failed")
	}
	return &xaiProxyNetworkRetryError{cause: cause}
}

type xaiProxyNetworkRetryError struct{ cause error }

func (e *xaiProxyNetworkRetryError) Error() string {
	if e == nil || e.cause == nil {
		return "xai proxy request failed after lane rotation"
	}
	return "xai proxy request failed after lane rotation: " + e.cause.Error()
}

func (e *xaiProxyNetworkRetryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *xaiProxyNetworkRetryError) StatusCode() int       { return http.StatusServiceUnavailable }
func (e *xaiProxyNetworkRetryError) IsRequestScoped() bool { return true }

func (e *xaiProxyNetworkRetryError) RetryAfter() *time.Duration {
	retry := 30 * time.Second
	return &retry
}

func (e *xaiProxyNetworkRetryError) Headers() http.Header {
	headers := make(http.Header)
	headers.Set("Retry-After", "30")
	return headers
}

func (e *xaiProxyNetworkRetryError) ManagedHeaders() http.Header { return e.Headers() }
