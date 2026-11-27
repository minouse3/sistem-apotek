package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"

	"github.com/minouse3/sistem-apotek/internal/auth"
)

type userKey struct{}

type forbiddenFunc func(w http.ResponseWriter, r *http.Request, user auth.User)

var onForbidden atomic.Pointer[forbiddenFunc]

// SetForbiddenHandler configures an HTML renderer for 403 responses on page requests.
func SetForbiddenHandler(fn func(w http.ResponseWriter, r *http.Request, user auth.User)) {
	if fn == nil {
		onForbidden.Store(nil)
		return
	}
	wrapped := forbiddenFunc(fn)
	onForbidden.Store(&wrapped)
}

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
			if fn := onForbidden.Load(); fn != nil && wantsHTML(r) {
				(*fn)(w, r, user)
				return
			}
			http.Error(w, "Akses ditolak", http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), userKey{}, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func wantsHTML(r *http.Request) bool {
	if strings.HasSuffix(r.URL.Path, ".csv") {
		return false
	}
	accept := r.Header.Get("Accept")
	return accept == "" || strings.Contains(accept, "text/html") || strings.Contains(accept, "*/*")
}
