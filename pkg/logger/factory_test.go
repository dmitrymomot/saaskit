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

func TestNew(t *testing.T) {
	t.Run("creates JSON logger", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(logger.WithOutput(buf))
		require.NotNil(t, log)
		log.Info("hello")
		var entry map[string]any
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)
		assert.Equal(t, "INFO", entry["level"])
		assert.Equal(t, "hello", entry["msg"])
	})

	t.Run("text formatter option", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(
			logger.WithOutput(buf),
			logger.WithTextFormatter(),
		)
		log.Info("hello")
		out := buf.String()
		assert.Contains(t, out, "INFO")
		assert.Contains(t, out, "hello")
	})

	t.Run("json formatter option", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(
			logger.WithOutput(buf),
			logger.WithTextFormatter(),
			logger.WithJSONFormatter(),
		)
		log.Info("hello")
		var entry map[string]any
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)
		assert.Equal(t, "hello", entry["msg"])
	})

	t.Run("includes default attributes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New(
			logger.WithOutput(buf),
			logger.WithAttr(slog.String("svc", "test")),
		)
		log.Info("msg")
		var entry map[string]any
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)
		assert.Equal(t, "test", entry["svc"])
	})

	t.Run("extracts from context", func(t *testing.T) {
		buf := &bytes.Buffer{}
		type key string
		ctxKey := key("id")
		log := logger.New(
			logger.WithOutput(buf),
			logger.WithContextExtractors(func(ctx context.Context) (slog.Attr, bool) {
				if v := ctx.Value(ctxKey); v != nil {
					return slog.String("id", v.(string)), true
				}
				return slog.Attr{}, false
			}),
		)
		ctx := context.WithValue(context.Background(), ctxKey, "42")
		log.InfoContext(ctx, "context msg")
		var entry map[string]any
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)
		assert.Equal(t, "42", entry["id"])
	})
}

func TestSetAsDefault(t *testing.T) {
	buf := &bytes.Buffer{}
	log := logger.New(logger.WithOutput(buf))
	logger.SetAsDefault(log)
	slog.Info("default")
	var entry map[string]any
	err := json.Unmarshal(buf.Bytes(), &entry)
	require.NoError(t, err)
	assert.Equal(t, "default", entry["msg"])
}

func TestWithFormatPanics(t *testing.T) {
	assert.Panics(t, func() {
		logger.New(logger.WithFormat(logger.Format("xml")))
	})
}
