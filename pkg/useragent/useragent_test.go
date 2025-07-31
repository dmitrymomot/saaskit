package useragent_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/useragent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			name:     "Windows 10",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expected: useragent.OSWindows,
		},
		{
			name:     "macOS",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expected: useragent.OSMacOS,
		},
		{
			name:     "iOS",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
			expected: useragent.OSiOS,
		},
		{
			name:     "Android",
			ua:       "Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Mobile Safari/537.36",
			expected: useragent.OSAndroid,
		},
		{
			name:     "Linux",
			ua:       "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:89.0) Gecko/20100101 Firefox/89.0",
			expected: useragent.OSLinux,
		},
		{
			name:     "Empty UA",
			ua:       "",
			expected: useragent.OSUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := useragent.ParseOS(strings.ToLower(tc.ua))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseBrowser(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ua       string
		expected useragent.Browser
	}{
		{
			name: "Chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expected: useragent.Browser{
				Name:    useragent.BrowserChrome,
				Version: "91.0.4472.124",
			},
		},
		{
			name: "Firefox",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:89.0) Gecko/20100101 Firefox/89.0",
			expected: useragent.Browser{
				Name:    useragent.BrowserFirefox,
				Version: "89.0",
			},
		},
		{
			name: "Safari",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0.3 Safari/605.1.15",
			expected: useragent.Browser{
				Name:    useragent.BrowserSafari,
				Version: "14.0.3",
			},
		},
		{
			name: "Edge",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59",
			expected: useragent.Browser{
				Name:    useragent.BrowserEdge,
				Version: "91.0.864.59",
			},
		},
		{
			name: "Empty UA",
			ua:   "",
			expected: useragent.Browser{
				Name:    useragent.BrowserUnknown,
				Version: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := useragent.ParseBrowser(strings.ToLower(tc.ua))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseUserAgent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		ua          string
		expected    useragent.UserAgent
		expectedErr error
	}{
		{
			name: "Desktop Chrome on Windows",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
			expected: useragent.New(
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				useragent.DeviceTypeDesktop,
				"", // deviceModel
				useragent.OSWindows,
				useragent.BrowserChrome,
				"91.0.4472.124",
			),
			expectedErr: nil,
		},
		{
			name: "Mobile Safari on iPhone",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
			expected: useragent.New(
				"Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
				useragent.DeviceTypeMobile,
				useragent.MobileDeviceIPhone, // deviceModel
				useragent.OSiOS,
				useragent.BrowserSafari,
				"14.0",
			),
			expectedErr: nil,
		},
		{
			name: "Googlebot",
			ua:   "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			expected: useragent.New(
				"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
				useragent.DeviceTypeBot,
				"", // deviceModel
				useragent.OSUnknown,
				useragent.BrowserUnknown,
				"",
			),
			expectedErr: nil,
		},
		{
			name:        "Empty UA",
			ua:          "",
			expected:    useragent.UserAgent{}, // Zero value for empty UA
			expectedErr: useragent.ErrEmptyUserAgent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := useragent.Parse(tc.ua)

			if tc.expectedErr != nil {
				assert.Equal(t, tc.expectedErr, err)
			} else {
				require.NoError(t, err)
			}

			// Use getter methods to compare values
			assert.Equal(t, tc.expected.UserAgent(), result.UserAgent())
			assert.Equal(t, tc.expected.DeviceType(), result.DeviceType())
			assert.Equal(t, tc.expected.OS(), result.OS())
			assert.Equal(t, tc.expected.BrowserName(), result.BrowserName())
			assert.Equal(t, tc.expected.BrowserVer(), result.BrowserVer())
			assert.Equal(t, tc.expected.IsBot(), result.IsBot())
			assert.Equal(t, tc.expected.IsMobile(), result.IsMobile())
			assert.Equal(t, tc.expected.IsDesktop(), result.IsDesktop())
			assert.Equal(t, tc.expected.IsTablet(), result.IsTablet())
			assert.Equal(t, tc.expected.IsUnknown(), result.IsUnknown())
		})
	}
}

