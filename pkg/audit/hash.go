package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type Hasher interface {
	Hash(event Event) string
}

type sha256Hasher struct{}

func NewSHA256Hasher() Hasher {
	return &sha256Hasher{}
}

func (h *sha256Hasher) Hash(event Event) string {
	data := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|%s|%d|%s",
		event.TenantID,
		event.UserID,
		event.SessionID,
		event.Action,
		event.Resource,
		event.ResourceID,
		event.Result,
		event.CreatedAt.Unix(),
		event.Error,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
