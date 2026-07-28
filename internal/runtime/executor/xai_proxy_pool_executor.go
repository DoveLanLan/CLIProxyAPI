package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/router-for-me/CLIProxyAPI/v7/internal/runtime/executor/helps"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/executor"
	"github.com/tidwall/gjson"
)

const xaiBlockedSpendingLimitCode = "personal-team-blocked:spending-limit"
const xaiProxyRouteAttribute = "_cliproxy_xai_proxy_route"

type xaiProxyRoutedAuth struct {
	auth      *cliproxyauth.Auth
	route     helps.XAIProxyRoute
	resinUsed bool
	poolUsed  bool
}

type xaiProxyRequestScopedNetworkError struct {
	cause error
}

func (e *xaiProxyRequestScopedNetworkError) Error() string {
	if e == nil || e.cause == nil {
		return "xai proxy network error"
	}
	return e.cause.Error()
}

func (e *xaiProxyRequestScopedNetworkError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *xaiProxyRequestScopedNetworkError) StatusCode() int { return http.StatusBadGateway }

func (e *xaiProxyRequestScopedNetworkError) IsRequestScoped() bool { return true }

func (e *XAIAutoExecutor) routedAuth(ctx context.Context, auth *cliproxyauth.Auth) (xaiProxyRoutedAuth, error) {
	if e == nil || auth == nil || strings.TrimSpace(auth.ProxyURL) != "" {
		return xaiProxyRoutedAuth{auth: auth}, nil
	}
	if e.resinProxy != nil {
		proxyURL, routed, errResin := e.resinProxy.ProxyURL(auth.ID)
		if errResin != nil {
			return xaiProxyRoutedAuth{}, errResin
		}
		if routed {
			cloned := auth.Clone()
			cloned.ProxyURL = proxyURL
			return xaiProxyRoutedAuth{auth: cloned, resinUsed: true}, nil
		}
	}
	if e.proxyPool == nil {
		return xaiProxyRoutedAuth{auth: auth}, nil
	}
	route, enrolled, errRoute := e.proxyPool.Route(ctx, auth.ID)
	if errRoute != nil {
		return xaiProxyRoutedAuth{}, errRoute
	}
	if !enrolled {
		return xaiProxyRoutedAuth{auth: auth}, nil
	}
	return xaiProxyRoutedAuth{auth: cloneXAIAuthWithRoute(auth, route), route: route, poolUsed: true}, nil
}

func cloneXAIAuthWithRoute(auth *cliproxyauth.Auth, route helps.XAIProxyRoute) *cliproxyauth.Auth {
	if auth == nil {
		return nil
	}
	cloned := auth.Clone()
	cloned.ProxyURL = strings.TrimSpace(route.ProxyURL)
	if cloned.Attributes == nil {
		cloned.Attributes = make(map[string]string)
	}
	cloned.Attributes[xaiProxyRouteAttribute] = route.LaneName + "\x00" + route.Node
	return cloned
}

func restoreXAIAuthProxy(refreshed *cliproxyauth.Auth, original *cliproxyauth.Auth) *cliproxyauth.Auth {
	if refreshed == nil {
		return nil
	}
	cloned := refreshed.Clone()
	if original == nil {
		cloned.ProxyURL = ""
	} else {
		cloned.ProxyURL = original.ProxyURL
	}
	delete(cloned.Attributes, xaiProxyRouteAttribute)
	return cloned
}

