package server

import (
	"context"

	"github.com/keskad/bigfred-os/apps/bigfred-os-ui/internal/auth"
)

type ctxKey int

const sessionKey ctxKey = 1

func withSession(ctx context.Context, sess auth.Session) context.Context {
	return context.WithValue(ctx, sessionKey, sess)
}

func sessionFromContext(ctx context.Context) (auth.Session, bool) {
	sess, ok := ctx.Value(sessionKey).(auth.Session)
	return sess, ok
}
