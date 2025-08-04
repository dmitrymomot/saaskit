package audit

import "context"

// TenantIDExtractor extracts tenant ID from request context.
// Returns (tenantID, found) where found indicates if extraction succeeded.
type TenantIDExtractor func(context.Context) (string, bool)

// UserIDExtractor extracts user ID from request context.
// Returns (userID, found) where found indicates if extraction succeeded.
type UserIDExtractor func(context.Context) (string, bool)

// SessionIDExtractor extracts session ID from request context.
// Returns (sessionID, found) where found indicates if extraction succeeded.
type SessionIDExtractor func(context.Context) (string, bool)
