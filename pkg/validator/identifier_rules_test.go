package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestValidSlug(t *testing.T) {
	t.Run("valid slugs", func(t *testing.T) {
		validSlugs := []string{
			"hello-world",
			"my-awesome-post",
			"product-123",
			"a",
			"123",
			"test-123-abc",
			"multiple-words-here",
		}

		for _, slug := range validSlugs {
			rule := validator.ValidSlug("slug", slug)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Slug should be valid: %s", slug)
		}
	})

	t.Run("invalid slugs", func(t *testing.T) {
		invalidSlugs := []string{
			"",
			"   ",
			"Hello-World",  // uppercase
			"hello_world",  // underscore
			"hello world",  // space
			"hello--world", // double hyphen
			"-hello-world", // starts with hyphen
			"hello-world-", // ends with hyphen
			"hello@world",  // special character
			"hello.world",  // dot
		}

		for _, slug := range invalidSlugs {
			rule := validator.ValidSlug("slug", slug)
			err := validator.Apply(rule)
			assert.Error(t, err, "Slug should be invalid: %s", slug)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.slug", validationErr[0].TranslationKey)
		}
	})
}

func TestValidUsername(t *testing.T) {
	t.Run("valid usernames", func(t *testing.T) {
		validUsernames := []string{
			"john_doe",
			"user123",
			"test-user",
			"JohnDoe",
			"user_123",
			"a",
			"user-name-123",
		}

		for _, username := range validUsernames {
			rule := validator.ValidUsername("username", username, 1, 20)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Username should be valid: %s", username)
		}
	})

	t.Run("invalid usernames", func(t *testing.T) {
		testCases := []struct {
			username string
			minLen   int
			maxLen   int
		}{
			{"", 3, 20},    // empty
			{"   ", 3, 20}, // whitespace
			{"ab", 3, 20},  // too short
			{"verylongusernamethatexceedslimit", 3, 20}, // too long
			{"user@name", 3, 20},                        // invalid character
			{"user.name", 3, 20},                        // dot not allowed
			{"user name", 3, 20},                        // space not allowed
			{"user+name", 3, 20},                        // plus not allowed
		}

		for _, tc := range testCases {
			rule := validator.ValidUsername("username", tc.username, tc.minLen, tc.maxLen)
			err := validator.Apply(rule)
			assert.Error(t, err, "Username should be invalid: %s", tc.username)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.username", validationErr[0].TranslationKey)
		}
	})
}

func TestValidHandle(t *testing.T) {
	t.Run("valid handles", func(t *testing.T) {
		validHandles := []string{
			"john_doe",
			"user123",
			"test-user",
			"JohnDoe",
			"a",
			"User_Name_123",
		}

		for _, handle := range validHandles {
			rule := validator.ValidHandle("handle", handle, 1, 20)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Handle should be valid: %s", handle)
		}
	})

	t.Run("invalid handles", func(t *testing.T) {
		testCases := []struct {
			handle string
			minLen int
			maxLen int
		}{
			{"", 3, 20},    // empty
			{"   ", 3, 20}, // whitespace
			{"ab", 3, 20},  // too short
			{"verylonghandlethatexceedslimit", 3, 20}, // too long
			{"1user", 3, 20},     // starts with number
			{"_user", 3, 20},     // starts with underscore
			{"-user", 3, 20},     // starts with hyphen
			{"user@name", 3, 20}, // invalid character
			{"user.name", 3, 20}, // dot not allowed
			{"user name", 3, 20}, // space not allowed
		}

		for _, tc := range testCases {
			rule := validator.ValidHandle("handle", tc.handle, tc.minLen, tc.maxLen)
			err := validator.Apply(rule)
			assert.Error(t, err, "Handle should be invalid: %s", tc.handle)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.handle", validationErr[0].TranslationKey)
		}
	})
}

func TestValidSKU(t *testing.T) {
	t.Run("valid SKUs", func(t *testing.T) {
		validSKUs := []string{
			"ABC123",
			"PRODUCT-001",
			"SKU-ABC-123",
			"123456",
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789012345678901234", // 50 chars
		}

		for _, sku := range validSKUs {
			rule := validator.ValidSKU("sku", sku)
			err := validator.Apply(rule)
			assert.NoError(t, err, "SKU should be valid: %s", sku)
		}
	})

	t.Run("invalid SKUs", func(t *testing.T) {
		invalidSKUs := []string{
			"",
			"   ",
			"AB", // too short
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890123456789012345", // too long (51 chars)
			"abc123",  // lowercase
			"ABC_123", // underscore
			"ABC.123", // dot
			"ABC 123", // space
			"ABC@123", // special character
		}

		for _, sku := range invalidSKUs {
			rule := validator.ValidSKU("sku", sku)
			err := validator.Apply(rule)
			assert.Error(t, err, "SKU should be invalid: %s", sku)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.sku", validationErr[0].TranslationKey)
		}
	})
}

