package useragent

import (
	"strings"
)

// keywordSet is a set of keywords for fast lookups
type keywordSet map[string]struct{}

// newKeywordSet creates a set from a slice of keywords
func newKeywordSet(keywords ...string) keywordSet {
	result := make(keywordSet, len(keywords))
	for _, word := range keywords {
		result[word] = struct{}{}
	}
	return result
}

// contains checks if the string contains any of the keywords in the set
func (k keywordSet) contains(s string) bool {
	for keyword := range k {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

// Device type keyword maps for faster lookups
var (
	// Maps for device type detection
	botKeywords     = newKeywordSet("bot", "spider", "crawler", "archiver", "ping", "lighthouse", "slurp", "daum", "sogou", "yeti", "facebook", "twitter", "slack", "linkedin", "whatsapp", "telegram", "discord", "camo asset", "generator", "monitor", "analyzer", "validator", "fetcher", "scraper", "check")
	tvKeywords      = newKeywordSet("tv", "appletv", "smarttv", "googletv", "android tv", "webos", "tizen")
	consoleKeywords = newKeywordSet("playstation", "xbox", "nintendo", "wiiu", "switch")
	tabletKeywords  = newKeywordSet("tablet", "kindle", "silk")
	mobileKeywords  = newKeywordSet("mobile", "iphone", "android", "windows phone", "iemobile", "blackberry", "nokia")
	desktopKeywords = newKeywordSet("windows", "macintosh", "mac os x", "linux", "x11", "ubuntu", "fedora", "debian", "chromeos", "cros")

	// Maps for mobile device models
	samsungMobileWords = newKeywordSet("samsung", "sm-g", "sm-a", "sm-n", "samsungbrowser")
	huaweiMobileWords  = newKeywordSet("huawei", "hwa-", "honor", "h60-", "h30-")
	xiaomiMobileWords  = newKeywordSet("xiaomi", "mi ", "redmi", "miui")
	oppoMobileWords    = newKeywordSet("oppo", "cph1", "cph2", "f1f")
	vivoMobileWords    = newKeywordSet("vivo", "viv-", "v1730", "v1731")

	// Maps for tablet device models
	samsungTabletWords = newKeywordSet("sm-t", "gt-p", "sm-p")
	huaweiTabletWords  = newKeywordSet("mediapad", "agassi")
	kindleWords        = newKeywordSet("kindle", "silk", "kftt", "kfjwi")
)

// ParseDeviceType determines the device type from a user agent string
// Optimized version using hash map lookups instead of regex
func ParseDeviceType(lowerUA string) string {
	if lowerUA == "" {
		return DeviceTypeUnknown
	}

	// First check for iOS devices which are very common
	// iPad is always a tablet
	if strings.Contains(lowerUA, "ipad") {
		return DeviceTypeTablet
	}

	// iPhone is always mobile
	if strings.Contains(lowerUA, "iphone") {
		return DeviceTypeMobile
	}

	// Check for bots
	if botKeywords.contains(lowerUA) {
		return DeviceTypeBot
	}

	// Android tablets don't have "Mobile" in their user agent string
	if strings.Contains(lowerUA, "android") {
		if !strings.Contains(lowerUA, "mobile") {
			return DeviceTypeTablet
		} else {
			return DeviceTypeMobile
		}
	}

	// Check for tablets
	if tabletKeywords.contains(lowerUA) {
		return DeviceTypeTablet
	}

	// Check for mobile devices
	if mobileKeywords.contains(lowerUA) {
		return DeviceTypeMobile
	}

	// Check for TV
	if tvKeywords.contains(lowerUA) {
		return DeviceTypeTV
	}

	// Check for gaming consoles
	if consoleKeywords.contains(lowerUA) {
		return DeviceTypeConsole
	}

	// Windows tablets check - must come before desktop check
	if strings.Contains(lowerUA, "windows") &&
		(strings.Contains(lowerUA, "touch") || strings.Contains(lowerUA, "tablet")) {
		return DeviceTypeTablet
	}

	// Check for desktop (most common)
	if desktopKeywords.contains(lowerUA) {
		return DeviceTypeDesktop
	}

	return DeviceTypeUnknown
}

// GetDeviceModel determines the specific device model from a user agent string
// Optimized version using hash map lookups instead of regex
func GetDeviceModel(lowerUA, deviceType string) string {
	// If not a mobile or tablet device, return empty string
	if deviceType != DeviceTypeMobile && deviceType != DeviceTypeTablet {
		return ""
	}

	// Check for mobile device models
	if deviceType == DeviceTypeMobile {
		// Check in order of popularity for early returns
		if strings.Contains(lowerUA, "iphone") {
			return MobileDeviceIPhone
		}

		if samsungMobileWords.contains(lowerUA) {
			return MobileDeviceSamsung
		}

		if huaweiMobileWords.contains(lowerUA) {
			return MobileDeviceHuawei
		}

		if xiaomiMobileWords.contains(lowerUA) {
			return MobileDeviceXiaomi
		}

		if oppoMobileWords.contains(lowerUA) {
			return MobileDeviceOppo
		}

		if vivoMobileWords.contains(lowerUA) {
			return MobileDeviceVivo
		}

		// Generic Android
		if strings.Contains(lowerUA, "android") {
			return MobileDeviceAndroid
		}

		return MobileDeviceUnknown
	}

	// Check for tablet device models
	if deviceType == DeviceTypeTablet {
		// Check in order of popularity
		if strings.Contains(lowerUA, "ipad") {
			return TabletDeviceIPad
		}

		// Surface tablets
		if strings.Contains(lowerUA, "windows") &&
			(strings.Contains(lowerUA, "touch") || strings.Contains(lowerUA, "tablet")) {
			return TabletDeviceSurface
		}

		if strings.Contains(lowerUA, "samsung") || samsungTabletWords.contains(lowerUA) {
			return TabletDeviceSamsung
		}

		if strings.Contains(lowerUA, "huawei") || huaweiTabletWords.contains(lowerUA) {
			return TabletDeviceHuawei
		}

		// Kindle Fire
		if kindleWords.contains(lowerUA) {
			return TabletDeviceKindleFire
		}

		// Generic Android tablet
		if strings.Contains(lowerUA, "android") {
			return TabletDeviceAndroid
		}

		return TabletDeviceUnknown
	}

	return ""
}
