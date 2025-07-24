package queue

import (
	"fmt"
	"strings"
)

func qualifiedStructName(v any) string {
	s := fmt.Sprintf("%T", v)
	s = strings.TrimLeft(s, "*")

	return s
}