func TestValidProductCode(t *testing.T) {
	t.Run("valid product codes", func(t *testing.T) {
		pattern := `^[A-Z]{2}-\d{4}$` // Example: AB-1234
		validCodes := []string{
			"AB-1234",
			"XY-9999",
			"ZZ-0000",
		}

		for _, code := range validCodes {
			rule := validator.ValidProductCode("code", code, pattern)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Product code should be valid: %s", code)
		}
	})

	t.Run("invalid product codes", func(t *testing.T) {
		pattern := `^[A-Z]{2}-\d{4}$`
		invalidCodes := []string{
			"",
			"   ",
			"AB-123",   // too few digits
			"ABC-1234", // too many letters
			"ab-1234",  // lowercase
			"AB_1234",  // underscore instead of hyphen
			"AB-12345", // too many digits
		}

		for _, code := range invalidCodes {
			rule := validator.ValidProductCode("code", code, pattern)
			err := validator.Apply(rule)
			assert.Error(t, err, "Product code should be invalid: %s", code)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.product_code", validationErr[0].TranslationKey)
		}
	})
}

func TestValidHexString(t *testing.T) {
	t.Run("valid hex strings", func(t *testing.T) {
		testCases := []struct {
			value  string
			length int
		}{
			{"ABCDEF", 6},
			{"123456", 6},
			{"abcdef", 6},
			{"0123456789ABCDEF", 16},
			{"deadbeef", 8},
		}

		for _, tc := range testCases {
			rule := validator.ValidHexString("hex", tc.value, tc.length)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Hex string should be valid: %s", tc.value)
		}
	})

	t.Run("invalid hex strings", func(t *testing.T) {
		testCases := []struct {
			value  string
			length int
		}{
			{"", 6},        // empty
			{"   ", 6},     // whitespace
			{"ABCDE", 6},   // wrong length
			{"ABCDEG", 6},  // invalid character
			{"ABCDEF1", 6}, // wrong length
			{"GHIJKL", 6},  // invalid characters
			{"ABC DEF", 6}, // space not allowed
		}

		for _, tc := range testCases {
			rule := validator.ValidHexString("hex", tc.value, tc.length)
			err := validator.Apply(rule)
			assert.Error(t, err, "Hex string should be invalid: %s", tc.value)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.hex_string", validationErr[0].TranslationKey)
		}
	})

	t.Run("any length hex strings", func(t *testing.T) {
		validHex := []string{
			"A",
			"AB",
			"ABCD",
			"123456789ABCDEF",
		}

		for _, hex := range validHex {
			rule := validator.ValidHexString("hex", hex, 0) // 0 means any length
			err := validator.Apply(rule)
			assert.NoError(t, err, "Hex string should be valid: %s", hex)
		}
	})
}

func TestValidBase64(t *testing.T) {
	t.Run("valid base64 strings", func(t *testing.T) {
		validBase64 := []string{
			"SGVsbG8gV29ybGQ=",             // "Hello World"
			"VGVzdA==",                     // "Test"
			"QQ==",                         // "A"
			"QWxhZGRpbjpvcGVuIHNlc2FtZQ==", // "Aladdin:open sesame"
		}

		for _, b64 := range validBase64 {
			rule := validator.ValidBase64("base64", b64)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Base64 string should be valid: %s", b64)
		}
	})

	t.Run("invalid base64 strings", func(t *testing.T) {
		invalidBase64 := []string{
			"",
			"   ",
			"SGVsbG8gV29ybGQ",    // missing padding
			"SGVsbG8gV29ybGQ===", // too much padding
			"SGVsbG8@V29ybGQ=",   // invalid character
			"SGVsbG8 V29ybGQ=",   // space not allowed
			"SGVsbG8gV29ybG",     // wrong length
		}

		for _, b64 := range invalidBase64 {
			rule := validator.ValidBase64("base64", b64)
			err := validator.Apply(rule)
			assert.Error(t, err, "Base64 string should be invalid: %s", b64)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.base64", validationErr[0].TranslationKey)
		}
	})
}

