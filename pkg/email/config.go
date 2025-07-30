package email

type Config struct {
	PostmarkServerToken  string `env:"POSTMARK_SERVER_TOKEN"`
	PostmarkAccountToken string `env:"POSTMARK_ACCOUNT_TOKEN"`
	SenderEmail          string `env:"SENDER_EMAIL,required"`
	SupportEmail         string `env:"SUPPORT_EMAIL,required"`
}
