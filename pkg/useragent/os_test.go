package useragent_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/useragent"

	"github.com/stretchr/testify/assert"
)

// TestParseOSDetection tests the OS detection with various edge cases
func TestParseOSDetection(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			name:     "Windows Phone",
			ua:       "mozilla/5.0 (compatible; msie 10.0; windows phone 8.0; trident/6.0; iuniverse/2.5.0.108; 730; 480; nokia; lumia 730 dual sim)",
			expected: useragent.OSWindowsPhone,
		},
		{
			name:     "HarmonyOS",
			ua:       "mozilla/5.0 (linux; android 10; harmonyos; nova 7 5g) applewebkit/537.36 (khtml, like gecko) chrome/88.0.4324.93 mobile safari/537.36",
			expected: useragent.OSAndroid, // It's being detected as Android based on current implementation precedence
		},
		{
			name:     "FireOS",
			ua:       "mozilla/5.0 (linux; android 9; kfmawi) applewebkit/537.36 (khtml, like gecko) silk/95.3.72 like chrome/95.0.4638.74 safari/537.36",
			expected: useragent.OSAndroid, // It's being detected as Android based on current implementation precedence
		},
		{
			name:     "ChromeOS",
			ua:       "mozilla/5.0 (x11; cros x86_64 14268.67.0) applewebkit/537.36 (khtml, like gecko) chrome/98.0.4758.107 safari/537.36",
			expected: useragent.OSChromeOS,
		},
		{
			name:     "Linux with X11",
			ua:       "mozilla/5.0 (x11; linux x86_64) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.124 safari/537.36",
			expected: useragent.OSLinux,
		},
		{
			name:     "Linux with Debian",
			ua:       "mozilla/5.0 (x11; debian; linux x86_64) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.124 safari/537.36",
			expected: useragent.OSLinux,
		},
		{
			name:     "Unknown OS",
			ua:       "some completely unknown user agent",
			expected: useragent.OSUnknown,
		},
		{
			name:     "Empty UA",
			ua:       "",
			expected: useragent.OSUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := useragent.ParseOS(tc.ua)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestOSAccessors tests the OS accessor methods on UserAgent
func TestOSAccessors(t *testing.T) {
	ua := useragent.New(
		"test-user-agent-string",
		useragent.DeviceTypeDesktop,
		"",
		useragent.OSWindows,
		useragent.BrowserChrome,
		"91.0.4472.124",
	)

	assert.Equal(t, useragent.OSWindows, ua.OS())
}
