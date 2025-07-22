package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/validator"
)

func TestValidEmail(t *testing.T) {
	t.Run("valid emails", func(t *testing.T) {
		validEmails := []string{
			"test@example.com",
			"user.name@domain.co.uk",
			"user+tag@example.org",
			"firstname.lastname@company.com",
			"email@123.123.123.123", // IP address domain
			"1234567890@example.com",
			"email@example-one.com",
			"_______@example.com",
			"email@example.name",
		}

		for _, email := range validEmails {
			rule := validator.ValidEmail("email", email)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Email should be valid: %s", email)
		}
	})

	t.Run("invalid emails", func(t *testing.T) {
		invalidEmails := []string{
			"",
			"   ",
			"plainaddress",
			"@missingdomain.com",
			"missing@.com",
			"missing@domain",
			"spaces @domain.com",
			"email @domain .com",
			"email..double.dot@domain.com",
			"email@domain..com",
		}

		for _, email := range invalidEmails {
			rule := validator.ValidEmail("email", email)
			err := validator.Apply(rule)
			assert.Error(t, err, "Email should be invalid: %s", email)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.email", validationErr[0].TranslationKey)
		}
	})
}

func TestValidURL(t *testing.T) {
	t.Run("valid URLs", func(t *testing.T) {
		validURLs := []string{
			"http://example.com",
			"https://example.com",
			"https://www.example.com/path",
			"https://example.com:8080",
			"https://example.com/path?query=value",
			"https://example.com/path#fragment",
			"ftp://files.example.com",
			"https://sub.domain.example.com",
		}

		for _, url := range validURLs {
			rule := validator.ValidURL("url", url)
			err := validator.Apply(rule)
			assert.NoError(t, err, "URL should be valid: %s", url)
		}
	})

	t.Run("invalid URLs", func(t *testing.T) {
		invalidURLs := []string{
			"",
			"   ",
			"not-a-url",
			"http://",
			"://example.com",
			"http://",
			"example.com", // missing scheme
		}

		for _, url := range invalidURLs {
			rule := validator.ValidURL("url", url)
			err := validator.Apply(rule)
			assert.Error(t, err, "URL should be invalid: %s", url)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.url", validationErr[0].TranslationKey)
		}
	})
}

func TestValidURLWithScheme(t *testing.T) {
	t.Run("valid URLs with allowed schemes", func(t *testing.T) {
		schemes := []string{"https", "http"}
		validURLs := []string{
			"https://example.com",
			"http://example.com",
		}

		for _, url := range validURLs {
			rule := validator.ValidURLWithScheme("url", url, schemes)
			err := validator.Apply(rule)
			assert.NoError(t, err, "URL should be valid: %s", url)
		}
	})

	t.Run("invalid URLs with wrong schemes", func(t *testing.T) {
		schemes := []string{"https"}
		invalidURLs := []string{
			"http://example.com",   // http not allowed
			"ftp://example.com",    // ftp not allowed
			"file:///path/to/file", // file not allowed
		}

		for _, url := range invalidURLs {
			rule := validator.ValidURLWithScheme("url", url, schemes)
			err := validator.Apply(rule)
			assert.Error(t, err, "URL should be invalid: %s", url)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.url_scheme", validationErr[0].TranslationKey)
		}
	})
}

func TestValidPhone(t *testing.T) {
	t.Run("valid phone numbers", func(t *testing.T) {
		validPhones := []string{
			"+1234567890",
			"+123456789012345", // max length
			"+44123456789",
			"+1-555-123-4567", // with dashes
			"+1 555 123 4567", // with spaces
			"1234567890",      // without plus
		}

		for _, phone := range validPhones {
			rule := validator.ValidPhone("phone", phone)
			err := validator.Apply(rule)
			assert.NoError(t, err, "Phone should be valid: %s", phone)
		}
	})

	t.Run("invalid phone numbers", func(t *testing.T) {
		invalidPhones := []string{
			"",
			"   ",
			"123",                 // too short
			"+0123456789",         // starts with 0 after +
			"abc123456789",        // contains letters
			"+123456789012345678", // too long
			"++1234567890",        // double plus
		}

		for _, phone := range invalidPhones {
			rule := validator.ValidPhone("phone", phone)
			err := validator.Apply(rule)
			assert.Error(t, err, "Phone should be invalid: %s", phone)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.phone", validationErr[0].TranslationKey)
		}
	})
}

