package useragent_test

import (
	"strings"
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/useragent"
)

var (
	// Common user agent strings for benchmarking
	chromeDesktopUA  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"
	safariMobileUA   = "Mozilla/5.0 (iPhone; CPU iPhone OS 14_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1"
	edgeBrowserUA    = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36 Edg/91.0.864.59"
	androidTabletUA  = "Mozilla/5.0 (Linux; Android 11; SM-T500) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.120 Safari/537.36"
	botUA            = "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	samsungBrowserUA = "Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/14.0 Chrome/87.0.4280.141 Mobile Safari/537.36"
	ucBrowserUA      = "Mozilla/5.0 (Linux; U; Android 11; en-US; SM-A515F) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/78.0.3904.108 UCBrowser/13.4.0.1306 Mobile Safari/537.36"
	emptyUA          = ""
)

// Helper function to avoid compiler optimizations removing the function call
var (
	result useragent.UserAgent
	err    error
)

// Benchmark Parse for Chrome Desktop
func BenchmarkParse_ChromeDesktop(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(chromeDesktopUA)
	}
}

// Benchmark Parse for Safari Mobile
func BenchmarkParse_SafariMobile(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(safariMobileUA)
	}
}

// Benchmark Parse for Edge Browser
func BenchmarkParse_EdgeBrowser(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(edgeBrowserUA)
	}
}

// Benchmark Parse for Android Tablet
func BenchmarkParse_AndroidTablet(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(androidTabletUA)
	}
}

// Benchmark Parse for Bot
func BenchmarkParse_Bot(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(botUA)
	}
}

// Benchmark Parse for Samsung Browser
func BenchmarkParse_SamsungBrowser(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(samsungBrowserUA)
	}
}

// Benchmark Parse for UC Browser
func BenchmarkParse_UCBrowser(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(ucBrowserUA)
	}
}

// Benchmark Parse for Empty User Agent
func BenchmarkParse_EmptyUA(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err = useragent.Parse(emptyUA)
	}
}

// Benchmark all Parse functions together
func BenchmarkParse_All(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Mix of different user agents to simulate real-world usage
		switch i % 8 {
		case 0:
			result, err = useragent.Parse(chromeDesktopUA)
		case 1:
			result, err = useragent.Parse(safariMobileUA)
		case 2:
			result, err = useragent.Parse(edgeBrowserUA)
		case 3:
			result, err = useragent.Parse(androidTabletUA)
		case 4:
			result, err = useragent.Parse(botUA)
		case 5:
			result, err = useragent.Parse(samsungBrowserUA)
		case 6:
			result, err = useragent.Parse(ucBrowserUA)
		case 7:
			result, err = useragent.Parse(emptyUA)
		}
	}
}

// Benchmark for ParseDeviceType function
func BenchmarkParseDeviceType(b *testing.B) {
	userAgents := []string{
		chromeDesktopUA,
		safariMobileUA,
		androidTabletUA,
		botUA,
		samsungBrowserUA,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use modulo to cycle through the user agents
		ua := userAgents[i%len(userAgents)]
		_ = useragent.ParseDeviceType(strings.ToLower(ua))
	}
}

// Benchmark for GetDeviceModel function
func BenchmarkGetDeviceModel(b *testing.B) {
	testCases := []struct {
		ua         string
		deviceType string
	}{
		{chromeDesktopUA, useragent.DeviceTypeDesktop},
		{safariMobileUA, useragent.DeviceTypeMobile},
		{androidTabletUA, useragent.DeviceTypeTablet},
		{samsungBrowserUA, useragent.DeviceTypeMobile},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use modulo to cycle through the test cases
		tc := testCases[i%len(testCases)]
		_ = useragent.GetDeviceModel(strings.ToLower(tc.ua), tc.deviceType)
	}
}

// Benchmark for ParseOS function
func BenchmarkParseOS(b *testing.B) {
	userAgents := []string{
		chromeDesktopUA,
		safariMobileUA,
		androidTabletUA,
		botUA,
		samsungBrowserUA,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use modulo to cycle through the user agents
		ua := userAgents[i%len(userAgents)]
		_ = useragent.ParseOS(strings.ToLower(ua))
	}
}

// Benchmark for ParseBrowser function
func BenchmarkParseBrowser(b *testing.B) {
	userAgents := []string{
		chromeDesktopUA,
		safariMobileUA,
		edgeBrowserUA,
		androidTabletUA,
		samsungBrowserUA,
		ucBrowserUA,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use modulo to cycle through the user agents
		ua := userAgents[i%len(userAgents)]
		_ = useragent.ParseBrowser(strings.ToLower(ua))
	}
}

// Benchmark for GetShortIdentifier function
func BenchmarkGetShortIdentifier(b *testing.B) {
	// Create user agents
	chromeUA, _ := useragent.Parse(chromeDesktopUA)
	iphoneUA, _ := useragent.Parse(safariMobileUA)
	botUserAgent, _ := useragent.Parse(botUA)

	userAgents := []useragent.UserAgent{
		chromeUA,
		iphoneUA,
		botUserAgent,
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use modulo to cycle through the user agents
		ua := userAgents[i%len(userAgents)]
		_ = ua.GetShortIdentifier()
	}
}
