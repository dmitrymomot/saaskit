package audit

import "context"

type TenantIDExtractor func(context.Context) (string, bool)

type UserIDExtractor func(context.Context) (string, bool)

type SessionIDExtractor func(context.Context) (string, bool)