func TestValidIPv4(t *testing.T) {
	t.Run("valid IPv4 addresses", func(t *testing.T) {
		validIPs := []string{
			"192.168.1.1",
			"127.0.0.1",
			"255.255.255.255",
			"0.0.0.0",
			"8.8.8.8",
		}

		for _, ip := range validIPs {
			rule := validator.ValidIPv4("ip", ip)
			err := validator.Apply(rule)
			assert.NoError(t, err, "IPv4 should be valid: %s", ip)
		}
	})

	t.Run("invalid IPv4 addresses", func(t *testing.T) {
		invalidIPs := []string{
			"",
			"   ",
			"192.168.1",       // incomplete
			"192.168.1.256",   // out of range
			"192.168.1.1.1",   // too many octets
			"abc.def.ghi.jkl", // not numeric
			"2001:db8::1",     // IPv6
		}

		for _, ip := range invalidIPs {
			rule := validator.ValidIPv4("ip", ip)
			err := validator.Apply(rule)
			assert.Error(t, err, "IPv4 should be invalid: %s", ip)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.ipv4", validationErr[0].TranslationKey)
		}
	})
}

func TestValidIPv6(t *testing.T) {
	t.Run("valid IPv6 addresses", func(t *testing.T) {
		validIPs := []string{
			"2001:db8::1",
			"::1",
			"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
			"2001:db8:85a3::8a2e:370:7334",
			"::ffff:192.0.2.1", // IPv4-mapped IPv6
		}

		for _, ip := range validIPs {
			rule := validator.ValidIPv6("ip", ip)
			err := validator.Apply(rule)
			assert.NoError(t, err, "IPv6 should be valid: %s", ip)
		}
	})

	t.Run("invalid IPv6 addresses", func(t *testing.T) {
		invalidIPs := []string{
			"",
			"   ",
			"192.168.1.1",    // IPv4
			"2001:db8::1::2", // double ::
			"gggg::1",        // invalid hex
			"2001:db8:85a3:0000:0000:8a2e:0370:7334:extra", // too long
		}

		for _, ip := range invalidIPs {
			rule := validator.ValidIPv6("ip", ip)
			err := validator.Apply(rule)
			assert.Error(t, err, "IPv6 should be invalid: %s", ip)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.ipv6", validationErr[0].TranslationKey)
		}
	})
}

func TestValidIP(t *testing.T) {
	t.Run("valid IP addresses", func(t *testing.T) {
		validIPs := []string{
			"192.168.1.1", // IPv4
			"2001:db8::1", // IPv6
			"127.0.0.1",   // IPv4 loopback
			"::1",         // IPv6 loopback
		}

		for _, ip := range validIPs {
			rule := validator.ValidIP("ip", ip)
			err := validator.Apply(rule)
			assert.NoError(t, err, "IP should be valid: %s", ip)
		}
	})

	t.Run("invalid IP addresses", func(t *testing.T) {
		invalidIPs := []string{
			"",
			"   ",
			"not-an-ip",
			"192.168.1.256",
			"gggg::1",
		}

		for _, ip := range invalidIPs {
			rule := validator.ValidIP("ip", ip)
			err := validator.Apply(rule)
			assert.Error(t, err, "IP should be invalid: %s", ip)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.ip", validationErr[0].TranslationKey)
		}
	})
}

