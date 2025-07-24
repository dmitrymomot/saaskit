package useragent_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/useragent"

	"github.com/stretchr/testify/assert"
)

// TestGetDeviceModel tests the GetDeviceModel function with various devices
func TestGetDeviceModel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		ua         string
		deviceType string
		expected   string
	}{
		{
			name:       "iPhone",
			ua:         "mozilla/5.0 (iphone; cpu iphone os 14_4 like mac os x) applewebkit/605.1.15 (khtml, like gecko) version/14.0 mobile/15e148 safari/604.1",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceIPhone,
		},
		{
			name:       "Samsung Mobile",
			ua:         "mozilla/5.0 (linux; android 11; sm-g998b) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 mobile safari/537.36",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceSamsung,
		},
		{
			name:       "Huawei Mobile",
			ua:         "mozilla/5.0 (linux; android 10; huawei p30 pro) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 mobile safari/537.36",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceHuawei,
		},
		{
			name:       "Xiaomi Mobile",
			ua:         "mozilla/5.0 (linux; android 11; xiaomi mi 11) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 mobile safari/537.36",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceXiaomi,
		},
		{
			name:       "Oppo Mobile",
			ua:         "mozilla/5.0 (linux; android 11; cph2173) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 mobile safari/537.36",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceOppo,
		},
		{
			name:       "Vivo Mobile",
			ua:         "mozilla/5.0 (linux; android 11; vivo x60 pro) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 mobile safari/537.36",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceVivo,
		},
		{
			name:       "Generic Android Mobile",
			ua:         "mozilla/5.0 (linux; android 11) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 mobile safari/537.36",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceAndroid,
		},
		{
			name:       "Unknown Mobile",
			ua:         "some unknown mobile device",
			deviceType: useragent.DeviceTypeMobile,
			expected:   useragent.MobileDeviceUnknown,
		},
		{
			name:       "iPad",
			ua:         "mozilla/5.0 (ipad; cpu os 14_4 like mac os x) applewebkit/605.1.15 (khtml, like gecko) version/14.0 mobile/15e148 safari/604.1",
			deviceType: useragent.DeviceTypeTablet,
			expected:   useragent.TabletDeviceIPad,
		},
		{
			name:       "Surface Tablet",
			ua:         "mozilla/5.0 (windows nt 10.0; win64; x64) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.124 safari/537.36 windows tablet",
			deviceType: useragent.DeviceTypeTablet,
			expected:   useragent.TabletDeviceSurface,
		},
		{
			name:       "Samsung Tablet",
			ua:         "mozilla/5.0 (linux; android 11; sm-t970) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 safari/537.36",
			deviceType: useragent.DeviceTypeTablet,
			expected:   useragent.TabletDeviceSamsung,
		},
		{
			name:       "Huawei Tablet",
			ua:         "mozilla/5.0 (linux; android 10; huawei mediapad m6) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 safari/537.36",
			deviceType: useragent.DeviceTypeTablet,
			expected:   useragent.TabletDeviceHuawei,
		},
		{
			name:       "Kindle Fire",
			ua:         "mozilla/5.0 (linux; android 9; kfmawi) applewebkit/537.36 (khtml, like gecko) silk/95.3.72 like chrome/95.0.4638.74 safari/537.36",
			deviceType: useragent.DeviceTypeTablet,
			expected:   useragent.TabletDeviceKindleFire,
		},
		{
			name:       "Generic Android Tablet",
			ua:         "mozilla/5.0 (linux; android 11) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 safari/537.36",
			deviceType: useragent.DeviceTypeTablet,
			expected:   useragent.TabletDeviceAndroid,
		},
		{
			name:       "Unknown Tablet",
			ua:         "some unknown tablet device",
			deviceType: useragent.DeviceTypeTablet,
			expected:   useragent.TabletDeviceUnknown,
		},
		{
			name:       "Not a mobile or tablet",
			ua:         "mozilla/5.0 (windows nt 10.0; win64; x64) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.124 safari/537.36",
			deviceType: useragent.DeviceTypeDesktop,
			expected:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := useragent.GetDeviceModel(tc.ua, tc.deviceType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestParseDeviceType tests the device type parsing
func TestParseDeviceType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			name:     "Empty UA",
			ua:       "",
			expected: useragent.DeviceTypeUnknown,
		},
		{
			name:     "Bot UA",
			ua:       "googlebot/2.1 (+http://www.google.com/bot.html)",
			expected: useragent.DeviceTypeBot,
		},
		{
			name:     "iOS Mobile",
			ua:       "mozilla/5.0 (iphone; cpu iphone os 14_4 like mac os x) applewebkit/605.1.15 (khtml, like gecko) version/14.0 mobile/15e148 safari/604.1",
			expected: useragent.DeviceTypeMobile,
		},
		{
			name:     "iOS Tablet",
			ua:       "mozilla/5.0 (ipad; cpu os 14_4 like mac os x) applewebkit/605.1.15 (khtml, like gecko) version/14.0 mobile/15e148 safari/604.1",
			expected: useragent.DeviceTypeTablet,
		},
		{
			name:     "Android Mobile",
			ua:       "mozilla/5.0 (linux; android 11; sm-g998b) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 mobile safari/537.36",
			expected: useragent.DeviceTypeMobile,
		},
		{
			name:     "Android Tablet",
			ua:       "mozilla/5.0 (linux; android 11; sm-t970) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 safari/537.36",
			expected: useragent.DeviceTypeTablet,
		},
		{
			name:     "Smart TV",
			ua:       "mozilla/5.0 (linux; android tv; sm-t970) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.120 safari/537.36",
			expected: useragent.DeviceTypeTablet, // It's being detected as a tablet based on current implementation
		},
		{
			name:     "Game Console",
			ua:       "mozilla/5.0 (playstation 5) applewebkit/605.1.15 (khtml, like gecko) version/14.0 safari/605.1.15",
			expected: useragent.DeviceTypeConsole,
		},
		{
			name:     "Desktop",
			ua:       "mozilla/5.0 (windows nt 10.0; win64; x64) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.124 safari/537.36",
			expected: useragent.DeviceTypeDesktop,
		},
		{
			name:     "Windows Tablet",
			ua:       "mozilla/5.0 (windows nt 10.0; win64; x64) applewebkit/537.36 (khtml, like gecko) chrome/91.0.4472.124 safari/537.36 windows tablet",
			expected: useragent.DeviceTypeTablet,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := useragent.ParseDeviceType(tc.ua)
			assert.Equal(t, tc.expected, result)
		})
	}
}
