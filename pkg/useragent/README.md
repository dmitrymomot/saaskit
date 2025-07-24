# User Agent Parser

A high-performance, memory-efficient package for parsing and analyzing HTTP User-Agent strings.

## Overview

The `useragent` package provides a fast, memory-efficient way to parse User-Agent strings from HTTP requests. It extracts detailed information about browsers, operating systems, and devices with minimal allocations. The package is designed to be thread-safe with no shared mutable state.

## Features

- Optimized for high performance and low memory usage
- Comprehensive device type detection (mobile/tablet/desktop/TV/console/bot)
- Accurate device model identification (iPhone, Samsung, Huawei, etc.)
- Operating system detection with version extraction
- Browser identification with version parsing
- Human-readable session identifiers for logging
- Zero external dependencies (except for Go standard library)

## Usage

### Basic Parsing

```go
import "github.com/dmitrymomot/saaskit/pkg/useragent"

// Parse the User-Agent string from an HTTP request
ua, err := useragent.Parse(r.Header.Get("User-Agent"))
if err != nil {
    // Handle error cases
    switch {
    case errors.Is(err, useragent.ErrEmptyUserAgent):
        // Handle empty user agent
    case errors.Is(err, useragent.ErrMalformedUserAgent):
        // Handle malformed user agent
    case errors.Is(err, useragent.ErrUnknownDevice):
        // Handle unknown device
    default:
        // Handle general error
    }
}

// Access parsed information
deviceType := ua.DeviceType()    // "mobile", "desktop", "tablet", etc.
deviceModel := ua.DeviceModel()  // "iphone", "samsung", "huawei", etc.
os := ua.OS()                    // "ios", "android", "windows", etc.
browserName := ua.BrowserName()  // "chrome", "safari", "firefox", etc.
browserVer := ua.BrowserVer()    // "91.0.4472.124", "15.0", etc.

// Get a concise identifier for logging
sessionID := ua.GetShortIdentifier()  // "Chrome/91.0 (Windows, desktop)"
```

### Device Type Detection

```go
// Check device type with boolean helpers
if ua.IsMobile() {
    // Handle mobile device logic
} else if ua.IsTablet() {
    // Handle tablet device logic
} else if ua.IsDesktop() {
    // Handle desktop logic
} else if ua.IsBot() {
    // Handle bot/crawler logic
} else if ua.IsTV() {
    // Handle smart TV logic
} else if ua.IsConsole() {
    // Handle gaming console logic
}

// Use the device type string with constants
switch ua.DeviceType() {
case useragent.DeviceTypeMobile:
    // Mobile device
case useragent.DeviceTypeTablet:
    // Tablet device
case useragent.DeviceTypeDesktop:
    // Desktop computer
case useragent.DeviceTypeBot:
    // Bot/crawler
case useragent.DeviceTypeTV:
    // Smart TV
case useragent.DeviceTypeConsole:
    // Gaming console
case useragent.DeviceTypeUnknown:
    // Unknown device type
}
```

### Individual Component Parsing

```go
// Parse only the components you need (more efficient)
uaString := r.Header.Get("User-Agent")
lowerUA := strings.ToLower(uaString) // Convert once for efficiency

// Get the device type
deviceType := useragent.ParseDeviceType(lowerUA)

// Get the device model when applicable
if deviceType == useragent.DeviceTypeMobile || deviceType == useragent.DeviceTypeTablet {
    deviceModel := useragent.GetDeviceModel(lowerUA, deviceType)
    // Returns: "iphone", "samsung", etc.
}

// Get just the operating system
os := useragent.ParseOS(lowerUA)
// Returns: "windows", "ios", "android", etc.

// Get just the browser information
browser := useragent.ParseBrowser(lowerUA)
// Returns: Browser{Name: "chrome", Version: "91.0.4472.124"}
```

### Custom User Agents

```go
// Create a custom user agent for testing or modeling
customUA := useragent.New(
    "Custom User Agent String",
    useragent.DeviceTypeMobile,
    useragent.MobileDeviceIPhone,
    useragent.OSiOS,
    useragent.BrowserSafari,
    "15.4"
)

// Use the custom user agent
originalString := customUA.String()           // Original string
identifier := customUA.GetShortIdentifier()   // Short identifier
```

## Best Practices

1. **Performance Optimization**:
    - Convert the user agent string to lowercase only once
    - Use individual component parsers when you only need specific information
    - Reuse the UserAgent object when making multiple checks