func TestValidMAC(t *testing.T) {
	t.Run("valid MAC addresses", func(t *testing.T) {
		validMACs := []string{
			"AA:BB:CC:DD:EE:FF",
			"aa:bb:cc:dd:ee:ff",
			"AA-BB-CC-DD-EE-FF",
			"aa-bb-cc-dd-ee-ff",
			"12:34:56:78:9A:BC",
		}

		for _, mac := range validMACs {
			rule := validator.ValidMAC("mac", mac)
			err := validator.Apply(rule)
			assert.NoError(t, err, "MAC should be valid: %s", mac)
		}
	})

	t.Run("invalid MAC addresses", func(t *testing.T) {
		invalidMACs := []string{
			"",
			"   ",
			"AA:BB:CC:DD:EE",       // too short
			"AA:BB:CC:DD:EE:FF:GG", // too long
			"GG:BB:CC:DD:EE:FF",    // invalid hex
			"AA.BB.CC.DD.EE.FF",    // wrong separator
		}

		for _, mac := range invalidMACs {
			rule := validator.ValidMAC("mac", mac)
			err := validator.Apply(rule)
			assert.Error(t, err, "MAC should be invalid: %s", mac)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.mac", validationErr[0].TranslationKey)
		}
	})
}

func TestValidAlphanumeric(t *testing.T) {
	t.Run("valid alphanumeric strings", func(t *testing.T) {
		validStrings := []string{
			"abc123",
			"ABC123",
			"123456",
			"abcdef",
			"ABCDEF",
			"a1B2c3",
		}

		for _, str := range validStrings {
			rule := validator.ValidAlphanumeric("field", str)
			err := validator.Apply(rule)
			assert.NoError(t, err, "String should be alphanumeric: %s", str)
		}
	})

	t.Run("invalid alphanumeric strings", func(t *testing.T) {
		invalidStrings := []string{
			"",
			"   ",
			"abc 123", // space
			"abc-123", // hyphen
			"abc_123", // underscore
			"abc@123", // special character
			"abc.123", // dot
		}

		for _, str := range invalidStrings {
			rule := validator.ValidAlphanumeric("field", str)
			err := validator.Apply(rule)
			assert.Error(t, err, "String should be invalid: %s", str)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.alphanumeric", validationErr[0].TranslationKey)
		}
	})
}

func TestValidAlpha(t *testing.T) {
	t.Run("valid alphabetic strings", func(t *testing.T) {
		validStrings := []string{
			"abc",
			"ABC",
			"abcDEF",
			"hello",
			"WORLD",
		}

		for _, str := range validStrings {
			rule := validator.ValidAlpha("field", str)
			err := validator.Apply(rule)
			assert.NoError(t, err, "String should be alphabetic: %s", str)
		}
	})

	t.Run("invalid alphabetic strings", func(t *testing.T) {
		invalidStrings := []string{
			"",
			"   ",
			"abc123",  // contains numbers
			"abc 123", // contains space and numbers
			"abc-def", // contains hyphen
			"abc_def", // contains underscore
			"abc@def", // contains special character
		}

		for _, str := range invalidStrings {
			rule := validator.ValidAlpha("field", str)
			err := validator.Apply(rule)
			assert.Error(t, err, "String should be invalid: %s", str)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.alpha", validationErr[0].TranslationKey)
		}
	})
}

func TestValidNumericString(t *testing.T) {
	t.Run("valid numeric strings", func(t *testing.T) {
		validStrings := []string{
			"123",
			"0",
			"999999",
			"123456789",
		}

		for _, str := range validStrings {
			rule := validator.ValidNumericString("field", str)
			err := validator.Apply(rule)
			assert.NoError(t, err, "String should be numeric: %s", str)
		}
	})

	t.Run("invalid numeric strings", func(t *testing.T) {
		invalidStrings := []string{
			"",
			"   ",
			"abc",    // letters
			"123abc", // mixed
			"12.34",  // decimal point
			"12-34",  // hyphen
			"12 34",  // space
			"+123",   // plus sign
			"-123",   // minus sign
		}

		for _, str := range invalidStrings {
			rule := validator.ValidNumericString("field", str)
			err := validator.Apply(rule)
			assert.Error(t, err, "String should be invalid: %s", str)

			validationErr := validator.ExtractValidationErrors(err)
			require.NotNil(t, validationErr)
			assert.Equal(t, "validation.numeric_string", validationErr[0].TranslationKey)
		}
	})
}
