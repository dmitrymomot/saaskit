package account

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Mountable interface {
	Handle() http.Handler
}

// RouterOptions configures which services to mount in the account module.
// Each service is optional and will only be mounted if provided.
type RouterOptions struct {
	// Authentication services
	Password    Mountable
	MagicLink   Mountable
	GoogleOAuth Mountable
	GithubOAuth Mountable
}

// Router creates a new account module router with configurable services.
//
// Example:
//
//	passwordSvc := account.NewPasswordService(cfg, storage, auth, sessionMgr, views)
//	oauthSvc := oauth.NewGoogleProvider(googleCfg)
//
//	r := chi.NewRouter()
//	r.Mount("/account", account.Router(account.RouterOptions{
//	    Password: passwordSvc,
//	    OAuth:    oauthSvc,
//	}))
func Router(opts RouterOptions) chi.Router {
	r := chi.NewRouter()

	r.Route("/auth", func(auth chi.Router) {
		if opts.Password != nil {
			auth.Mount("/password", opts.Password.Handle())
		}
		if opts.MagicLink != nil {
			auth.Mount("/magic", opts.MagicLink.Handle())
		}
		if opts.GoogleOAuth != nil {
			auth.Mount("/google", opts.GoogleOAuth.Handle())
		}
		if opts.GithubOAuth != nil {
			auth.Mount("/github", opts.GithubOAuth.Handle())
		}
	})

	return r
}
