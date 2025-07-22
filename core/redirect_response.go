package core

import (
	"net/http"
	"net/url"

	"github.com/starfederation/datastar-go/datastar"
)

// redirectResponse handles redirects for both DataStar and regular requests
type redirectResponse struct {
	url  string
	code int
}

// Render performs the redirect, handling both DataStar and regular requests
func (r redirectResponse) Render(w http.ResponseWriter, req *http.Request) error {
	if IsDataStar(req) {
		sse := datastar.NewSSE(w, req)
		return sse.Redirect(r.url)
	}
	http.Redirect(w, req, r.url, r.code)
	return nil
}

// Redirect creates a redirect response with status 303 (See Other).
// For DataStar requests, it uses Server-Sent Events to trigger a client-side redirect.
// For regular requests, it performs a standard HTTP redirect.
//
// Example:
//
//	handler := saaskit.HandlerFunc[saaskit.Context, CreateUserRequest](
//		func(ctx saaskit.Context, req CreateUserRequest) saaskit.Response {
//			user := createUser(req)
//			return saaskit.Redirect("/users/" + user.ID)
//		},
//	)
func Redirect(url string) Response {
	return redirectResponse{
		url:  url,
		code: http.StatusSeeOther,
	}
}

// RedirectWithCode creates a redirect response with a specific status code.
// Valid codes are 301 (Moved Permanently), 302 (Found), 303 (See Other),
// 307 (Temporary Redirect), and 308 (Permanent Redirect).
//
// Example:
//
//	// Permanent redirect
//	return saaskit.RedirectWithCode("/new-location", http.StatusMovedPermanently)
func RedirectWithCode(url string, code int) Response {
	return redirectResponse{
		url:  url,
		code: code,
	}
}

// redirectBackResponse handles redirect to referrer
type redirectBackResponse struct {
	fallback string
	code     int
}

// Render redirects back to the referrer or fallback URL
func (r redirectBackResponse) Render(w http.ResponseWriter, req *http.Request) error {
	referer := req.Header.Get("Referer")
	targetURL := r.fallback

	if referer != "" && isValidRedirectURL(referer, req) {
		targetURL = referer
	}

	if IsDataStar(req) {
		sse := datastar.NewSSE(w, req)
		return sse.Redirect(targetURL)
	}

	http.Redirect(w, req, targetURL, r.code)
	return nil
}

// RedirectBack creates a redirect back to the referrer or fallback URL.
// It validates that the referrer is from the same host for security.
// Uses status 303 (See Other) for the redirect.
//
// Example:
//
//	handler := saaskit.HandlerFunc[saaskit.Context, DeleteRequest](
//		func(ctx saaskit.Context, req DeleteRequest) saaskit.Response {
//			deleteItem(req.ID)
//			// Go back to where the user came from, or home if no referrer
//			return saaskit.RedirectBack("/")
//		},
//	)
func RedirectBack(fallback string) Response {
	return redirectBackResponse{
		fallback: fallback,
		code:     http.StatusSeeOther,
	}
}

// RedirectBackWithCode creates a redirect back response with a specific status code
func RedirectBackWithCode(fallback string, code int) Response {
	return redirectBackResponse{
		fallback: fallback,
		code:     code,
	}
}

// isValidRedirectURL checks if a URL is safe to redirect to
func isValidRedirectURL(urlStr string, r *http.Request) bool {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	// Only allow same-host redirects (empty host means relative URL)
	return parsed.Host == "" || parsed.Host == r.Host
}
