package useragent_test

import (
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/useragent"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOS(t *testing.T) {
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
			result := useragent.ParseOS(strings.ToLower(tc.ua))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseBrowser(t *testing.T) {
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
			result := useragent.ParseBrowser(strings.ToLower(tc.ua))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParseUserAgent(t *testing.T) {
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
			name: "Empty UA",
			ua:   "",
			expected: useragent.New(
				"",
				useragent.DeviceTypeUnknown,
				"", // deviceModel
				useragent.OSUnknown,
				useragent.BrowserUnknown,
				"",
			),
			expectedErr: useragent.ErrEmptyUserAgent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
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
			result := tc.ua.GetShortIdentifier()
			assert.Equal(t, tc.expected, result)
		})
	}
}
