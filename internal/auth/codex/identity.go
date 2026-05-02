package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// IdentityMetadata contains non-secret Codex account identity details derived
// from an ID token. AccountID is the ChatGPT account/workspace identifier used
// by upstream headers; it is not unique to a human user on team accounts.
type IdentityMetadata struct {
	Email            string
	ChatgptAccountID string
	ChatgptUserID    string
	CodexAccountHash string
	CodexUserHash    string
	PlanType         string
}

// ExtractIdentityMetadata derives safe, non-secret identity metadata from a
// Codex ID token. Invalid or missing ID tokens simply return fallback values.
func ExtractIdentityMetadata(idToken, fallbackEmail string) IdentityMetadata {
	identity := IdentityMetadata{
		Email: strings.TrimSpace(fallbackEmail),
	}
	claims, err := ParseJWTToken(strings.TrimSpace(idToken))
	if err != nil || claims == nil {
		return identity
	}

	if identity.Email == "" {
		identity.Email = strings.TrimSpace(claims.Email)
	}
	identity.ChatgptAccountID = strings.TrimSpace(claims.CodexAuthInfo.ChatgptAccountID)
	identity.ChatgptUserID = strings.TrimSpace(claims.CodexAuthInfo.ChatgptUserID)
	identity.PlanType = strings.TrimSpace(claims.CodexAuthInfo.ChatgptPlanType)

	identity.CodexAccountHash = HashIdentityValue(identity.ChatgptAccountID)
	userHashSource := firstNonEmpty(
		identity.ChatgptUserID,
		claims.CodexAuthInfo.UserID,
		claims.Sub,
		identity.Email,
	)
	identity.CodexUserHash = HashIdentityValue(userHashSource)

	return identity
}

// HashIdentityValue returns the short stable hash used in Codex auth filenames
// and non-secret metadata.
func HashIdentityValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])[:8]
}

// ApplyToMetadata adds non-secret identity fields to a top-level auth metadata
// map without removing existing fields.
func (i IdentityMetadata) ApplyToMetadata(metadata map[string]any) {
	if metadata == nil {
		return
	}
	if i.Email != "" {
		metadata["email"] = i.Email
	}
	if i.ChatgptAccountID != "" {
		metadata["chatgpt_account_id"] = i.ChatgptAccountID
	}
	if i.ChatgptUserID != "" {
		metadata["chatgpt_user_id"] = i.ChatgptUserID
	}
	if i.CodexAccountHash != "" {
		metadata["codex_account_hash"] = i.CodexAccountHash
	}
	if i.CodexUserHash != "" {
		metadata["codex_user_hash"] = i.CodexUserHash
	}
	if i.PlanType != "" {
		metadata["plan_type"] = i.PlanType
	}
}

// ApplyToAttributes adds non-secret identity fields to runtime auth attributes.
func (i IdentityMetadata) ApplyToAttributes(attributes map[string]string) {
	if attributes == nil {
		return
	}
	if i.Email != "" {
		attributes["email"] = i.Email
	}
	if i.ChatgptAccountID != "" {
		attributes["chatgpt_account_id"] = i.ChatgptAccountID
	}
	if i.ChatgptUserID != "" {
		attributes["chatgpt_user_id"] = i.ChatgptUserID
	}
	if i.CodexAccountHash != "" {
		attributes["codex_account_hash"] = i.CodexAccountHash
	}
	if i.CodexUserHash != "" {
		attributes["codex_user_hash"] = i.CodexUserHash
	}
	if i.PlanType != "" {
		attributes["plan_type"] = i.PlanType
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
