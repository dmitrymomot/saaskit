package logger_test

import (
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/saaskit/pkg/logger"
)

func TestGroup(t *testing.T) {
	attr := logger.Group("req", slog.String("id", "1"), slog.Int("n", 2))
	require.Equal(t, "req", attr.Key)
	require.Equal(t, slog.KindGroup, attr.Value.Kind())
	g := attr.Value.Group()
	require.Len(t, g, 2)
	assert.Equal(t, "id", g[0].Key)
	assert.Equal(t, "n", g[1].Key)
}

func TestErrors(t *testing.T) {
	err1 := errors.New("first")
	err2 := errors.New("second")

	attr := logger.Errors(err1, nil, err2)
	require.Equal(t, "errors", attr.Key)
	require.Equal(t, slog.KindGroup, attr.Value.Kind())
	g := attr.Value.Group()
	require.Len(t, g, 2)
	assert.Equal(t, err1, g[0].Value.Any())
	assert.Equal(t, err2, g[1].Value.Any())

	empty := logger.Errors(nil)
	assert.True(t, empty.Equal(slog.Attr{}))
}

func TestError(t *testing.T) {
	err := errors.New("boom")
	attr := logger.Error(err)
	require.Equal(t, "error", attr.Key)
	assert.Equal(t, err, attr.Value.Any())

	empty := logger.Error(nil)
	assert.True(t, empty.Equal(slog.Attr{}))
}

func TestUserID(t *testing.T) {
	attr := logger.UserID("123")
	require.Equal(t, "user_id", attr.Key)
	assert.Equal(t, "123", attr.Value.Any())
}

func TestWorkspaceID(t *testing.T) {
	attr := logger.WorkspaceID("ws1")
	require.Equal(t, "workspace_id", attr.Key)
	assert.Equal(t, "ws1", attr.Value.Any())
}

func TestRole(t *testing.T) {
	attr := logger.Role("admin")
	require.Equal(t, "role", attr.Key)
	assert.Equal(t, "admin", attr.Value.Any())
}

func TestRequestID(t *testing.T) {
	attr := logger.RequestID("abc")
	require.Equal(t, "request_id", attr.Key)
	assert.Equal(t, "abc", attr.Value.Any())
}
