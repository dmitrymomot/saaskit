package totp

type Config struct {
	EncryptionKey string `env:"TOTP_ENCRYPTION_KEY,required"`
}
