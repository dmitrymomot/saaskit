package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"log/slog"

	"github.com/dmitrymomot/saaskit/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDevelopment(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(
		logger.WithDevelopment("svc"),
		logger.WithOutput(buf),
	)
	require.NotNil(t, log)
	log.Debug("msg")
	output := buf.String()
	assert.Contains(t, output, "DEBUG")
	assert.Contains(t, output, "service=svc")
}

func TestWithProduction(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(
		logger.WithProduction("svc"),
		logger.WithOutput(buf),
	)
	require.NotNil(t, log)
	log.Info("msg")
	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, "svc", entry["service"])
}

func TestEnvironmentOptions(t *testing.T) {
	dev := logger.New(logger.WithDevelopment("svc"))
	prod := logger.New(logger.WithProduction("svc"))
	require.NotNil(t, dev)
	require.NotNil(t, prod)
}

func TestWithExtractors(t *testing.T) {
	buf := &bytes.Buffer{}
	type key string
	k := key("id")
	extractor := func(ctx context.Context) (slog.Attr, bool) {
		if v := ctx.Value(k); v != nil {
			return slog.String("id", v.(string)), true
		}
		return slog.Attr{}, false
	}
	log := logger.New(
		logger.WithProduction("svc"),
		logger.WithOutput(buf),
		logger.WithContextExtractors(extractor),
	)
	ctx := context.WithValue(context.Background(), k, "123")
	log.InfoContext(ctx, "msg")
	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, "123", entry["id"])
}