func executeXAIWithProxyPool[T any](ctx context.Context, e *XAIAutoExecutor, auth *cliproxyauth.Auth, run func(*cliproxyauth.Auth) (T, error)) (T, error) {
	var zero T
	routed, errRoute := e.routedAuth(ctx, auth)
	if errRoute != nil {
		return zero, errRoute
	}
	result, errRun := run(routed.auth)
	if routed.resinUsed {
		if !shouldRetryXAIResinNetworkError(ctx, errRun) {
			return result, requestScopeXAIProxyNetworkError(errRun)
		}
		retryResult, errRetry := run(routed.auth)
		return retryResult, requestScopeXAIProxyNetworkError(errRetry)
	}
	if !routed.poolUsed || errRun == nil {
		return result, errRun
	}
	if isXAIBlockedSpendingLimit(errRun) {
		return retryBlockedXAI(ctx, e, auth, routed.route, run)
	}
	if isXAIProxyNetworkError(errRun) {
		next, retry, errNetwork := e.proxyPool.HandlePreconnectFailure(ctx, routed.route)
		if errNetwork != nil {
			return zero, errNetwork
		}
		if retry {
			retryResult, errRetry := run(cloneXAIAuthWithRoute(auth, next))
			if isXAIBlockedSpendingLimit(errRetry) {
				return retryBlockedXAI(ctx, e, auth, next, run)
			}
			if isXAIProxyNetworkError(errRetry) {
				return zero, &xaiProxyRequestScopedNetworkError{cause: errRetry}
			}
			return retryResult, errRetry
		}
		return zero, &xaiProxyRequestScopedNetworkError{cause: errRun}
	}
	return result, errRun
}

func retryBlockedXAI[T any](ctx context.Context, e *XAIAutoExecutor, auth *cliproxyauth.Auth, current helps.XAIProxyRoute, run func(*cliproxyauth.Auth) (T, error)) (T, error) {
	var zero T
	e.proxyPool.RecordExact402(ctx)
	lease, errProbe := e.proxyPool.AcquireProbe(ctx, current)
	if errProbe != nil {
		return zero, errProbe
	}
	alternate, errAlternate := run(cloneXAIAuthWithRoute(auth, lease.AlternateRoute()))
	if errAlternate == nil {
		if errConfirm := lease.ConfirmIPBlock(ctx); errConfirm != nil {
			return zero, &helps.XAIProxyPoolError{Message: "xai proxy pool could not promote the verified alternate", Retry: 30 * time.Second}
		}
		return alternate, nil
	}
	if isXAIBlockedSpendingLimit(errAlternate) {
		lease.CredentialFailure()
		return alternate, errAlternate
	}
	lease.Unavailable()
	return zero, &helps.XAIProxyPoolError{Message: "xai proxy pool could not verify the suspected blocked egress", Retry: 30 * time.Second}
}

func (e *XAIAutoExecutor) executeStreamWithProxyPool(ctx context.Context, auth *cliproxyauth.Auth, run func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error)) (*cliproxyexecutor.StreamResult, error) {
	routed, errRoute := e.routedAuth(ctx, auth)
	if errRoute != nil {
		return nil, errRoute
	}
	if routed.resinUsed {
		return executeXAIResinStream(ctx, routed.auth, run)
	}
	if !routed.poolUsed {
		return run(routed.auth)
	}
	return e.executeRoutedXAIStream(ctx, auth, routed.auth, routed.route, run, true)
}

func executeXAIResinStream(
	ctx context.Context,
	auth *cliproxyauth.Auth,
	run func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error),
) (*cliproxyexecutor.StreamResult, error) {
	stream, errRun := run(auth)
	if errRun != nil {
		if !shouldRetryXAIResinNetworkError(ctx, errRun) {
			return nil, requestScopeXAIProxyNetworkError(errRun)
		}
		stream, errRun = run(auth)
		if errRun != nil {
			return nil, requestScopeXAIProxyNetworkError(errRun)
		}
		return bootstrapXAIResinStream(ctx, stream, false, auth, run)
	}
	return bootstrapXAIResinStream(ctx, stream, true, auth, run)
}

func bootstrapXAIResinStream(
	ctx context.Context,
	stream *cliproxyexecutor.StreamResult,
	allowRetry bool,
	auth *cliproxyauth.Auth,
	run func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error),
) (*cliproxyexecutor.StreamResult, error) {
	if stream == nil || stream.Chunks == nil {
		return stream, nil
	}
	buffered, closed, bootstrapErr := readXAIProxyStreamBootstrap(ctx, stream)
	if bootstrapErr == nil {
		return wrapXAIResinStream(ctx, stream, buffered, closed), nil
	}
	if !allowRetry || !shouldRetryXAIResinNetworkError(ctx, bootstrapErr) {
		return xaiStreamErrorResult(stream.Headers, requestScopeXAIProxyNetworkError(bootstrapErr)), nil
	}
	drainXAIProxyStream(stream)
	retryStream, errRetry := run(auth)
	if errRetry != nil {
		return nil, requestScopeXAIProxyNetworkError(errRetry)
	}
	return bootstrapXAIResinStream(ctx, retryStream, false, auth, run)
}

