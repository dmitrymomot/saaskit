package audit_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/audit"
)

func TestNewSHA256Hasher(t *testing.T) {
	t.Parallel()

	hasher := audit.NewSHA256Hasher()
	require.NotNil(t, hasher)

	// Verify it implements the Hasher interface
	var _ audit.Hasher = hasher
}

func TestSHA256Hasher_Hash(t *testing.T) {
	t.Parallel()

	hasher := audit.NewSHA256Hasher()
	require.NotNil(t, hasher)

	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("basic event hashing", func(t *testing.T) {
		event := audit.Event{
			ID:         "evt_123",
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "create",
			Resource:   "user",
			ResourceID: "user_new",
			Result:     audit.ResultSuccess,
			Error:      "",
			Metadata:   map[string]interface{}{"ip": "192.168.1.1"},
			Hash:       "should_be_ignored",
			PrevHash:   "should_also_be_ignored",
			CreatedAt:  baseTime,
		}

		hash := hasher.Hash(event)

		// Hash should be a 64-character hex string (SHA256)
		assert.Len(t, hash, 64)
		assert.Regexp(t, "^[a-f0-9]{64}$", hash)
		assert.NotEmpty(t, hash)
	})

	t.Run("same input produces same hash (deterministic)", func(t *testing.T) {
		event := audit.Event{
			ID:         "evt_123",
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "update",
			Resource:   "profile",
			ResourceID: "profile_123",
			Result:     audit.ResultSuccess,
			Error:      "",
			Metadata:   map[string]interface{}{"key": "value"},
			Hash:       "old_hash_1",
			PrevHash:   "old_hash_2",
			CreatedAt:  baseTime,
		}

		hash1 := hasher.Hash(event)
		hash2 := hasher.Hash(event)

		assert.Equal(t, hash1, hash2)
		assert.NotEmpty(t, hash1)
	})

	t.Run("different inputs produce different hashes", func(t *testing.T) {
		event1 := audit.Event{
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "create",
			Resource:   "user",
			ResourceID: "user_new",
			Result:     audit.ResultSuccess,
			CreatedAt:  baseTime,
		}

		event2 := audit.Event{
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "update", // Different action
			Resource:   "user",
			ResourceID: "user_new",
			Result:     audit.ResultSuccess,
			CreatedAt:  baseTime,
		}

		hash1 := hasher.Hash(event1)
		hash2 := hasher.Hash(event2)

		assert.NotEqual(t, hash1, hash2)
		assert.NotEmpty(t, hash1)
		assert.NotEmpty(t, hash2)
	})

	t.Run("hash excludes Hash and PrevHash fields", func(t *testing.T) {
		baseEvent := audit.Event{
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "delete",
			Resource:   "post",
			ResourceID: "post_123",
			Result:     audit.ResultSuccess,
			CreatedAt:  baseTime,
		}

		// Same event with different Hash and PrevHash
		event1 := baseEvent
		event1.Hash = "hash_value_1"
		event1.PrevHash = "prev_hash_1"

		event2 := baseEvent
		event2.Hash = "hash_value_2"
		event2.PrevHash = "prev_hash_2"

		hash1 := hasher.Hash(event1)
		hash2 := hasher.Hash(event2)

		// Hashes should be identical since Hash and PrevHash are excluded
		assert.Equal(t, hash1, hash2)
		assert.NotEmpty(t, hash1)
	})

	t.Run("hash excludes ID and Metadata fields", func(t *testing.T) {
		baseEvent := audit.Event{
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "view",
			Resource:   "document",
			ResourceID: "doc_456",
			Result:     audit.ResultSuccess,
			CreatedAt:  baseTime,
		}

		// Same event with different ID and Metadata
		event1 := baseEvent
		event1.ID = "event_id_1"
		event1.Metadata = map[string]interface{}{"user_agent": "browser1", "ip": "1.1.1.1"}

		event2 := baseEvent
		event2.ID = "event_id_2"
		event2.Metadata = map[string]interface{}{"user_agent": "browser2", "ip": "2.2.2.2"}

		hash1 := hasher.Hash(event1)
		hash2 := hasher.Hash(event2)

		// Hashes should be identical since ID and Metadata are excluded
		assert.Equal(t, hash1, hash2)
		assert.NotEmpty(t, hash1)
	})

	t.Run("different timestamps produce different hashes", func(t *testing.T) {
		baseEvent := audit.Event{
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "login",
			Resource:   "auth",
			ResourceID: "session_new",
			Result:     audit.ResultSuccess,
		}

		event1 := baseEvent
		event1.CreatedAt = baseTime

		event2 := baseEvent
		event2.CreatedAt = baseTime.Add(time.Second) // Different timestamp

		hash1 := hasher.Hash(event1)
		hash2 := hasher.Hash(event2)

		assert.NotEqual(t, hash1, hash2)
		assert.NotEmpty(t, hash1)
		assert.NotEmpty(t, hash2)
	})

	t.Run("different result types produce different hashes", func(t *testing.T) {
		baseEvent := audit.Event{
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "payment",
			Resource:   "subscription",
			ResourceID: "sub_123",
			CreatedAt:  baseTime,
		}

		event1 := baseEvent
		event1.Result = audit.ResultSuccess

		event2 := baseEvent
		event2.Result = audit.ResultFailure

		event3 := baseEvent
		event3.Result = audit.ResultError

		hash1 := hasher.Hash(event1)
		hash2 := hasher.Hash(event2)
		hash3 := hasher.Hash(event3)

		// All hashes should be different
		assert.NotEqual(t, hash1, hash2)
		assert.NotEqual(t, hash2, hash3)
		assert.NotEqual(t, hash1, hash3)
	})

	t.Run("error field affects hash", func(t *testing.T) {
		baseEvent := audit.Event{
			TenantID:   "tenant_456",
			UserID:     "user_789",
			SessionID:  "session_abc",
			Action:     "api_call",
			Resource:   "endpoint",
			ResourceID: "/api/users",
			Result:     audit.ResultError,
			CreatedAt:  baseTime,
		}

		event1 := baseEvent
		event1.Error = ""

		event2 := baseEvent
		event2.Error = "connection timeout"

		hash1 := hasher.Hash(event1)
		hash2 := hasher.Hash(event2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty string fields", func(t *testing.T) {
		event := audit.Event{
			TenantID:   "",
			UserID:     "",
			SessionID:  "",
			Action:     "",
			Resource:   "",
			ResourceID: "",
			Result:     "",
			Error:      "",
			CreatedAt:  time.Time{}, // Zero time
		}

		hash := hasher.Hash(event)

		// Should still produce a valid hash even with all empty fields
		assert.Len(t, hash, 64)
		assert.Regexp(t, "^[a-f0-9]{64}$", hash)
		assert.NotEmpty(t, hash)
	})
}

// BenchmarkSHA256Hasher_Hash benchmarks the Hash function performance
func BenchmarkSHA256Hasher_Hash(t *testing.B) {
	hasher := audit.NewSHA256Hasher()
	baseTime := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

	t.Run("typical_event", func(b *testing.B) {
		event := audit.Event{
			ID:         "evt_1234567890",
			TenantID:   "tenant_abcdef123456",
			UserID:     "user_987654321",
			SessionID:  "session_qwertyuiop",
			Action:     "create_subscription",
			Resource:   "billing",
			ResourceID: "sub_premium_monthly_123",
			Result:     audit.ResultSuccess,
			Error:      "",
			Metadata: map[string]interface{}{
				"ip_address":    "192.168.1.100",
				"user_agent":    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
				"plan_type":     "premium",
				"billing_cycle": "monthly",
				"amount":        29.99,
			},
			Hash:      "previous_hash_value_64_chars_long_abcdef1234567890abcdef12",
			PrevHash:  "even_older_hash_value_64_chars_long_123456789abcdef0123456",
			CreatedAt: baseTime,
		}

		b.ResetTimer()
		for b.Loop() {
			hash := hasher.Hash(event)
			if len(hash) != 64 {
				b.Fatal("invalid hash length")
			}
		}
	})

	t.Run("minimal_event", func(b *testing.B) {
		event := audit.Event{
			TenantID:   "t1",
			UserID:     "u1",
			SessionID:  "s1",
			Action:     "login",
			Resource:   "auth",
			ResourceID: "session",
			Result:     audit.ResultSuccess,
			Error:      "",
			CreatedAt:  baseTime,
		}

		b.ResetTimer()
		for b.Loop() {
			hash := hasher.Hash(event)
			if len(hash) != 64 {
				b.Fatal("invalid hash length")
			}
		}
	})

	t.Run("error_event", func(b *testing.B) {
		event := audit.Event{
			TenantID:   "tenant_large_scale_enterprise_system",
			UserID:     "user_admin_with_elevated_privileges",
			SessionID:  "session_long_running_background_task",
			Action:     "bulk_data_processing_operation",
			Resource:   "data_warehouse_analytics_engine",
			ResourceID: "report_quarterly_financial_analysis_2024",
			Result:     audit.ResultError,
			Error:      "database connection timeout after 30 seconds, retry attempted but failed due to network partition, escalated to ops team for investigation",
			CreatedAt:  baseTime,
		}

		b.ResetTimer()
		for b.Loop() {
			hash := hasher.Hash(event)
			if len(hash) != 64 {
				b.Fatal("invalid hash length")
			}
		}
	})
}
