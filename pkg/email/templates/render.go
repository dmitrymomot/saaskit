package templates

import (
	"context"
	"strings"

	"github.com/a-h/templ"
)

// Render converts a templ.Component to HTML string for email bodies.
// Uses strings.Builder for zero-allocation string construction during
// template rendering, which is critical for email throughput performance.
func Render(ctx context.Context, tpl templ.Component) (string, error) {
	var sb strings.Builder
	err := tpl.Render(ctx, &sb)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}
