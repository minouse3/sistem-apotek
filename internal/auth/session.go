package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookie = "apotek_session"
	sessionTTL    = 8 * time.Hour
	dummyHash     = "$2a$10$KL1PbMRsYCcKB/GKIvZKiOoQyzqQb5s9KfCO87SMKoC34C4.13fqa"
)

type Role string

const (
	RolePemilik Role = "pemilik"
	RoleStaf    Role = "staf"
	RoleKasir   Role = "kasir"
)

type User struct {
	ID   int
	Name string
	Role Role
}

var (
	ErrNoSession          = errors.New("no active session")
	errInvalidCredentials = errors.New("invalid credentials")
)

func (role Role) valid() bool {
	return role == RolePemilik || role == RoleStaf || role == RoleKasir
}

func tokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func (s *Service) signIn(ctx context.Context, username, password string) (string, time.Time, error) {
	if username == "" || utf8.RuneCountInString(username) > 50 || strings.ContainsRune(username, 0) || len(password) == 0 || len(password) > 72 {
		return "", time.Time{}, errInvalidCredentials
	}

	var id int
	var role Role
	var hash string
	err := s.pool.QueryRow(ctx, `
		SELECT u.id_pengguna, r.nama_role, u.password_hash
		FROM pengguna u JOIN role r USING (id_role)
		WHERE u.username = $1 AND u.status_aktif`, username).Scan(&id, &role, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		// Pay the bcrypt cost even when no active account matched.
		_ = bcrypt.CompareHashAndPassword([]byte(dummyHash), []byte(password))
		return "", time.Time{}, errInvalidCredentials
	}
	if err != nil {
		return "", time.Time{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || !role.valid() {
		return "", time.Time{}, errInvalidCredentials
	}

	token := rand.Text()
	expiresAt := time.Now().Add(sessionTTL).UTC()
	// Recheck the account after bcrypt in case it changed while the password was verified.
	err = s.pool.QueryRow(ctx, `
		WITH signed_in AS (
			UPDATE pengguna SET last_login = now()
			WHERE id_pengguna = $1 AND status_aktif AND password_hash = $2
			RETURNING id_pengguna
		)
		INSERT INTO sessions (id_pengguna, token_hash, expires_at)
		SELECT id_pengguna, $3, $4 FROM signed_in
		RETURNING expires_at`, id, hash, tokenHash(token), expiresAt).Scan(&expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", time.Time{}, errInvalidCredentials
	}
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *Service) SessionUser(r *http.Request) (User, error) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || cookie.Value == "" {
		return User{}, ErrNoSession
	}

	var user User
	err = s.pool.QueryRow(r.Context(), `
		SELECT u.id_pengguna, u.nama_pengguna, r.nama_role
		FROM sessions s
		JOIN pengguna u USING (id_pengguna)
		JOIN role r USING (id_role)
		WHERE s.token_hash = $1 AND s.expires_at > now() AND u.status_aktif`,
		tokenHash(cookie.Value)).Scan(&user.ID, &user.Name, &user.Role)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !user.Role.valid()) {
		return User{}, ErrNoSession
	}
	return user, err
}
