package handler

import (
	"encoding/json"
	"net/http"

	"github.com/uptrace/bun"
	"surimbim-chat-api/internal/auth"
	"surimbim-chat-api/internal/config"
	"surimbim-chat-api/internal/model"

	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func Login(db *bun.DB, ts *auth.TokenStore, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Username == "" || req.Password == "" {
			respondError(w, http.StatusBadRequest, "username and password required")
			return
		}

		var user model.User
		err := db.NewSelect().Model(&user).Where("username = ?", req.Username).Scan(r.Context())
		if err != nil {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}

		token := ts.Generate(user.ID)

		cookie := &http.Cookie{
			Name:     "token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   int(cfg.SessionTTL.Seconds()),
		}
		if cfg.ENV == "prod" {
			cookie.Secure = true
		}
		http.SetCookie(w, cookie)

		respondJSON(w, http.StatusOK, loginResponse{Token: token})
	}
}

func Logout(ts *auth.TokenStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("token"); err == nil {
			ts.Remove(c.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		w.WriteHeader(http.StatusNoContent)
	}
}
