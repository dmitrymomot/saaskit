package token_test

import (
	"testing"

	"github.com/dmitrymomot/saaskit/pkg/token"
)

func BenchmarkGenerateToken(b *testing.B) {
	payload := testPayload{ID: 123, Name: "benchmark"}
	secret := "benchmark-secret"

	for b.Loop() {
		_, err := token.GenerateToken(payload, secret)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseToken(b *testing.B) {
	payload := testPayload{ID: 123, Name: "benchmark"}
	secret := "benchmark-secret"

	// Pre-generate token
	tokenStr, err := token.GenerateToken(payload, secret)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		_, err := token.ParseToken[testPayload](tokenStr, secret)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateToken_LargePayload(b *testing.B) {
	type largePayload struct {
		ID   int    `json:"id"`
		Data string `json:"data"`
	}

	// Create a ~1KB payload
	largeData := make([]byte, 1024)
	for i := range largeData {
		largeData[i] = 'a'
	}

	payload := largePayload{ID: 123, Data: string(largeData)}
	secret := "benchmark-secret"

	for b.Loop() {
		_, err := token.GenerateToken(payload, secret)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseToken_LargePayload(b *testing.B) {
	type largePayload struct {
		ID   int    `json:"id"`
		Data string `json:"data"`
	}

	// Create a ~1KB payload
	largeData := make([]byte, 1024)
	for i := range largeData {
		largeData[i] = 'a'
	}

	payload := largePayload{ID: 123, Data: string(largeData)}
	secret := "benchmark-secret"

	// Pre-generate token
	tokenStr, err := token.GenerateToken(payload, secret)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for b.Loop() {
		var parsed largePayload
		parsed, err = token.ParseToken[largePayload](tokenStr, secret)
		if err != nil {
			b.Fatal(err)
		}
		_ = parsed
	}
}