func TestValidCustomID(t *testing.T) {
	t.Run("valid custom IDs", func(t *testing.T) {
		pattern := `^USR-\d{6}$`
		description := "user ID"
		validIDs := []string{
			"USR-123456",
			"USR-000001",
			"USR-999999",
		}

		for _, id := range validIDs {
			rule := validator.ValidCustomID("id", id, pattern, description)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Custom ID should be valid: %s", id)
		}
	})

	t.Run("invalid custom IDs", func(t *testing.T) {
		pattern := `^USR-\d{6}$`
		description := "user ID"
		invalidIDs := []string{
			"",
			"   ",
			"USR-12345",   // too few digits
			"USR-1234567", // too many digits
			"usr-123456",  // lowercase
			"USR_123456",  // underscore
			"ABC-123456",  // wrong prefix
		}

		for _, id := range invalidIDs {
			rule := validator.ValidCustomID("id", id, pattern, description)
			err := validator.Apply(rule)
			assert.Error(t, err, "Custom ID should be invalid: %s", id)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.custom_id", validationErr[0].TranslationKey)
		}
	})
}

func TestValidDomainName(t *testing.T) {
	t.Run("valid domain names", func(t *testing.T) {
		validDomains := []string{
			"example.com",
			"sub.example.com",
			"test-domain.org",
			"my-site.co.uk",
			"123domain.net",
			"a.co",
		}

		for _, domain := range validDomains {
			rule := validator.ValidDomainName("domain", domain)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Domain should be valid: %s", domain)
		}
	})

	t.Run("invalid domain names", func(t *testing.T) {
		invalidDomains := []string{
			"",
			"   ",
			"example",       // no TLD
			".example.com",  // starts with dot
			"example.com.",  // ends with dot
			"ex..ample.com", // double dot
			"-example.com",  // starts with hyphen
			"example-.com",  // ends with hyphen
			"example.c",     // TLD too short
			"example..com",  // double dot
		}

		for _, domain := range invalidDomains {
			rule := validator.ValidDomainName("domain", domain)
			err := validator.Apply(rule)
			assert.Error(t, err, "Domain should be invalid: %s", domain)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.domain_name", validationErr[0].TranslationKey)
		}
	})
}

func TestValidSubdomain(t *testing.T) {
	t.Run("valid subdomains", func(t *testing.T) {
		validSubdomains := []string{
			"api",
			"www",
			"mail",
			"test-api",
			"my-service",
			"123",
			"a",
		}

		for _, subdomain := range validSubdomains {
			rule := validator.ValidSubdomain("subdomain", subdomain)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Subdomain should be valid: %s", subdomain)
		}
	})

	t.Run("invalid subdomains", func(t *testing.T) {
		invalidSubdomains := []string{
			"",
			"   ",
			"-api",        // starts with hyphen
			"api-",        // ends with hyphen
			"api.service", // contains dot
			"api_service", // contains underscore
			"verylongsubdomainthatexceedssixtythreecharacterslimitandthensome1234567890", // too long (73 chars)
		}

		for _, subdomain := range invalidSubdomains {
			rule := validator.ValidSubdomain("subdomain", subdomain)
			err := validator.Apply(rule)
			assert.Error(t, err, "Subdomain should be invalid: %s", subdomain)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.subdomain", validationErr[0].TranslationKey)
		}
	})
}

func TestValidAPIKey(t *testing.T) {
	t.Run("valid API keys", func(t *testing.T) {
		validKeys := []string{
			"abc123def456",
			"API-KEY-12345",
			"my_api_key_123",
			"ABCDEF123456",
		}

		for _, key := range validKeys {
			rule := validator.ValidAPIKey("api_key", key, 8, 50)
			err := validator.Apply(rule)
			assert.NoError(t, err, "API key should be valid: %s", key)
		}
	})

	t.Run("invalid API keys", func(t *testing.T) {
		testCases := []struct {
			key    string
			minLen int
			maxLen int
		}{
			{"", 8, 50},      // empty
			{"   ", 8, 50},   // whitespace
			{"short", 8, 50}, // too short
			{"verylongapikeyverylongapikeyverylongapikeyverylongapikey", 8, 50}, // too long
			{"api@key", 8, 50}, // invalid character
			{"api.key", 8, 50}, // dot not allowed
			{"api key", 8, 50}, // space not allowed
			{"api+key", 8, 50}, // plus not allowed
		}

		for _, tc := range testCases {
			rule := validator.ValidAPIKey("api_key", tc.key, tc.minLen, tc.maxLen)
			err := validator.Apply(rule)
			assert.Error(t, err, "API key should be invalid: %s", tc.key)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.api_key", validationErr[0].TranslationKey)
		}
	})
}

