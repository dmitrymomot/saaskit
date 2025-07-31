package useragent

import (
	"strings"
)

// keywordSet optimizes keyword lookups using map structure for O(1) access
type keywordSet map[string]struct{}

func newKeywordSet(keywords ...string) keywordSet {
	result := make(keywordSet, len(keywords))
	for _, word := range keywords {
		result[word] = struct{}{}
	}
	return result
}

func (k keywordSet) contains(s string) bool {
	for keyword := range k {
		if strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

// Keyword sets organized by device type for efficient classification.
// Bot detection includes social media crawlers and monitoring tools.
var (
	botKeywords     = newKeywordSet("bot", "spider", "crawler", "archiver", "ping", "lighthouse", "slurp", "daum", "sogou", "yeti", "facebook", "twitter", "slack", "linkedin", "whatsapp", "telegram", "discord", "camo asset", "generator", "monitor", "analyzer", "validator", "fetcher", "scraper", "check")
	tvKeywords      = newKeywordSet("tv", "appletv", "smarttv", "googletv", "android tv", "webos", "tizen")
	consoleKeywords = newKeywordSet("playstation", "xbox", "nintendo", "wiiu", "switch")
	tabletKeywords  = newKeywordSet("tablet", "kindle", "silk")
	mobileKeywords  = newKeywordSet("mobile", "iphone", "android", "windows phone", "iemobile", "blackberry", "nokia")
	desktopKeywords = newKeywordSet("windows", "macintosh", "mac os x", "linux", "x11", "ubuntu", "fedora", "debian", "chromeos", "cros")

	// Mobile device brand detection based on common UA patterns
	samsungMobileWords = newKeywordSet("samsung", "sm-g", "sm-a", "sm-n", "samsungbrowser")
	huaweiMobileWords  = newKeywordSet("huawei", "hwa-", "honor", "h60-", "h30-")
	xiaomiMobileWords  = newKeywordSet("xiaomi", "mi ", "redmi", "miui")
	oppoMobileWords    = newKeywordSet("oppo", "cph1", "cph2", "f1f")
	vivoMobileWords    = newKeywordSet("vivo", "viv-", "v1730", "v1731")

	// Tablet device brand detection patterns
	samsungTabletWords = newKeywordSet("sm-t", "gt-p", "sm-p")
	huaweiTabletWords  = newKeywordSet("mediapad", "agassi")
	kindleWords        = newKeywordSet("kindle", "silk", "kftt", "kfjwi")
)

// ParseDeviceType classifies devices using fast string matching.
// Order matters: iOS devices first (common), then Android logic, then fallbacks.
func ParseDeviceType(lowerUA string) string {
	if lowerUA == "" {
		return DeviceTypeUnknown
	}

	// iOS devices have unambiguous identifiers
	if strings.Contains(lowerUA, "ipad") {
		return DeviceTypeTablet
	}

	if strings.Contains(lowerUA, "iphone") {
		return DeviceTypeMobile
	}

	if botKeywords.contains(lowerUA) {
		return DeviceTypeBot
	}

	// Android tablets omit 'Mobile' keyword, unlike phones
	if strings.Contains(lowerUA, "android") {
		if !strings.Contains(lowerUA, "mobile") {
			return DeviceTypeTablet
		} else {
			return DeviceTypeMobile
		}
	}

	if tabletKeywords.contains(lowerUA) {
		return DeviceTypeTablet
	}

	if mobileKeywords.contains(lowerUA) {
		return DeviceTypeMobile
	}

	if tvKeywords.contains(lowerUA) {
		return DeviceTypeTV
	}

	if consoleKeywords.contains(lowerUA) {
		return DeviceTypeConsole
	}

	// Windows tablets require special detection before general desktop matching
	if strings.Contains(lowerUA, "windows") &&
		(strings.Contains(lowerUA, "touch") || strings.Contains(lowerUA, "tablet")) {
		return DeviceTypeTablet
	}

	if desktopKeywords.contains(lowerUA) {
		return DeviceTypeDesktop
	}

	return DeviceTypeUnknown
}

// GetDeviceModel identifies specific device brands for mobile and tablet devices.
// Returns empty string for other device types since model detection isn't meaningful.
func GetDeviceModel(lowerUA, deviceType string) string {
	if deviceType != DeviceTypeMobile && deviceType != DeviceTypeTablet {
		return ""
	}

	if deviceType == DeviceTypeMobile {
		// Ordered by global market share for faster common-case detection
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

		// Fallback for unrecognized Android devices
		if strings.Contains(lowerUA, "android") {
			return MobileDeviceAndroid
		}

		return MobileDeviceUnknown
	}

	if deviceType == DeviceTypeTablet {
		// Ordered by tablet market share
		if strings.Contains(lowerUA, "ipad") {
			return TabletDeviceIPad
		}

		// Microsoft Surface detection via Windows + touch indicators
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

		// Amazon's Android-based tablets
		if kindleWords.contains(lowerUA) {
			return TabletDeviceKindleFire
		}

		// Fallback for unrecognized Android tablets
		if strings.Contains(lowerUA, "android") {
			return TabletDeviceAndroid
		}

		return TabletDeviceUnknown
	}

	return ""
}
