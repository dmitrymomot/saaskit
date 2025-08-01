package useragent

import (
	"strings"
)

// OS detection keyword sets optimized for common traffic patterns
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

// ParseOS identifies operating systems using keyword matching.
// Order reflects typical web traffic patterns: Windows first, then mobile OSes.
func ParseOS(lowerUA string) string {
	if lowerUA == "" {
		return OSUnknown
	}

	// Windows dominates desktop traffic, check it first
	if windowsKeywords.contains(lowerUA) {
		if windowsPhoneKeywords.contains(lowerUA) {
			return OSWindowsPhone
		}
		return OSWindows
	}

	if iOSKeywords.contains(lowerUA) {
		return OSiOS
	}

	if macOSKeywords.contains(lowerUA) {
		return OSMacOS
	}

	// Android check includes fallback for edge cases where keyword detection fails
	if androidKeywords.contains(lowerUA) || strings.Contains(lowerUA, "android") {
		return OSAndroid
	}

	// Less common OSes use keyword sets for maintainability
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
