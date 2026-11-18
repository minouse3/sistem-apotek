package auth

import (
	"errors"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const invalidLoginMessage = "Username atau password yang anda masukkan salah!"

type Service struct {
	pool          *pgxpool.Pool
	loginTemplate *template.Template
	secureCookies bool
}

func New(pool *pgxpool.Pool, loginTemplate *template.Template, secureCookies bool) *Service {
	return &Service{pool: pool, loginTemplate: loginTemplate, secureCookies: secureCookies}
}

type loginData struct {
	Username string
	Error    string
	Invalid  bool
}

func (s *Service) LoginPage(w http.ResponseWriter, r *http.Request) {
	if s.redirectSignedIn(w, r) {
		return
	}
	s.renderLogin(w, http.StatusOK, loginData{})
}

func (s *Service) SubmitLogin(w http.ResponseWriter, r *http.Request) {
	if s.redirectSignedIn(w, r) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	if err := r.ParseForm(); err != nil {
		s.renderLogin(w, http.StatusBadRequest, loginData{Error: "Formulir tidak dapat dibaca. Isi kembali lalu coba lagi."})
		return
	}

	username := strings.TrimSpace(r.PostForm.Get("username"))
	token, expiresAt, err := s.signIn(r.Context(), username, r.PostForm.Get("password"))
	if errors.Is(err, errInvalidCredentials) {
		s.renderLogin(w, http.StatusUnauthorized, loginData{Username: username, Error: invalidLoginMessage, Invalid: true})
		return
	}
	if err != nil {
		log.Printf("login: %v", err)
		s.renderLogin(w, http.StatusInternalServerError, loginData{Username: username, Error: "Tidak dapat masuk saat ini. Coba lagi."})
		return
	}

	s.setSessionCookie(w, r, token, expiresAt)
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Service) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookie)
	if err == nil {
		if _, err := s.pool.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash = $1`, tokenHash(cookie.Value)); err != nil {
			log.Printf("logout: %v", err)
			http.Error(w, "Tidak dapat keluar saat ini. Coba lagi.", http.StatusInternalServerError)
			return
		}
	}

	s.setSessionCookie(w, r, "", time.Unix(1, 0))
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Service) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, expiresAt time.Time) {
	cookie := http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   s.secureCookies || r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	}
	if token == "" {
		cookie.MaxAge = -1
	}
	http.SetCookie(w, &cookie)
}

func (s *Service) redirectSignedIn(w http.ResponseWriter, r *http.Request) bool {
	_, err := s.SessionUser(r)
	switch {
	case err == nil:
		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, "/", http.StatusSeeOther)
	case errors.Is(err, ErrNoSession):
		return false
	default:
		log.Printf("login session: %v", err)
		s.renderLogin(w, http.StatusInternalServerError, loginData{Error: "Tidak dapat memeriksa sesi. Coba lagi."})
	}
	return true
}

func (s *Service) renderLogin(w http.ResponseWriter, status int, data loginData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := s.loginTemplate.Execute(w, data); err != nil {
		log.Printf("render login: %v", err)
	}
}
