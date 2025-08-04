package audit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataFilter_DefaultPII(t *testing.T) {
	f := NewMetadataFilter()

	metadata := map[string]any{
		"user_id":      "123",
		"password":     "secret123",
		"email":        "user@example.com",
		"phone":        "1234567890",
		"credit_card":  "4111111111111111",
		"normal_field": "normal_value",
	}

	result := f.Filter(metadata)

	assert.Equal(t, metadata["user_id"], result["user_id"])
	assert.Nil(t, result["password"])
	assert.NotEqual(t, metadata["email"], result["email"])
	assert.Contains(t, result["email"], "") // Should be hashed
	assert.Equal(t, "12******90", result["phone"])
	assert.Equal(t, "41************11", result["credit_card"])
	assert.Equal(t, metadata["normal_field"], result["normal_field"])
}

func TestMetadataFilter_CustomFields(t *testing.T) {
	f := NewMetadataFilter(
		WithCustomField("api_endpoint", FilterActionRemove),
		WithCustomField("internal_id", FilterActionHash),
		WithCustomField("account_number", FilterActionMask),
	)

	metadata := map[string]any{
		"api_endpoint":   "/internal/admin",
		"internal_id":    "INT-12345",
		"account_number": "ACC1234567890",
	}

	result := f.Filter(metadata)

	assert.Nil(t, result["api_endpoint"])
	assert.NotEqual(t, "INT-12345", result["internal_id"])
	assert.Contains(t, result["internal_id"], "") // Should be hashed
	assert.Equal(t, "AC*********90", result["account_number"])
}

func TestMetadataFilter_AllowedFields(t *testing.T) {
	f := NewMetadataFilter(
		WithAllowedField("password"),
		WithAllowedField("email"),
	)

	metadata := map[string]any{
		"password": "secret123",
		"email":    "user@example.com",
		"phone":    "1234567890",
	}

	result := f.Filter(metadata)

	// Allowed fields should pass through unchanged
	assert.Equal(t, "secret123", result["password"])
	assert.Equal(t, "user@example.com", result["email"])
	// Non-allowed PII should still be filtered
	assert.Equal(t, "12******90", result["phone"])
}

func TestMetadataFilter_WithoutPIIDefaults(t *testing.T) {
	f := NewMetadataFilter(WithoutPIIDefaults())

	metadata := map[string]any{
		"password": "secret123",
		"email":    "user@example.com",
		"phone":    "1234567890",
	}

	result := f.Filter(metadata)

	// Without PII defaults, all fields pass through
	assert.Equal(t, "secret123", result["password"])
	assert.Equal(t, "user@example.com", result["email"])
	assert.Equal(t, "1234567890", result["phone"])
}

func TestMetadataFilter_WildcardPatterns(t *testing.T) {
	f := NewMetadataFilter(
		WithCustomField("*.password", FilterActionRemove),
		WithCustomField("secret.*", FilterActionHash),
		WithCustomField("*token*", FilterActionRemove),
	)

	metadata := map[string]any{
		"user.password":   "secret123",
		"admin.password":  "admin123",
		"secret.key":      "key123",
		"secret.value":    "value123",
		"access_token_v2": "token123",
		"refresh_token":   "refresh123",
		"normal_field":    "normal",
	}

	result := f.Filter(metadata)

	assert.Nil(t, result["user.password"])
	assert.Nil(t, result["admin.password"])
	assert.NotEqual(t, "key123", result["secret.key"])
	assert.NotEqual(t, "value123", result["secret.value"])
	assert.Nil(t, result["access_token_v2"])
	assert.Nil(t, result["refresh_token"])
	assert.Equal(t, "normal", result["normal_field"])
}

func TestMetadataFilter_MaskingRules(t *testing.T) {
	f := NewMetadataFilter(
		WithCustomField("short", FilterActionMask),
		WithCustomField("medium", FilterActionMask),
		WithCustomField("long", FilterActionMask),
	)

	metadata := map[string]any{
		"short":  "123",
		"medium": "123456",
		"long":   "1234567890ABCDEF",
	}

	result := f.Filter(metadata)

	assert.Equal(t, "***", result["short"])
	assert.Equal(t, "1****6", result["medium"])
	assert.Equal(t, "12************EF", result["long"])
}

func TestMetadataFilter_NilMetadata(t *testing.T) {
	f := NewMetadataFilter()
	result := f.Filter(nil)
	assert.Nil(t, result)
}

func TestMetadataFilter_EmptyMetadata(t *testing.T) {
	f := NewMetadataFilter()
	result := f.Filter(map[string]any{})
	require.NotNil(t, result)
	assert.Empty(t, result)
}

func TestMetadataFilter_CaseInsensitive(t *testing.T) {
	f := NewMetadataFilter()

	metadata := map[string]any{
		"PASSWORD": "secret123",
		"Password": "secret456",
		"password": "secret789",
		"Email":    "user@example.com",
		"EMAIL":    "admin@example.com",
	}

	result := f.Filter(metadata)

	// All password variants should be removed
	assert.Nil(t, result["PASSWORD"])
	assert.Nil(t, result["Password"])
	assert.Nil(t, result["password"])
	// Email variants should be hashed
	assert.NotEqual(t, "user@example.com", result["Email"])
	assert.NotEqual(t, "admin@example.com", result["EMAIL"])
}

func TestMetadataFilter_ComplexScenario(t *testing.T) {
	f := NewMetadataFilter(
		WithCustomField("internal.*", FilterActionRemove),
		WithCustomField("*.debug", FilterActionRemove),
		WithAllowedField("user_email"),
		WithCustomField("request_body", FilterActionHash),
	)

	metadata := map[string]any{
		"user_id":         "123",
		"user_email":      "allowed@example.com",
		"email":           "filtered@example.com",
		"internal.key":    "should_be_removed",
		"internal.secret": "also_removed",
		"app.debug":       "debug_info",
		"request_body":    `{"data": "sensitive"}`,
		"password":        "secret",
		"normal_data":     "stays",
	}

	result := f.Filter(metadata)

	assert.Equal(t, "123", result["user_id"])
	assert.Equal(t, "allowed@example.com", result["user_email"]) // Explicitly allowed
	assert.NotEqual(t, "filtered@example.com", result["email"])  // Default PII rule
	assert.Nil(t, result["internal.key"])
	assert.Nil(t, result["internal.secret"])
	assert.Nil(t, result["app.debug"])
	assert.NotEqual(t, `{"data": "sensitive"}`, result["request_body"])
	assert.Nil(t, result["password"])
	assert.Equal(t, "stays", result["normal_data"])
}
