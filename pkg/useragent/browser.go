package useragent

import (
	"regexp"
	"strings"
)

type Browser struct {
	Name    string
	Version string
}

// BrowserPattern defines a pattern for detecting a browser
type BrowserPattern struct {
	Name      string
	Keywords  []string
	Excludes  []string
	Regex     *regexp.Regexp
	OrderHint int
}

func extractVersion(ua string, regex *regexp.Regexp) string {
	matches := regex.FindStringSubmatch(ua)
	if len(matches) > 1 {
		version := matches[1]
		// Prevent memory issues from pathological version strings
		if len(version) > 20 {
			version = version[:20]
		}
		return version
	}
	return ""
}

func matchPattern(ua string, pattern BrowserPattern) bool {
	// Edge masquerades as Chrome in its UA string, requiring special handling
	if pattern.Name == BrowserEdge {
		for _, keyword := range pattern.Keywords {
			if strings.Contains(ua, keyword) {
				return true
			}
		}
		return false
	}

	// All required keywords must be present
	for _, keyword := range pattern.Keywords {
		if !strings.Contains(ua, keyword) {
			return false
		}
	}
	// Any excluded keyword disqualifies the match
	for _, exclude := range pattern.Excludes {
		if strings.Contains(ua, exclude) {
			return false
		}
	}
	return true
}

// Browser detection patterns ordered by specificity to avoid false positives.
// More specific browsers (Edge, Samsung) must be checked before generic ones (Chrome).
var browserPatterns = []BrowserPattern{
	{
		Name:      BrowserEdge,
		Keywords:  []string{"edg/", "edge/"},
		Regex:     regexp.MustCompile(`(?i)(?:edge|edg)[/ ]([\d.]+)`),
		OrderHint: 10,
	},
	{
		Name:      BrowserSamsung,
		Keywords:  []string{"samsungbrowser"},
		Regex:     regexp.MustCompile(`(?i)samsungbrowser[/\s]([\d.]+)`),
		OrderHint: 20,
	},
	{
		Name:      BrowserUC,
		Keywords:  []string{"ucbrowser"},
		Regex:     regexp.MustCompile(`(?i)ucbrowser[/\s]([\d.]+)`),
		OrderHint: 30,
	},
	{
		Name:      BrowserQQ,
		Keywords:  []string{"qqbrowser"},
		Regex:     regexp.MustCompile(`(?i)(?:qqbrowser|qq)[/\s]([\d.]+)`),
		OrderHint: 40,
	},
	{
		Name:      BrowserQQ, // Alternative QQ browser detection
		Keywords:  []string{"qq", "browser"},
		Regex:     regexp.MustCompile(`(?i)(?:qqbrowser|qq)[/\s]([\d.]+)`),
		OrderHint: 45,
	},
	{
		Name:      BrowserHuawei,
		Keywords:  []string{"huaweibrowser"},
		Regex:     regexp.MustCompile(`(?i)huaweibrowser[/\s]([\d.]+)`),
		OrderHint: 50,
	},
	{
		Name:      BrowserVivo,
		Keywords:  []string{"vivobrowser"},
		Regex:     regexp.MustCompile(`(?i)vivobrowser[/\s]([\d.]+)`),
		OrderHint: 60,
	},
	{
		Name:      BrowserMIUI,
		Keywords:  []string{"miuibrowser"},
		Regex:     regexp.MustCompile(`(?i)miuibrowser[/\s]([\d.]+)`),
		OrderHint: 70,
	},
	{
		Name:      BrowserMIUI, // Alternative MIUI browser detection
		Keywords:  []string{"miui"},
		Regex:     regexp.MustCompile(`(?i)miui[/\s]([\d.]+)`),
		OrderHint: 75,
	},
	{
		Name:      BrowserYandex,
		Keywords:  []string{"yabrowser"},
		Regex:     regexp.MustCompile(`(?i)yabrowser[/\s]([\d.]+)`),
		OrderHint: 80,
	},
	{
		Name:      BrowserYandex, // Alternative Yandex browser detection
		Keywords:  []string{"yandexbrowser"},
		Regex:     regexp.MustCompile(`(?i)yandexbrowser[/\s]([\d.]+)`),
		OrderHint: 85,
	},
	{
		Name:      BrowserVivaldi,
		Keywords:  []string{"vivaldi"},
		Regex:     regexp.MustCompile(`(?i)vivaldi[/\s]([\d.]+)`),
		OrderHint: 90,
	},
	{
		Name:      BrowserBrave,
		Keywords:  []string{"brave"},
		Regex:     regexp.MustCompile(`(?i)brave[/\s]([\d.]+)`),
		OrderHint: 100,
	},
	{
		Name:      BrowserOpera,
		Keywords:  []string{"opr"},
		Regex:     regexp.MustCompile(`(?i)opr[/\s]([\d.]+)`),
		OrderHint: 110,
	},
	{
		Name:      BrowserOpera, // Alternative Opera browser detection
		Keywords:  []string{"opera"},
		Regex:     regexp.MustCompile(`(?i)opera[/\s]([\d.]+)`),
		OrderHint: 115,
	},
	{
		Name:      BrowserChrome,
		Keywords:  []string{"chrome"},
		Regex:     regexp.MustCompile(`(?i)chrome[/\s]([\d.]+)`),
		OrderHint: 120,
	},
	{
		Name:      BrowserFirefox,
		Keywords:  []string{"firefox"},
		Regex:     regexp.MustCompile(`(?i)firefox[/\s]([\d.]+)`),
		OrderHint: 130,
	},
	{
		Name:      BrowserSafari,
		Keywords:  []string{"safari"},
		Excludes:  []string{"chrome", "firefox"},
		Regex:     regexp.MustCompile(`(?i)version[/\s]([\d.]+)`),
		OrderHint: 140,
	},
	{
		Name:      BrowserIE,
		Keywords:  []string{"msie"},
		Regex:     regexp.MustCompile(`(?i)msie ([\d.]+)`),
		OrderHint: 150,
	},
	{
		Name:      BrowserIE,
		Keywords:  []string{"trident/"},
		OrderHint: 160,
	},
}

func ParseBrowser(lowerUA string) Browser {
	// IE 11 doesn't include 'MSIE' in its UA string, only 'Trident'
	if strings.Contains(lowerUA, "trident/") && !strings.Contains(lowerUA, "msie") {
		return Browser{
			Name:    BrowserIE,
			Version: "11.0",
		}
	}

	for _, pattern := range browserPatterns {
		if matchPattern(lowerUA, pattern) {
			version := extractVersion(lowerUA, pattern.Regex)
			return Browser{
				Name:    pattern.Name,
				Version: version,
			}
		}
	}

	return Browser{
		Name:    BrowserUnknown,
		Version: "",
	}
}