func (e *XAIAutoExecutor) executeRoutedXAIStream(ctx context.Context, auth *cliproxyauth.Auth, attemptAuth *cliproxyauth.Auth, route helps.XAIProxyRoute, run func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error), allowNetworkRetry bool) (*cliproxyexecutor.StreamResult, error) {
	stream, errRun := run(attemptAuth)
	if errRun != nil {
		if isXAIBlockedSpendingLimit(errRun) {
			return e.retryBlockedXAIStream(ctx, auth, route, run)
		}
		if isXAIProxyNetworkError(errRun) {
			return e.retryXAIStreamAfterNetworkFailure(ctx, auth, route, errRun, run, allowNetworkRetry)
		}
		return nil, errRun
	}
	buffered, closed, bootstrapErr := readXAIProxyStreamBootstrap(ctx, stream)
	if bootstrapErr != nil && isXAIBlockedSpendingLimit(bootstrapErr) {
		drainXAIProxyStream(stream)
		return e.retryBlockedXAIStream(ctx, auth, route, run)
	}
	if bootstrapErr != nil {
		drainXAIProxyStream(stream)
		if isXAIProxyNetworkError(bootstrapErr) {
			return e.retryXAIStreamAfterNetworkFailure(ctx, auth, route, bootstrapErr, run, allowNetworkRetry)
		}
		return nil, bootstrapErr
	}
	return wrapXAIProxyStream(ctx, e.proxyPool, route, stream, buffered, closed), nil
}

func (e *XAIAutoExecutor) retryXAIStreamAfterNetworkFailure(ctx context.Context, auth *cliproxyauth.Auth, route helps.XAIProxyRoute, cause error, run func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error), allowNetworkRetry bool) (*cliproxyexecutor.StreamResult, error) {
	if !allowNetworkRetry {
		return nil, &xaiProxyRequestScopedNetworkError{cause: cause}
	}
	next, retry, errNetwork := e.proxyPool.HandlePreconnectFailure(ctx, route)
	if errNetwork != nil {
		return nil, errNetwork
	}
	if !retry {
		return nil, &xaiProxyRequestScopedNetworkError{cause: cause}
	}
	return e.executeRoutedXAIStream(ctx, auth, cloneXAIAuthWithRoute(auth, next), next, run, false)
}

func (e *XAIAutoExecutor) retryBlockedXAIStream(ctx context.Context, auth *cliproxyauth.Auth, current helps.XAIProxyRoute, run func(*cliproxyauth.Auth) (*cliproxyexecutor.StreamResult, error)) (*cliproxyexecutor.StreamResult, error) {
	e.proxyPool.RecordExact402(ctx)
	lease, errProbe := e.proxyPool.AcquireProbe(ctx, current)
	if errProbe != nil {
		return nil, errProbe
	}
	alternateStream, errAlternate := run(cloneXAIAuthWithRoute(auth, lease.AlternateRoute()))
	if errAlternate != nil {
		if isXAIBlockedSpendingLimit(errAlternate) {
			lease.CredentialFailure()
			return nil, errAlternate
		}
		lease.Unavailable()
		return nil, &helps.XAIProxyPoolError{Message: "xai proxy pool could not verify the suspected blocked egress", Retry: 30 * time.Second}
	}
	buffered, closed, bootstrapErr := readXAIProxyStreamBootstrap(ctx, alternateStream)
	if bootstrapErr != nil {
		drainXAIProxyStream(alternateStream)
		if isXAIBlockedSpendingLimit(bootstrapErr) {
			lease.CredentialFailure()
			return nil, bootstrapErr
		}
		lease.Unavailable()
		return nil, &helps.XAIProxyPoolError{Message: "xai proxy pool could not verify the suspected blocked egress", Retry: 30 * time.Second}
	}
	if errConfirm := lease.ConfirmIPBlock(ctx); errConfirm != nil {
		drainXAIProxyStream(alternateStream)
		return nil, &helps.XAIProxyPoolError{Message: "xai proxy pool could not promote the verified alternate", Retry: 30 * time.Second}
	}
	observedRoute := lease.AlternateRoute()
	observedRoute.LaneName = current.LaneName
	observedRoute.Selector = current.Selector
	return wrapXAIProxyStream(ctx, e.proxyPool, observedRoute, alternateStream, buffered, closed), nil
}

