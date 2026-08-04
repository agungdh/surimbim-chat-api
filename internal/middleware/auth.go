package mw

import (
	"context"
	"net/http"
	"strings"

	"surimbim-chat-api/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func RequireAuth(ts *auth.TokenStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := TokenFromRequest(r)
			if token == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			userID, valid := ts.Validate(token)
			if !valid {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TokenFromRequest(r *http.Request) string {
	if h := r.Header.Get("Authorization"); h != "" {
		if t, ok := strings.CutPrefix(h, "Bearer "); ok {
			return t
		}
	}
	if c, err := r.Cookie("token"); err == nil {
		return c.Value
	}
	return ""
}

func UserIDFromCtx(ctx context.Context) int64 {
	id, _ := ctx.Value(UserIDKey).(int64)
	return id
}
