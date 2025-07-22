package fingerprint

import (
	"context"
)

type fingerprintContextKey struct{}

func SetFingerprintToContext(ctx context.Context, fingerprint string) context.Context {
	return context.WithValue(ctx, fingerprintContextKey{}, fingerprint)
}

func GetFingerprintFromContext(ctx context.Context) string {
	fingerprint, _ := ctx.Value(fingerprintContextKey{}).(string)
	return fingerprint
}
