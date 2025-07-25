package email

type Config struct {
	PostmarkServerToken  string `env:"POSTMARK_SERVER_TOKEN"`  // Postmark API server token
	PostmarkAccountToken string `env:"POSTMARK_ACCOUNT_TOKEN"` // Postmark API account token
	SenderEmail          string `env:"SENDER_EMAIL,required"`  // Email address of the sender.
	SupportEmail         string `env:"SUPPORT_EMAIL,required"` // Email address for customer support.
}
