package templates

import (
	"context"
	"strings"

	"github.com/a-h/templ"
)

// Render takes a templ.Component and renders it to a string.
// It uses a strings.Builder to efficiently build the string from the component.
//
// Parameters:
//   - ctx: The context for rendering the component
//   - tpl: The templ component to render
//
// Returns:
//   - string: The rendered HTML as a string
//   - error: Any error encountered during rendering
func Render(ctx context.Context, tpl templ.Component) (string, error) {
	var sb strings.Builder
	err := tpl.Render(ctx, &sb)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}
