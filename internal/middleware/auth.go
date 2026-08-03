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
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			token, ok := strings.CutPrefix(authHeader, "Bearer ")
			if !ok {
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

func UserIDFromCtx(ctx context.Context) int64 {
	id, _ := ctx.Value(UserIDKey).(int64)
	return id
}