2. **Context Usage**:
    - Store the UserAgent in the request context for reuse across handlers
    - Include the short identifier in logs for easy session tracking

3. **Error Handling**:
    - Always check for errors when parsing user agent strings
    - Handle the different error types appropriately (ErrEmptyUserAgent, ErrMalformedUserAgent)
    - Have fallback behaviors for unknown device types

4. **Device Detection**:
    - Combine device type with OS information for the most accurate device detection
    - For responsive design, make decisions based on device type rather than specific models

## API Reference

### Types

```go
// UserAgent represents a parsed user agent string
type UserAgent struct {
    // Private fields not directly accessible
}

// Browser represents browser information
type Browser struct {
    Name    string
    Version string
}
```

### Functions

```go
// Parse a user agent string into a UserAgent struct
func Parse(userAgent string) (UserAgent, error)

// Create a new UserAgent with the specified attributes
func New(ua, deviceType, deviceModel, os, browserName, browserVer string) UserAgent

// Parse only the device type from a lowercase user agent string
func ParseDeviceType(lowerUA string) string

// Determine the device model from a lowercase user agent and device type
func GetDeviceModel(lowerUA, deviceType string) string

// Parse only the operating system from a lowercase user agent string
func ParseOS(lowerUA string) string

// Parse only the browser information from a lowercase user agent string
func ParseBrowser(lowerUA string) Browser
```

### UserAgent Methods

```go
// Get the original user agent string
func (ua UserAgent) String() string
func (ua UserAgent) UserAgent() string

// Get the device type (mobile, desktop, tablet, bot, tv, console)
func (ua UserAgent) DeviceType() string

// Get the device model (iphone, samsung, etc.)
func (ua UserAgent) DeviceModel() string

// Get the operating system name
func (ua UserAgent) OS() string

// Get the browser name
func (ua UserAgent) BrowserName() string

// Get the browser version
func (ua UserAgent) BrowserVer() string

// Get the browser information as a Browser struct
func (ua UserAgent) BrowserInfo() Browser

// Check if the device is mobile
func (ua UserAgent) IsMobile() bool

// Check if the device is a tablet
func (ua UserAgent) IsTablet() bool

// Check if the device is a desktop computer
func (ua UserAgent) IsDesktop() bool

// Check if the device is a bot/crawler
func (ua UserAgent) IsBot() bool

// Check if the device is a smart TV
func (ua UserAgent) IsTV() bool

// Check if the device is a gaming console
func (ua UserAgent) IsConsole() bool

// Check if the device is unknown
func (ua UserAgent) IsUnknown() bool

// Get a short, human-readable identifier for the user agent
func (ua UserAgent) GetShortIdentifier() string
```

### Error Types

```go
// Error variables for specific error conditions
var ErrEmptyUserAgent     = errors.New("empty user agent string")
var ErrMalformedUserAgent = errors.New("malformed user agent string")
var ErrUnsupportedBrowser = errors.New("unsupported browser")
var ErrUnsupportedOS      = errors.New("unsupported operating system")
var ErrUnknownDevice      = errors.New("unknown device type")
var ErrParsingFailed      = errors.New("failed to parse user agent")
```

### Constants

```go
// Device types
const (
    DeviceTypeBot     = "bot"
    DeviceTypeMobile  = "mobile"
    DeviceTypeTablet  = "tablet"
    DeviceTypeDesktop = "desktop"
    DeviceTypeTV      = "tv"
    DeviceTypeConsole = "console"
    DeviceTypeUnknown = "unknown"
)

// Mobile device models
const (
    MobileDeviceIPhone   = "iphone"
    MobileDeviceAndroid  = "android"
    MobileDeviceSamsung  = "samsung"
    MobileDeviceHuawei   = "huawei"
    MobileDeviceXiaomi   = "xiaomi"
    // And many more...
)

// Browser names
const (
    BrowserChrome  = "chrome"
    BrowserFirefox = "firefox"
    BrowserSafari  = "safari"
    BrowserEdge    = "edge"
    BrowserUnknown = "unknown"
    // And many more...
)

// Operating systems
const (
    OSWindows    = "windows"
    OSMacOS      = "macos"
    OSiOS        = "ios"
    OSAndroid    = "android"
    OSLinux      = "linux"
    OSHarmonyOS  = "harmonyos"
    OSUnknown    = "unknown"
    // And many more...
)
```
