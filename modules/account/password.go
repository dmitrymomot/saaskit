package account

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/dmitrymomot/saaskit/handler"
	"github.com/dmitrymomot/saaskit/pkg/binder"
	"github.com/dmitrymomot/saaskit/pkg/session"
	"github.com/dmitrymomot/saaskit/svc/auth"
)

// PasswordStorage defines the storage operations needed for password authentication.
type PasswordStorage interface {
	GetUserByEmail(ctx context.Context, email string) (*auth.User, error)
	CreateUser(ctx context.Context, user *auth.User) error
	UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
}

type PasswordService struct {
	cfg          Config
	storage      PasswordStorage
	passwordAuth auth.PasswordAuthenticator
	sessionMgr   *session.Manager
	views        *PasswordServiceViews
	errorHandler handler.ErrorHandler[handler.Context]
}

type PasswordServiceViews struct {
	// Login views
	LoginPage func(PasswordLoginPageParams) templ.Component
	LoginForm func(PasswordLoginFormParams) templ.Component

	// Registration views
	RegisterPage    func(PasswordRegisterPageParams) templ.Component
	RegisterForm    func(PasswordRegisterFormParams) templ.Component
	RegisterSuccess func(PasswordRegisterSuccessParams) templ.Component

	// Password recovery views
	ForgotPasswordPage    func(PasswordForgotPasswordPageParams) templ.Component
	ForgotPasswordForm    func(PasswordForgotPasswordFormParams) templ.Component
	ForgotPasswordSuccess func(PasswordForgotPasswordSuccessParams) templ.Component

	// Password reset views
	ResetPasswordPage    func(PasswordResetPasswordPageParams) templ.Component
	ResetPasswordForm    func(PasswordResetPasswordFormParams) templ.Component
	ResetPasswordSuccess func(PasswordResetPasswordSuccessParams) templ.Component

	// Error views
	Error     func(PasswordErrorPageParams) templ.Component
	ErrorPage func(PasswordErrorPageParams) templ.Component
}

func NewPasswordService(
	cfg Config,
	storage PasswordStorage,
	passwordAuth auth.PasswordAuthenticator,
	sessionMgr *session.Manager,
	views *PasswordServiceViews,
	errorHandler handler.ErrorHandler[handler.Context],
) *PasswordService {
	return &PasswordService{
		cfg:          cfg,
		storage:      storage,
		passwordAuth: passwordAuth,
		sessionMgr:   sessionMgr,
		views:        views,
		errorHandler: errorHandler,
	}
}

func (s *PasswordService) Handle() http.Handler {
	r := chi.NewRouter()

	// Login route - handles both GET and POST
	r.HandleFunc("/login", handler.Wrap(s.login,
		handler.WithBinders[handler.Context, LoginRequest](
			binder.Query(), // Always works
			binder.Form(),  // Skipped for GET, applied for POST
		),
		handler.WithErrorHandler[handler.Context, LoginRequest](s.errorHandler),
	))

	// Registration route - handles both GET and POST
	r.HandleFunc("/register", handler.Wrap(s.register,
		handler.WithBinders[handler.Context, RegisterRequest](
			binder.Query(), // Always works
			binder.Form(),  // Skipped for GET, applied for POST
		),
		handler.WithErrorHandler[handler.Context, RegisterRequest](s.errorHandler),
	))

	// Password recovery route - handles both GET and POST
	r.HandleFunc("/forgot-password", handler.Wrap(s.forgotPassword,
		handler.WithBinders[handler.Context, ForgotPasswordRequest](
			binder.Query(), // Always works
			binder.Form(),  // Skipped for GET, applied for POST
		),
		handler.WithErrorHandler[handler.Context, ForgotPasswordRequest](s.errorHandler),
	))

	// Password reset route - handles both GET and POST
	r.HandleFunc("/reset-password", handler.Wrap(s.resetPassword,
		handler.WithBinders[handler.Context, ResetPasswordRequest](
			binder.Query(), // Always works
			binder.Form(),  // Skipped for GET, applied for POST
		),
		handler.WithErrorHandler[handler.Context, ResetPasswordRequest](s.errorHandler),
	))

	return r
}

// LoginRequest handles both GET (query params) and POST (form data)
type LoginRequest struct {
	Email       string `form:"email" query:"email"`
	Password    string `form:"password"`
	RememberMe  bool   `form:"remember_me"`
	RedirectURL string `form:"redirect_url" query:"redirect"`
}

// PasswordLoginPageParams contains data for rendering the login page.
type PasswordLoginPageParams struct {
	Email       string
	RedirectURL string
}

// PasswordLoginFormParams contains data for rendering the login form.
type PasswordLoginFormParams struct {
	Email       string
	RedirectURL string
}

