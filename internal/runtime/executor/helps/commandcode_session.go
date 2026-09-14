package helps

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const CommandCodeMaxSessions = 256

var commandCodeKeyPattern = regexp.MustCompile(`^user_[a-zA-Z0-9_-]+$`)

// ValidCommandCodeKey deliberately rejects embedded prefixes and malformed keys.
func ValidCommandCodeKey(key string) bool { return commandCodeKeyPattern.MatchString(key) }

type commandCodeSession struct {
	id      string
	expires time.Time
	ready   chan struct{}
	err     error
}

// CommandCodeSessions bounds retained state and coalesces concurrent initialization.
// Entries contain hashed identities, never raw credentials or host information.
type CommandCodeSessions struct {
	mu      sync.Mutex
	entries map[[32]byte]*commandCodeSession
}

func commandCodeIdentity(base, key string, headers http.Header) [32]byte {
	stable := headers.Clone()
	stable.Del("traceparent")
	stable.Del("x-session-id")
	encoded, _ := json.Marshal(stable)
	return sha256.Sum256([]byte(base + "\x00" + key + "\x00" + string(encoded)))
}

// Ensure initializes the selected upstream credential using the request's transport.
func (s *CommandCodeSessions) Ensure(ctx context.Context, client *http.Client, base, key string, headers http.Header) (string, error) {
	identity := commandCodeIdentity(base, key, headers)
	now := time.Now()
	s.mu.Lock()
	if s.entries == nil {
		s.entries = make(map[[32]byte]*commandCodeSession)
	}
	if entry := s.entries[identity]; entry != nil && now.Before(entry.expires) {
		s.mu.Unlock()
		select {
		case <-entry.ready:
			return entry.id, entry.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	for id, entry := range s.entries {
		select {
		case <-entry.ready:
			if !now.Before(entry.expires) {
				delete(s.entries, id)
			}
		default:
		}
	}
	if len(s.entries) >= CommandCodeMaxSessions {
		var oldestID [32]byte
		var oldest *commandCodeSession
		for id, entry := range s.entries {
			select {
			case <-entry.ready:
				if oldest == nil || entry.expires.Before(oldest.expires) {
					oldestID, oldest = id, entry
				}
			default:
			}
		}
		if oldest == nil {
			s.mu.Unlock()
			return "", CommandCodeError(503, "credential initialization capacity reached")
		}
		delete(s.entries, oldestID)
	}
	entry := &commandCodeSession{id: uuid.NewString(), expires: now.Add(8 * time.Hour), ready: make(chan struct{})}
	s.entries[identity] = entry
	s.mu.Unlock()

	err := commandCodeInitialize(ctx, client, base, key, entry.id, headers)
	s.mu.Lock()
	entry.err = err
	if err != nil {
		delete(s.entries, identity)
	}
	close(entry.ready)
	s.mu.Unlock()
	return entry.id, err
}

func commandCodeInitialize(ctx context.Context, client *http.Client, base, key, session string, headers http.Header) error {
	fingerprint := CommandCodeFingerprint(key)
	lifecycle := map[string]any{"eventType": "cli_session_exists", "metadata": map[string]any{"sessionId": "sess_" + strings.ReplaceAll(session, "-", "")[:16], "cliVersion": CommandCodeVersion, "mode": "interactive", "os": "win32-x64"}}
	for _, call := range []struct {
		path string
		body any
	}{{"/alpha/fingerprint/record", fingerprint}, {"/alpha/lifecycle-events", lifecycle}} {
		body, err := json.Marshal(call.body)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+call.path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header = headers.Clone()
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		// Bound initialization bodies and close even when they are unexpectedly large.
		_, errRead := io.Copy(io.Discard, io.LimitReader(response.Body, 64<<10))
		if errClose := response.Body.Close(); errClose != nil {
			log.Debug("commandcode: initialization body close failed")
		}
		if errRead != nil {
			return errRead
		}
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return CommandCodeHTTPError(response.StatusCode, "credential initialization rejected", response.Header)
		}
	}
	return nil
}

// CommandCodeFingerprint uses stable anonymous signals, not real host identifiers.
// Signal hashing follows the inspected CLI protocol; it is not an authentication guarantee.
func CommandCodeFingerprint(key string) map[string]any {
	digest := func(field string) string {
		h := sha256.Sum256([]byte("\x00" + key + "\x00" + field))
		return hex.EncodeToString(h[:])
	}
	hash := func(value string) string {
		h := sha256.Sum256([]byte("command-code:device-fingerprint:v1\x00" + strings.ToLower(value)))
		return hex.EncodeToString(h[:])
	}
	mid := digest("machineId")[:32]
	machine := fmt.Sprintf("%s-%s-%s-%s-%s", mid[:8], mid[8:12], mid[12:16], mid[16:20], mid[20:])
	macHex := digest("mac0")[:12]
	mac := fmt.Sprintf("%s:%s:%s:%s:%s:%s", macHex[:2], macHex[2:4], macHex[4:6], macHex[6:8], macHex[8:10], macHex[10:])
	thumb := sha256.Sum256([]byte("command-code:device-fingerprint:v1\x00machine\x00" + machine + "|" + mac))
	return map[string]any{"thumbmark": hex.EncodeToString(thumb[:]), "components": map[string]any{
		"machineIdHash": hash(machine), "macHashes": []string{hash(mac)}, "osUserHash": hash("dev"), "hostnameHash": hash("DESKTOP-" + digest("hostname")[:8]),
		"platform": "win32", "arch": "x64", "osRelease": "10.0.22631", "cpuModel": "AMD Ryzen 5 7600", "cpuCount": 6, "memGiB": 16,
		"isContainer": false, "timezone": "UTC", "runtime": "cli", "collectorVersion": 1,
	}}
}

// ApplyCommandCodeHeaders pins the protocol version and isolates credentials.
func ApplyCommandCodeHeaders(req *http.Request, key, session string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/x-ndjson, text/event-stream")
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("User-Agent", "cli")
	req.Header.Set("x-command-code-version", CommandCodeVersion)
	req.Header.Set("x-cli-environment", "production")
	req.Header.Set("x-project-slug", "c-users-dev-projects-app")
	req.Header.Set("x-taste-learning", "false")
	if session != "" {
		req.Header.Set("x-session-id", session)
	}
	trace := strings.ReplaceAll(uuid.NewString(), "-", "")
	span := strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	req.Header.Set("traceparent", "00-"+trace+"-"+span+"-01")
}

type commandCodeError struct {
	status     int
	message    string
	retryAfter *time.Duration
	headers    http.Header
}

func (e *commandCodeError) Error() string {
	errorType := "upstream_error"
	switch e.status {
	case 400, 413:
		errorType = "invalid_request_error"
	case 401, 403:
		errorType = "authentication_error"
	case 429:
		errorType = "rate_limit_error"
	}
	body, _ := json.Marshal(map[string]any{"error": map[string]any{"message": "commandcode: " + e.message, "type": errorType}})
	return string(body)
}
func (e *commandCodeError) StatusCode() int            { return e.status }
func (e *commandCodeError) RetryAfter() *time.Duration { return e.retryAfter }
func (e *commandCodeError) Headers() http.Header       { return e.headers.Clone() }

// CommandCodeHTTPError preserves a valid upstream Retry-After without forwarding
// arbitrary headers or exposing the upstream error body.
func CommandCodeHTTPError(status int, message string, headers http.Header) error {
	err := CommandCodeError(status, message).(*commandCodeError)
	raw := headers.Get("Retry-After")
	var delay time.Duration
	if seconds, parseErr := strconv.ParseInt(raw, 10, 32); parseErr == nil && seconds >= 0 {
		delay = time.Duration(seconds) * time.Second
	} else if deadline, parseErr := http.ParseTime(raw); parseErr == nil {
		delay = time.Until(deadline)
	}
	if delay > 0 {
		err.retryAfter = &delay
		err.headers = http.Header{"Retry-After": []string{raw}}
	}
	return err
}

// CommandCodeError maps upstream status without leaking response bodies or credentials.
func CommandCodeError(status int, message string) error {
	switch status {
	case 402:
		status = 429
	case 422:
		status = 400
	case 400, 401, 403, 404, 413, 429, 499, 500, 502, 503, 504:
	default:
		status = 502
	}
	return &commandCodeError{status: status, message: message}
}

// CloseCommandCodeBody closes a response without logging upstream error text.
func CloseCommandCodeBody(body io.ReadCloser) {
	if err := body.Close(); err != nil {
		log.Debug("commandcode: response body close failed")
	}
}

// CommandCodeResponseHeaders drops upstream compression and length headers after conversion.
func CommandCodeResponseHeaders(stream bool) http.Header {
	header := make(http.Header)
	if stream {
		header.Set("Content-Type", "text/event-stream")
	} else {
		header.Set("Content-Type", "application/json")
	}
	return header
}
