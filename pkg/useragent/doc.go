// Package useragent provides fast and memory-efficient parsing and classification
// of HTTP User-Agent strings.
//
// It identifies:
//   - Device type – desktop, mobile, tablet, TV, console, bot or unknown
//   - Device model – iPhone, Samsung, Huawei, etc. (when available)
//   - Operating system – Windows, macOS, iOS, Android, Linux, ChromeOS, etc.
//   - Browser name and version – Chrome, Safari, Firefox, …
//
// In addition, helper methods make it trivial to test whether a UA belongs to a
// particular class (IsBot, IsMobile, IsDesktop, …) and to build short human-readable
// identifiers for logging and analytics.
//
// Parsing is performed with plain-string look-ups and pre-compiled regular
// expressions – no heavyweight dependency on the upstream Chromium UA-parser –
// which keeps allocations low and makes the package suitable for high-traffic
// servers and edge environments.
//
// # Architecture
//
// The high-level entry point is Parse, which orchestrates dedicated parsers for
// device type, operating system and browser. Each of those lives in its own
// file (device.go, os.go, browser.go) and relies on curated keyword sets to
// avoid expensive regex evaluations where possible. Common string constants and
// public enumerations reside in constants.go. All domain-specific errors are
// grouped in errors.go.
//
//	┌────────────┐  UA string ┌───────────────┐
//	│    Parse    │──────────▶│  device.go    │──┐
//	└────────────┘            └───────────────┘  │
//	    ▲   │                                    │
//	    │   │         ┌───────────────┐          │
//	    │   └────────▶│   os.go       │──────────┼──► final UserAgent struct
//	    │             └───────────────┘          │
//	    │                                        │
//	    │             ┌───────────────┐          │
//	    └────────────▶│  browser.go   │──────────┘
//	                  └───────────────┘
//
// # Usage
//
// Import the package:
//
//	import "github.com/dmitrymomot/saaskit/pkg/useragent"
//
// Parse an incoming request’s UA and inspect the result:
//
//	ua, err := useragent.Parse(r.UserAgent())
//	if err != nil {
//	    // Handle ErrEmptyUserAgent, ErrUnknownDevice, …
//	}
//
//	log.Printf("client=%s", ua.GetShortIdentifier())
//
//	if ua.IsBot() {
//	    // throttle, skip heavy rendering, …
//	}
//
//	if ua.DeviceType() == useragent.DeviceTypeMobile {
//	    // serve mobile-optimised assets
//	}
//
// # Error Handling
//
// Parse may return the following sentinel errors, all export-visible via
// errors.Is: ErrEmptyUserAgent, ErrMalformedUserAgent, ErrUnsupportedBrowser,
// ErrUnsupportedOS, ErrUnknownDevice and ErrParsingFailed.
//
// # Performance
//
// • Zero allocations when called with an already lower-cased UA string.
// • Single pass over the input for most common paths.
// • Hot keyword sets implemented by map[string]struct{} look-ups.
//
// Benchmarks live next to the implementation (benchmark_test.go) and show sub-µs
// parsing times on 2024-class CPUs.
//
// # Examples
//
// See the example_test.go files for self-verifying usage scenarios covering all
// supported device classes.
package useragent
