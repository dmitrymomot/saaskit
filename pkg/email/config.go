package email

// Config holds email service configuration.
// PostmarkServerToken and PostmarkAccountToken are optional to support
// development environments where email sending is disabled.
// SenderEmail and SupportEmail are required as they establish the sender identity
// and reply-to behavior for all outbound emails.
type Config struct {
	PostmarkServerToken  string `env:"POSTMARK_SERVER_TOKEN"`
	PostmarkAccountToken string `env:"POSTMARK_ACCOUNT_TOKEN"`
	SenderEmail          string `env:"SENDER_EMAIL,required"`
	SupportEmail         string `env:"SUPPORT_EMAIL,required"`
}
