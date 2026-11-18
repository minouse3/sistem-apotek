package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"

	"github.com/minouse3/sistem-apotek/internal/auth"
)

type userKey struct{}

func CurrentUser(ctx context.Context) (auth.User, bool) {
	user, ok := ctx.Value(userKey{}).(auth.User)
	return user, ok
}

// Require accepts any supported role when allowed is empty.
func Require(accounts *auth.Service, next http.Handler, allowed ...auth.Role) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		user, err := accounts.SessionUser(r)
		if errors.Is(err, auth.ErrNoSession) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if err != nil {
			log.Printf("session: %v", err)
			http.Error(w, "Tidak dapat memeriksa sesi. Coba lagi.", http.StatusInternalServerError)
			return
		}
		if len(allowed) > 0 && !slices.Contains(allowed, user.Role) {
			http.Error(w, "Akses ditolak", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), userKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