func readXAIProxyStreamBootstrap(ctx context.Context, stream *cliproxyexecutor.StreamResult) ([]cliproxyexecutor.StreamChunk, bool, error) {
	if stream == nil || stream.Chunks == nil {
		return nil, true, nil
	}
	buffered := make([]cliproxyexecutor.StreamChunk, 0, 1)
	for {
		var chunk cliproxyexecutor.StreamChunk
		var ok bool
		if ctx == nil {
			chunk, ok = <-stream.Chunks
		} else {
			select {
			case <-ctx.Done():
				return nil, false, ctx.Err()
			case chunk, ok = <-stream.Chunks:
			}
		}
		if !ok {
			return buffered, true, nil
		}
		if chunk.Err != nil {
			return nil, false, chunk.Err
		}
		buffered = append(buffered, chunk)
		if len(chunk.Payload) > 0 {
			return buffered, false, nil
		}
	}
}

func wrapXAIProxyStream(ctx context.Context, pool xaiProxyPoolClient, route helps.XAIProxyRoute, stream *cliproxyexecutor.StreamResult, buffered []cliproxyexecutor.StreamChunk, closed bool) *cliproxyexecutor.StreamResult {
	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		emit := func(chunk cliproxyexecutor.StreamChunk) bool {
			if chunk.Err != nil && isXAIProxyNetworkError(chunk.Err) {
				pool.ObserveMidResponseFailure(ctx, route)
				chunk.Err = &xaiProxyRequestScopedNetworkError{cause: chunk.Err}
			}
			if ctx == nil {
				out <- chunk
				return true
			}
			select {
			case <-ctx.Done():
				return false
			case out <- chunk:
				return true
			}
		}
		for _, chunk := range buffered {
			if !emit(chunk) {
				drainXAIProxyStream(stream)
				return
			}
		}
		if closed || stream == nil {
			return
		}
		for chunk := range stream.Chunks {
			if !emit(chunk) {
				drainXAIProxyStream(stream)
				return
			}
		}
	}()
	headers := http.Header(nil)
	if stream != nil && stream.Headers != nil {
		headers = stream.Headers.Clone()
	}
	return &cliproxyexecutor.StreamResult{Headers: headers, Chunks: out}
}

func wrapXAIResinStream(
	ctx context.Context,
	stream *cliproxyexecutor.StreamResult,
	buffered []cliproxyexecutor.StreamChunk,
	closed bool,
) *cliproxyexecutor.StreamResult {
	if stream == nil || stream.Chunks == nil {
		return stream
	}
	out := make(chan cliproxyexecutor.StreamChunk)
	go func() {
		defer close(out)
		emit := func(chunk cliproxyexecutor.StreamChunk) bool {
			chunk.Err = requestScopeXAIProxyNetworkError(chunk.Err)
			if ctx == nil {
				out <- chunk
				return true
			}
			select {
			case <-ctx.Done():
				return false
			case out <- chunk:
				return true
			}
		}
		for _, chunk := range buffered {
			if !emit(chunk) {
				drainXAIProxyStream(stream)
				return
			}
		}
		if closed {
			return
		}
		for chunk := range stream.Chunks {
			if !emit(chunk) {
				drainXAIProxyStream(stream)
				return
			}
		}
	}()
	headers := http.Header(nil)
	if stream.Headers != nil {
		headers = stream.Headers.Clone()
	}
	return &cliproxyexecutor.StreamResult{Headers: headers, Chunks: out}
}