func (s *PasswordService) login(ctx handler.Context, req LoginRequest) handler.Response {
	// TODO: Implement actual login logic
	// - For POST with credentials: validate, create session, redirect
	// - For GET or validation errors: show form

	formParams := PasswordLoginFormParams{
		Email:       req.Email,
		RedirectURL: req.RedirectURL,
	}
	pageParams := PasswordLoginPageParams{
		Email:       req.Email,
		RedirectURL: req.RedirectURL,
	}

	return handler.TemplPartial(
		s.views.LoginForm(formParams),
		s.views.LoginPage(pageParams),
		handler.WithTarget("#login-form"),
	)
}

// RegisterRequest handles both GET (query params) and POST (form data)
type RegisterRequest struct {
	Name            string `form:"name"`
	Email           string `form:"email"`
	Password        string `form:"password"`
	PasswordConfirm string `form:"password_confirm"`
	RedirectURL     string `form:"redirect_url" query:"redirect"`
}

// PasswordRegisterPageParams contains data for rendering the registration page.
type PasswordRegisterPageParams struct {
	Name        string
	Email       string
	RedirectURL string
}

// PasswordRegisterFormParams contains data for rendering the registration form.
type PasswordRegisterFormParams struct {
	Name        string
	Email       string
	RedirectURL string
}

// PasswordRegisterSuccessParams contains data for rendering the registration success page.
type PasswordRegisterSuccessParams struct {
	Email string
}

func (s *PasswordService) register(ctx handler.Context, req RegisterRequest) handler.Response {
	// TODO: Implement actual registration logic
	// - For POST with valid data: create user, send verification, show success
	// - For GET or validation errors: show form

	formParams := PasswordRegisterFormParams{
		Name:        req.Name,
		Email:       req.Email,
		RedirectURL: req.RedirectURL,
	}
	pageParams := PasswordRegisterPageParams{
		Name:        req.Name,
		Email:       req.Email,
		RedirectURL: req.RedirectURL,
	}

	return handler.TemplPartial(
		s.views.RegisterForm(formParams),
		s.views.RegisterPage(pageParams),
		handler.WithTarget("#register-form"),
	)
}

// ForgotPasswordRequest handles both GET and POST
type ForgotPasswordRequest struct {
	Email string `form:"email" query:"email"`
}

// PasswordForgotPasswordPageParams contains data for rendering the forgot password page.
type PasswordForgotPasswordPageParams struct {
	Email string
}

// PasswordForgotPasswordFormParams contains data for rendering the forgot password form.
type PasswordForgotPasswordFormParams struct {
	Email string
}

// PasswordForgotPasswordSuccessParams contains data for rendering the forgot password success page.
type PasswordForgotPasswordSuccessParams struct {
	Email string
}

func (s *PasswordService) forgotPassword(ctx handler.Context, req ForgotPasswordRequest) handler.Response {
	// TODO: Implement actual forgot password logic
	// - For POST with valid email: send reset link, show success
	// - For GET or errors: show form

	formParams := PasswordForgotPasswordFormParams{
		Email: req.Email,
	}
	pageParams := PasswordForgotPasswordPageParams{
		Email: req.Email,
	}

	return handler.TemplPartial(
		s.views.ForgotPasswordForm(formParams),
		s.views.ForgotPasswordPage(pageParams),
		handler.WithTarget("#forgot-password-form"),
	)
}

// ResetPasswordRequest handles both GET (query params) and POST (form data)
type ResetPasswordRequest struct {
	Token           string `form:"token" query:"token"`
	Password        string `form:"password"`
	PasswordConfirm string `form:"password_confirm"`
}

// PasswordResetPasswordPageParams contains data for rendering the reset password page.
type PasswordResetPasswordPageParams struct {
	Token string
}

// PasswordResetPasswordFormParams contains data for rendering the reset password form.
type PasswordResetPasswordFormParams struct {
	Token string
}

// PasswordResetPasswordSuccessParams contains data for rendering the reset password success page.
type PasswordResetPasswordSuccessParams struct{}

// PasswordErrorPageParams contains data for rendering error pages
type PasswordErrorPageParams struct {
	Error      string
	StatusCode int
	RequestID  string
	RetryURL   string
}

func (s *PasswordService) resetPassword(ctx handler.Context, req ResetPasswordRequest) handler.Response {
	// TODO: Implement actual reset password logic
	// - For POST with valid token and passwords: update password, show success
	// - For GET with token or errors: show form
	// - For invalid token: show error

	formParams := PasswordResetPasswordFormParams{
		Token: req.Token,
	}
	pageParams := PasswordResetPasswordPageParams{
		Token: req.Token,
	}

	return handler.TemplPartial(
		s.views.ResetPasswordForm(formParams),
		s.views.ResetPasswordPage(pageParams),
		handler.WithTarget("#reset-password-form"),
	)
}