// TestNewUserAgent tests the NewUserAgent constructor
func TestNewUserAgent(t *testing.T) {
	t.Parallel()
	ua := useragent.New(
		"test-ua",
		useragent.DeviceTypeMobile,
		useragent.MobileDeviceIPhone, // Added device model
		useragent.OSiOS,
		useragent.BrowserSafari,
		"15.0",
	)

	assert.Equal(t, "test-ua", ua.UserAgent())
	assert.Equal(t, useragent.DeviceTypeMobile, ua.DeviceType())
	assert.Equal(t, useragent.MobileDeviceIPhone, ua.DeviceModel())
	assert.Equal(t, useragent.OSiOS, ua.OS())
	assert.Equal(t, useragent.BrowserSafari, ua.BrowserName())
	assert.Equal(t, "15.0", ua.BrowserVer())
	assert.True(t, ua.IsMobile())
	assert.False(t, ua.IsDesktop())
	assert.False(t, ua.IsTablet())
	assert.False(t, ua.IsBot())
	assert.False(t, ua.IsUnknown())
	assert.False(t, ua.IsTV())
	assert.False(t, ua.IsConsole())
}

// TestGetShortIdentifier tests the GetShortIdentifier method
func TestGetShortIdentifier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ua       useragent.UserAgent
		expected string
	}{
		{
			name: "Chrome on Windows",
			ua: useragent.New(
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
				useragent.DeviceTypeDesktop,
				"", // deviceModel
				useragent.OSWindows,
				useragent.BrowserChrome,
				"91.0.4472.124",
			),
			expected: "Chrome/91.0.44721 (Windows, desktop)",
		},
		{
			name: "Safari on iOS",
			ua: useragent.New(
				"Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
				useragent.DeviceTypeMobile,
				useragent.MobileDeviceIPhone, // deviceModel
				useragent.OSiOS,
				useragent.BrowserSafari,
				"14.0",
			),
			expected: "Safari/14.0 (iOS, mobile)",
		},
		{
			name: "Bot",
			ua: useragent.New(
				"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
				useragent.DeviceTypeBot,
				"", // deviceModel
				useragent.OSUnknown,
				useragent.BrowserUnknown,
				"",
			),
			expected: "Bot: Googlebot",
		},
		{
			name: "All Unknown Components - Empty Strings",
			ua: useragent.New(
				"",
				"",
				"", // deviceModel
				"",
				"",
				"",
			),
			expected: "Unknown device",
		},
		{
			name: "All Unknown Components - Unknown Constants",
			ua: useragent.New(
				"",
				useragent.DeviceTypeUnknown,
				"", // deviceModel
				useragent.OSUnknown,
				useragent.BrowserUnknown,
				"",
			),
			expected: "Unknown device",
		},
		{
			name: "Unknown Browser but Known OS and Device",
			ua: useragent.New(
				"Some obscure browser",
				useragent.DeviceTypeDesktop,
				"", // deviceModel
				useragent.OSWindows,
				useragent.BrowserUnknown,
				"",
			),
			expected: "Windows desktop",
		},
		{
			name: "Known Browser but Unknown OS and Device",
			ua: useragent.New(
				"Partial information",
				useragent.DeviceTypeUnknown,
				"", // deviceModel
				useragent.OSUnknown,
				useragent.BrowserChrome,
				"100.0",
			),
			expected: "Chrome/100.0 (Unknown OS)",
		},
		{
			name: "Browser with long version",
			ua: useragent.New(
				"Browser with long version string",
				useragent.DeviceTypeDesktop,
				"", // deviceModel
				useragent.OSWindows,
				useragent.BrowserFirefox,
				"100.0.12345.67890.beta",
			),
			expected: "Firefox/100.0.1234 (Windows desktop)",
		},
		{
			name: "Browser with version ending with dot",
			ua: useragent.New(
				"Browser with version ending with dot",
				useragent.DeviceTypeDesktop,
				"", // deviceModel
				useragent.OSWindows,
				useragent.BrowserFirefox,
				"100.0.12345.",
			),
			expected: "Firefox/100.0.1234 (Windows desktop)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := tc.ua.GetShortIdentifier()
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestParseComplexBots tests complex bot detection scenarios using direct t.Run() style
func TestParseComplexBots(t *testing.T) {
	t.Parallel()

	t.Run("Googlebot with complex pattern", func(t *testing.T) {
		t.Parallel()
		ua := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
		result, err := useragent.Parse(ua)
		require.NoError(t, err)
		assert.True(t, result.IsBot())
		assert.Equal(t, "Bot: Googlebot", result.GetShortIdentifier())
	})

	t.Run("Bot with unusual casing and special characters", func(t *testing.T) {
		t.Parallel()
		ua := "MyCompany-WebCrawler_v2.1.3-bot (+https://example.com/crawler)"
		result, err := useragent.Parse(ua)
		require.NoError(t, err)
		assert.True(t, result.IsBot())
		assert.Contains(t, result.GetShortIdentifier(), "Bot:")
	})

	t.Run("Bot pattern at end of UA string", func(t *testing.T) {
		t.Parallel()
		ua := "CustomUserAgent/1.0 (compatible; Linux x86_64) SearchBot"
		result, err := useragent.Parse(ua)
		require.NoError(t, err)
		assert.True(t, result.IsBot())
	})
}

// TestParseMalformedUserAgents tests edge cases with malformed user agents
func TestParseMalformedUserAgents(t *testing.T) {
	t.Parallel()

	t.Run("Empty user agent string", func(t *testing.T) {
		t.Parallel()
		result, err := useragent.Parse("")
		require.Error(t, err)
		assert.True(t, errors.Is(err, useragent.ErrEmptyUserAgent))
		assert.Equal(t, "Unknown device", result.GetShortIdentifier())
	})

	t.Run("Random gibberish without any recognizable patterns", func(t *testing.T) {
		t.Parallel()
		ua := "!@#$%^&*()_+-={}[]|:;<>?,./"
		_, err := useragent.Parse(ua)
		require.Error(t, err)
		// The gibberish is detected as an unknown device, not malformed
		assert.True(t, errors.Is(err, useragent.ErrUnknownDevice))
	})

	t.Run("Very long user agent exceeding reasonable limits", func(t *testing.T) {
		t.Parallel()
		// Create a very long string that might trigger length checks
		longUA := strings.Repeat("Mozilla/5.0 ", 100)
		result, err := useragent.Parse(longUA)
		// Should handle gracefully without panicking
		assert.NotNil(t, result)
		if err != nil {
			assert.True(t, errors.Is(err, useragent.ErrMalformedUserAgent) || errors.Is(err, useragent.ErrUnknownDevice))
		}
	})
}

// TestParseMultiStepVerification tests scenarios requiring multiple parsing steps
func TestParseMultiStepVerification(t *testing.T) {
	t.Parallel()

	t.Run("Mobile device with all components detected", func(t *testing.T) {
		t.Parallel()
		ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1"
		result, err := useragent.Parse(ua)
		require.NoError(t, err)

		// Verify each component was parsed correctly
		assert.Equal(t, useragent.DeviceTypeMobile, result.DeviceType())
		assert.Equal(t, "iphone", result.DeviceModel())
		assert.Equal(t, useragent.OSiOS, result.OS())
		assert.Equal(t, useragent.BrowserSafari, result.BrowserName())
		assert.NotEmpty(t, result.BrowserVer())

		// Verify the short identifier format
		identifier := result.GetShortIdentifier()
		assert.Contains(t, identifier, "Safari")
		assert.Contains(t, identifier, "iOS")
		assert.Contains(t, identifier, "mobile")
	})

	t.Run("Desktop with partial information", func(t *testing.T) {
		t.Parallel()
		// UA with browser but missing OS details
		ua := "CustomBrowser/1.0"
		result, err := useragent.Parse(ua)
		// Should not error for partial information
		if err != nil {
			assert.True(t, errors.Is(err, useragent.ErrUnknownDevice))
		}
		assert.NotNil(t, result)
	})

	t.Run("Tablet device detection and formatting", func(t *testing.T) {
		t.Parallel()
		ua := "Mozilla/5.0 (iPad; CPU OS 13_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.1.1 Mobile/15E148 Safari/604.1"
		result, err := useragent.Parse(ua)
		require.NoError(t, err)

		assert.Equal(t, useragent.DeviceTypeTablet, result.DeviceType())
		assert.Equal(t, "ipad", result.DeviceModel())
		assert.Equal(t, useragent.OSiOS, result.OS())
	})
}