func xaiStreamErrorResult(headers http.Header, err error) *cliproxyexecutor.StreamResult {
	chunks := make(chan cliproxyexecutor.StreamChunk, 1)
	chunks <- cliproxyexecutor.StreamChunk{Err: err}
	close(chunks)
	return &cliproxyexecutor.StreamResult{Headers: headers.Clone(), Chunks: chunks}
}

func drainXAIProxyStream(stream *cliproxyexecutor.StreamResult) {
	if stream == nil || stream.Chunks == nil {
		return
	}
	go func() {
		for range stream.Chunks {
		}
	}()
}

func isXAIBlockedSpendingLimit(err error) bool {
	if err == nil || xaiProxyErrorStatus(err) != http.StatusPaymentRequired {
		return false
	}
	message := strings.TrimSpace(err.Error())
	for _, path := range []string{"code", "error.code"} {
		if strings.TrimSpace(gjson.Get(message, path).String()) == xaiBlockedSpendingLimitCode {
			return true
		}
	}
	return strings.Contains(message, `"code":"`+xaiBlockedSpendingLimitCode+`"`) ||
		strings.Contains(message, `"code": "`+xaiBlockedSpendingLimitCode+`"`)
}

func xaiProxyErrorStatus(err error) int {
	type statusCoder interface{ StatusCode() int }
	var status statusCoder
	if errors.As(err, &status) && status != nil {
		return status.StatusCode()
	}
	return 0
}

func isXAIProxyNetworkError(err error) bool {
	if err == nil || xaiProxyErrorStatus(err) != 0 {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return true
	}
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		return true
	}
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		strings.Contains(strings.ToLower(err.Error()), "connection closed") ||
		strings.Contains(strings.ToLower(err.Error()), "connection reset") ||
		strings.Contains(strings.ToLower(err.Error()), "broken pipe")
}

func requestScopeXAIProxyNetworkError(err error) error {
	if !isXAIProxyNetworkError(err) {
		return err
	}
	return &xaiProxyRequestScopedNetworkError{cause: err}
}

func shouldRetryXAIResinNetworkError(ctx context.Context, err error) bool {
	if !isXAIProxyNetworkError(err) {
		return false
	}
	return ctx == nil || ctx.Err() == nil
}

func cloneReplayableXAIRequest(ctx context.Context, req *http.Request) (*http.Request, error) {
	if req == nil {
		return nil, fmt.Errorf("xai proxy pool: request is nil")
	}
	cloned := req.Clone(nonNilXAIContext(ctx, req.Context()))
	if req.Body == nil {
		return cloned, nil
	}
	if req.GetBody == nil {
		return nil, &helps.XAIProxyPoolError{Message: "xai proxy pool cannot replay this request body", Retry: 30 * time.Second}
	}
	body, errBody := req.GetBody()
	if errBody != nil {
		return nil, &helps.XAIProxyPoolError{Message: "xai proxy pool could not replay this request body", Retry: 30 * time.Second}
	}
	cloned.Body = body
	return cloned, nil
}

func nonNilXAIContext(primary context.Context, fallback context.Context) context.Context {
	if primary != nil {
		return primary
	}
	if fallback != nil {
		return fallback
	}
	return context.Background()
}

func responseIsXAIBlockedSpendingLimit(resp *http.Response) (bool, []byte, error) {
	if resp == nil || resp.StatusCode != http.StatusPaymentRequired {
		return false, nil, nil
	}
	body, errRead := io.ReadAll(resp.Body)
	if errClose := resp.Body.Close(); errClose != nil {
		errRead = errors.Join(errRead, errClose)
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if errRead != nil {
		return false, body, errRead
	}
	errStatus := xaiStatusErr(resp.StatusCode, body)
	return isXAIBlockedSpendingLimit(errStatus), body, nil
}
