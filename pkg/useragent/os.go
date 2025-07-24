package useragent

import (
	"strings"
)

// OS detection keyword sets for faster lookups
var (
	windowsPhoneKeywords = newKeywordSet("windows phone")
	windowsKeywords      = newKeywordSet("windows")
	iOSKeywords          = newKeywordSet("iphone", "ipad", "ipod")
	macOSKeywords        = newKeywordSet("macintosh", "mac os x")
	harmonyOSKeywords    = newKeywordSet("harmonyos")
	androidKeywords      = newKeywordSet("android")
	fireOSKeywords       = newKeywordSet("kindle", "silk")
	chromeOSKeywords     = newKeywordSet("cros", "chromeos", "chrome os")
	linuxKeywords        = newKeywordSet("linux", "ubuntu", "debian", "fedora", "mint", "x11")
)

// ParseOS determines the operating system from a user agent string
// Optimized version using map-based lookups for faster performance
func ParseOS(lowerUA string) string {
	if lowerUA == "" {
		return OSUnknown
	}

	// Order checks by frequency for typical traffic patterns
	// Windows is most common in desktop traffic
	if windowsKeywords.contains(lowerUA) {
		if windowsPhoneKeywords.contains(lowerUA) {
			return OSWindowsPhone
		}
		return OSWindows
	}

	// iOS and macOS checks
	if iOSKeywords.contains(lowerUA) {
		return OSiOS
	}

	if macOSKeywords.contains(lowerUA) {
		return OSMacOS
	}

	// Android is very common in mobile
	if androidKeywords.contains(lowerUA) || strings.Contains(lowerUA, "android") {
		return OSAndroid
	}

	// Less common OS checks
	// Use map lookups for less common patterns to reduce code size
	if harmonyOSKeywords.contains(lowerUA) {
		return OSHarmonyOS
	}

	if fireOSKeywords.contains(lowerUA) {
		return OSFireOS
	}

	if chromeOSKeywords.contains(lowerUA) {
		return OSChromeOS
	}

	if linuxKeywords.contains(lowerUA) {
		return OSLinux
	}

	return OSUnknown
}
