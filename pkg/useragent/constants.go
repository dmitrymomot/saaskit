package useragent

// Device types represent the category of device that made the request
const (
	// DeviceTypeBot identifies automated crawlers, bots, and spiders
	DeviceTypeBot = "bot"

	// DeviceTypeMobile identifies smartphones and feature phones
	DeviceTypeMobile = "mobile"

	// DeviceTypeTablet identifies tablet devices (iPad, Android tablets, etc.)
	DeviceTypeTablet = "tablet"

	// DeviceTypeDesktop identifies desktop computers and laptops
	DeviceTypeDesktop = "desktop"

	// DeviceTypeTV identifies smart TVs and streaming devices
	DeviceTypeTV = "tv"

	// DeviceTypeConsole identifies gaming consoles
	DeviceTypeConsole = "console"

	// DeviceTypeUnknown is used when the device type cannot be determined
	DeviceTypeUnknown = "unknown"
)

// Mobile device model identifiers
const (
	// MobileDeviceIPhone identifies Apple iPhone devices
	MobileDeviceIPhone = "iphone"

	// MobileDeviceAndroid identifies generic Android mobile devices
	MobileDeviceAndroid = "android"

	// MobileDeviceSamsung identifies Samsung mobile devices
	MobileDeviceSamsung = "samsung"

	// MobileDeviceHuawei identifies Huawei mobile devices
	MobileDeviceHuawei = "huawei"

	// MobileDeviceXiaomi identifies Xiaomi mobile devices
	MobileDeviceXiaomi = "xiaomi"

	// MobileDeviceOppo identifies Oppo mobile devices
	MobileDeviceOppo = "oppo"

	// MobileDeviceVivo identifies Vivo mobile devices
	MobileDeviceVivo = "vivo"

	// MobileDeviceUnknown is used when the mobile device model cannot be determined
	MobileDeviceUnknown = "unknown"
)

// Tablet device model identifiers
const (
	// TabletDeviceIPad identifies Apple iPad tablets
	TabletDeviceIPad = "ipad"

	// TabletDeviceAndroid identifies generic Android tablets
	TabletDeviceAndroid = "android"

	// TabletDeviceSamsung identifies Samsung tablets
	TabletDeviceSamsung = "samsung"

	// TabletDeviceHuawei identifies Huawei tablets
	TabletDeviceHuawei = "huawei"

	// TabletDeviceKindleFire identifies Amazon Kindle Fire tablets
	TabletDeviceKindleFire = "kindle"

	// TabletDeviceSurface identifies Microsoft Surface tablets
	TabletDeviceSurface = "surface"

	// TabletDeviceUnknown is used when the tablet model cannot be determined
	TabletDeviceUnknown = "unknown"
)

// Browser name identifiers
const (
	// BrowserChrome identifies Google Chrome browser
	BrowserChrome = "chrome"

	// BrowserFirefox identifies Mozilla Firefox browser
	BrowserFirefox = "firefox"

	// BrowserSafari identifies Apple Safari browser
	BrowserSafari = "safari"

	// BrowserEdge identifies Microsoft Edge browser
	BrowserEdge = "edge"

	// BrowserOpera identifies Opera browser
	BrowserOpera = "opera"

	// BrowserIE identifies Internet Explorer browser
	BrowserIE = "ie"

	// BrowserSamsung identifies Samsung Internet browser
	BrowserSamsung = "samsung"

	// BrowserUC identifies UC Browser
	BrowserUC = "uc"

	// BrowserQQ identifies QQ Browser
	BrowserQQ = "qq"

	// BrowserHuawei identifies Huawei Browser
	BrowserHuawei = "huawei"

	// BrowserVivo identifies Vivo Browser
	BrowserVivo = "vivo"

	// BrowserMIUI identifies Xiaomi MIUI Browser
	BrowserMIUI = "miui"

	// BrowserBrave identifies Brave Browser
	BrowserBrave = "brave"

	// BrowserVivaldi identifies Vivaldi Browser
	BrowserVivaldi = "vivaldi"

	// BrowserYandex identifies Yandex Browser
	BrowserYandex = "yandex"

	// BrowserUnknown is used when the browser cannot be determined
	BrowserUnknown = "unknown"
)

// Operating system identifiers
const (
	// OSWindows identifies Microsoft Windows operating system
	OSWindows = "windows"

	// OSWindowsPhone identifies Microsoft Windows Phone operating system
	OSWindowsPhone = "windows phone"

	// OSMacOS identifies Apple macOS operating system
	OSMacOS = "macos"

	// OSiOS identifies Apple iOS mobile operating system
	OSiOS = "ios"

	// OSAndroid identifies Google Android operating system
	OSAndroid = "android"

	// OSLinux identifies Linux-based operating systems
	OSLinux = "linux"

	// OSChromeOS identifies Google Chrome OS operating system
	OSChromeOS = "chromeos"

	// OSHarmonyOS identifies Huawei HarmonyOS operating system
	OSHarmonyOS = "harmonyos"

	// OSFireOS identifies Amazon Fire OS operating system
	OSFireOS = "fireos"

	// OSUnknown is used when the operating system cannot be determined
	OSUnknown = "unknown"
)
