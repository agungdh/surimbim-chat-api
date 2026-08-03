package mw

import (
	"net/http"
	"strconv"
)

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Header.Get("user-id")
		if uid == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		_, err := strconv.ParseInt(uid, 10, 64)
		if err != nil {
			http.Error(w, `{"error":"invalid user-id"}`, http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r.WithContext(r.Context()))
	})
}