func TestValidTicketNumber(t *testing.T) {
	t.Run("valid ticket numbers", func(t *testing.T) {
		testCases := []struct {
			ticket string
			prefix string
		}{
			{"TICKET123456", "TICKET"},
			{"SUP999999", "SUP"},
			{"INC-ABC123", "INC-"},
			{"123456", ""}, // no prefix
			{"ABCDEF", ""}, // no prefix
		}

		for _, tc := range testCases {
			rule := validator.ValidTicketNumber("ticket", tc.ticket, tc.prefix)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Ticket number should be valid: %s", tc.ticket)
		}
	})

	t.Run("invalid ticket numbers", func(t *testing.T) {
		testCases := []struct {
			ticket string
			prefix string
		}{
			{"", "TICKET"},           // empty
			{"   ", "TICKET"},        // whitespace
			{"WRONG123", "TICKET"},   // wrong prefix
			{"ticket123", "TICKET"},  // lowercase prefix
			{"TICKETabc", "TICKET"},  // lowercase after prefix
			{"TICKET@123", "TICKET"}, // invalid character
			{"TICKET.123", "TICKET"}, // dot not allowed
			{"TICKET 123", "TICKET"}, // space not allowed
		}

		for _, tc := range testCases {
			rule := validator.ValidTicketNumber("ticket", tc.ticket, tc.prefix)
			err := validator.Apply(rule)
			assert.Error(t, err, "Ticket number should be invalid: %s", tc.ticket)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.ticket_number", validationErr[0].TranslationKey)
		}
	})
}

func TestValidVersion(t *testing.T) {
	t.Run("valid semantic versions", func(t *testing.T) {
		validVersions := []string{
			"1.0.0",
			"0.0.1",
			"10.20.30",
			"1.0.0-alpha",
			"1.0.0-alpha.1",
			"1.0.0-beta.2",
			"1.0.0-rc.1",
			"1.0.0+build.1",
			"1.0.0-alpha+build.1",
			"2.0.0-rc.1+build.123",
			"1.2.3-beta.11",
			"1.0.0-x.7.z.92",
		}

		for _, version := range validVersions {
			rule := validator.ValidVersion("version", version)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Version should be valid: %s", version)
		}
	})

	t.Run("invalid semantic versions", func(t *testing.T) {
		invalidVersions := []string{
			"",
			"   ",
			"1",
			"1.2",
			"1.2.3.4",
			"1.2.3-",
			"1.2.3+",
			"1.2.3-+",
			"01.2.3",         // leading zero
			"1.02.3",         // leading zero
			"1.2.03",         // leading zero
			"1.2.3-01",       // leading zero in prerelease
			"1.2.3-alpha..1", // double dot
			"1.2.3-alpha.",   // trailing dot
			"1.2.3-.alpha",   // leading dot
			"v1.2.3",         // prefix not allowed
		}

		for _, version := range invalidVersions {
			rule := validator.ValidVersion("version", version)
			err := validator.Apply(rule)
			assert.Error(t, err, "Version should be invalid: %s", version)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.version", validationErr[0].TranslationKey)
		}
	})
}

func TestIdentifierValidationCombination(t *testing.T) {
	t.Run("comprehensive identifier validation", func(t *testing.T) {
		slug := "my-awesome-product"
		username := "john_doe"
		sku := "PROD-123"
		version := "1.2.3"

		err := validator.Apply(
			validator.ValidSlug("slug", slug),
			validator.ValidUsername("username", username, 3, 20),
			validator.ValidSKU("sku", sku),
			validator.ValidVersion("version", version),
		)

		assert.NoError(t, err, "Valid identifier data should pass all validations")
	})

	t.Run("invalid identifier data fails multiple validations", func(t *testing.T) {
		slug := "Invalid Slug!"
		username := "u"      // too short
		sku := "invalid-sku" // lowercase
		version := "1.2"     // incomplete

		err := validator.Apply(
			validator.ValidSlug("slug", slug),
			validator.ValidUsername("username", username, 3, 20),
			validator.ValidSKU("sku", sku),
			validator.ValidVersion("version", version),
		)

		assert.Error(t, err, "Invalid identifier data should fail validations")

		validationErr := validator.ExtractValidationErrors(err)
		require.NotNil(t, validationErr)
		assert.True(t, len(validationErr) > 1, "Should have multiple validation errors")
	})
}
