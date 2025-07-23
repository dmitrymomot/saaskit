package useragent_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/useragent"

	"github.com/stretchr/testify/assert"
)

// TestBrowserInfo tests the BrowserInfo method
func TestBrowserInfo(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ua       useragent.UserAgent
		expected useragent.Browser
	}{
		{
			name: "Chrome browser",
			ua: useragent.New(
				"test-user-agent-string",
				useragent.DeviceTypeDesktop,
				"",
				useragent.OSWindows,
				useragent.BrowserChrome,
				"91.0.4472.124",
			),
			expected: useragent.Browser{
				Name:    useragent.BrowserChrome,
				Version: "91.0.4472.124",
			},
		},
		{
			name: "Firefox browser",
			ua: useragent.New(
				"test-user-agent-string",
				useragent.DeviceTypeDesktop,
				"",
				useragent.OSWindows,
				useragent.BrowserFirefox,
				"89.0",
			),
			expected: useragent.Browser{
				Name:    useragent.BrowserFirefox,
				Version: "89.0",
			},
		},
		{
			name: "Unknown browser",
			ua: useragent.New(
				"test-user-agent-string",
				useragent.DeviceTypeDesktop,
				"",
				useragent.OSWindows,
				useragent.BrowserUnknown,
				"",
			),
			expected: useragent.Browser{
				Name:    useragent.BrowserUnknown,
				Version: "",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			browserInfo := tc.ua.BrowserInfo()
			assert.Equal(t, tc.expected.Name, browserInfo.Name)
			assert.Equal(t, tc.expected.Version, browserInfo.Version)
		})
	}
}

// Additional tests for Browser parsing are already in useragent_test.go (TestParseBrowser)
