package obat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
)

type karyawan struct {
	ID        int
	Nama      string
	Username  string
	Email     string
	Role      auth.Role
	Aktif     bool
	LastLogin *time.Time
}

type daftarKaryawanData struct {
	Halaman
	Daftar []karyawan
}

func (h *Handler) daftarKaryawan(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT u.id_pengguna, u.nama_pengguna, u.username, COALESCE(u.email, ''), r.nama_role,
		       u.status_aktif, u.last_login
		FROM pengguna u JOIN role r USING (id_role)
		ORDER BY u.status_aktif DESC, u.nama_pengguna`)
	if err != nil {
		gagal(w, "daftar karyawan", err)
		return
	}
	daftar, err := pgx.CollectRows(rows, pgx.RowToStructByPos[karyawan])
	if err != nil {
		gagal(w, "daftar karyawan", err)
		return
	}
	h.render(w, http.StatusOK, "karyawan/daftar", daftarKaryawanData{
		Halaman: h.halaman(r, "Akun karyawan", "karyawan"),
		Daftar:  daftar,
	})
}

type role struct {
	ID   int
	Nama auth.Role
}

type isianKaryawan struct {
	Nama     string
	Username string
	Email    string
	IDRole   int
}

type formKaryawanData struct {
	Halaman
	Baru        bool
	Aksi        string
	Isian       isianKaryawan
	Roles       []role
	AkunSendiri bool
	Errors      map[string]string
}

func (h *Handler) tampilFormKaryawan(w http.ResponseWriter, r *http.Request, status int, data formKaryawanData) {
	rows, err := h.pool.Query(r.Context(), `SELECT id_role, nama_role FROM role ORDER BY id_role`)
	if err != nil {
		gagal(w, "daftar role", err)
		return
	}
	roles, err := pgx.CollectRows(rows, pgx.RowToStructByPos[role])
	if err != nil {
		gagal(w, "daftar role", err)
		return
	}
	title := "Ubah akun karyawan"
	if data.Baru {
		title = "Tambah akun karyawan"
	}
	data.Halaman = h.halaman(r, title, "karyawan")
	data.Halaman.Pesan = ""
	data.Roles = roles
	h.render(w, status, "karyawan/form", data)
}

func (h *Handler) formKaryawanBaru(w http.ResponseWriter, r *http.Request) {
	h.tampilFormKaryawan(w, r, http.StatusOK, formKaryawanData{Baru: true, Aksi: "/karyawan"})
}

func usernameValid(s string) bool {
	if len(s) < 3 || len(s) > 50 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func bacaIsianKaryawan(r *http.Request) (isianKaryawan, map[string]string) {
	isian := isianKaryawan{
		Nama:     strings.Join(strings.Fields(r.PostForm.Get("nama_pengguna")), " "),
		Username: strings.ToLower(strings.TrimSpace(r.PostForm.Get("username"))),
		Email:    strings.TrimSpace(r.PostForm.Get("email")),
	}
	isian.IDRole, _ = strconv.Atoi(r.PostForm.Get("id_role"))

	errs := map[string]string{}
	switch {
	case isian.Nama == "":
		errs["nama_pengguna"] = "Isi nama lengkap."
	case utf8.RuneCountInString(isian.Nama) > 100:
		errs["nama_pengguna"] = "Nama maksimal 100 karakter."
	}
	if !usernameValid(isian.Username) {
		errs["username"] = "Gunakan 3-50 karakter: huruf kecil, angka, titik, garis bawah, atau tanda hubung."
	}
	if isian.Email != "" {
		addr, err := mail.ParseAddress(isian.Email)
		if err != nil || addr.Address != isian.Email || len(isian.Email) > 150 {
			errs["email"] = "Isi alamat email yang valid, misalnya nama@contoh.com, atau kosongkan."
		}
	}
	if isian.IDRole <= 0 {
		errs["id_role"] = "Pilih peran."
	}
	return isian, errs
}

// bacaPassword validates a new password; bcrypt only reads the first 72 bytes.
func bacaPassword(r *http.Request, wajib bool, errs map[string]string) string {
	password := r.PostForm.Get("password")
	konfirmasi := r.PostForm.Get("konfirmasi_password")
	if password == "" && !wajib {
		return ""
	}
	switch {
	case password == "":
		errs["password"] = "Isi password."
	case utf8.RuneCountInString(password) < 8 || len(password) > 72:
		errs["password"] = "Password harus 8-72 karakter."
	case password != konfirmasi:
		errs["konfirmasi_password"] = "Konfirmasi password tidak sama."
	}
	return password
}

func errorUnikKaryawan(err error, errs map[string]string) bool {
	switch {
	case pelanggaranUnik(err, "pengguna_username_key"):
		errs["username"] = "Username ini sudah dipakai. Pilih username lain."
	case pelanggaranUnik(err, "pengguna_email_key"):
		errs["email"] = "Email ini sudah dipakai akun lain."
	case fkGagal(err):
		errs["id_role"] = "Peran tidak ditemukan. Pilih ulang."
	default:
		return false
	}
	return true
}

func (h *Handler) simpanKaryawanBaru(w http.ResponseWriter, r *http.Request) {
	if !bacaForm(w, r) {
		return
	}
	isian, errs := bacaIsianKaryawan(r)
	password := bacaPassword(r, true, errs)
	data := formKaryawanData{Baru: true, Aksi: "/karyawan", Isian: isian, Errors: errs}
	if len(errs) > 0 {
		h.tampilFormKaryawan(w, r, http.StatusUnprocessableEntity, data)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		gagal(w, "hash password", err)
		return
	}
	_, err = h.pool.Exec(r.Context(), `
		INSERT INTO pengguna (id_role, nama_pengguna, username, email, password_hash)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5)`,
		isian.IDRole, isian.Nama, isian.Username, isian.Email, string(hash))
	if errorUnikKaryawan(err, errs) {
		h.tampilFormKaryawan(w, r, http.StatusConflict, data)
		return
	}
	if err != nil {
		gagal(w, "tambah karyawan", err)
		return
	}
	http.Redirect(w, r, "/karyawan?pesan=karyawan-baru", http.StatusSeeOther)
}

func (h *Handler) formUbahKaryawan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	var isian isianKaryawan
	err := h.pool.QueryRow(r.Context(), `
		SELECT nama_pengguna, username, COALESCE(email, ''), id_role
		FROM pengguna WHERE id_pengguna = $1`, id).
		Scan(&isian.Nama, &isian.Username, &isian.Email, &isian.IDRole)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		gagal(w, "ubah karyawan", err)
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	h.tampilFormKaryawan(w, r, http.StatusOK, formKaryawanData{
		Aksi:        fmt.Sprintf("/karyawan/%d/ubah", id),
		Isian:       isian,
		AkunSendiri: user.ID == id,
	})
}

func (h *Handler) simpanUbahKaryawan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !bacaForm(w, r) {
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	sendiri := user.ID == id
	isian, errs := bacaIsianKaryawan(r)
	password := bacaPassword(r, false, errs)
	data := formKaryawanData{Aksi: fmt.Sprintf("/karyawan/%d/ubah", id), Isian: isian, AkunSendiri: sendiri, Errors: errs}

	if sendiri && len(errs) == 0 {
		// Owners cannot demote themselves and lock everyone out of account management.
		var rolePilihan auth.Role
		err := h.pool.QueryRow(r.Context(), `SELECT nama_role FROM role WHERE id_role = $1`, isian.IDRole).Scan(&rolePilihan)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			gagal(w, "role", err)
			return
		}
		if rolePilihan != auth.RolePemilik {
			errs["id_role"] = "Anda tidak bisa mengganti peran akun sendiri."
		}
	}
	if len(errs) > 0 {
		h.tampilFormKaryawan(w, r, http.StatusUnprocessableEntity, data)
		return
	}

	var hash string
	if password != "" {
		b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			gagal(w, "hash password", err)
			return
		}
		hash = string(b)
	}

	err := pgx.BeginFunc(r.Context(), h.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(), `
			UPDATE pengguna
			SET nama_pengguna = $2, username = $3, email = NULLIF($4, ''), id_role = $5,
			    password_hash = COALESCE(NULLIF($6, ''), password_hash), updated_at = now()
			WHERE id_pengguna = $1`,
			id, isian.Nama, isian.Username, isian.Email, isian.IDRole, hash)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		if hash != "" && !sendiri {
			return akhiriSesi(r.Context(), tx, id)
		}
		return nil
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		http.NotFound(w, r)
	case errorUnikKaryawan(err, errs):
		h.tampilFormKaryawan(w, r, http.StatusConflict, data)
	case err != nil:
		gagal(w, "ubah karyawan", err)
	default:
		http.Redirect(w, r, "/karyawan?pesan=karyawan-ubah", http.StatusSeeOther)
	}
}

func akhiriSesi(ctx context.Context, tx pgx.Tx, id int) error {
	_, err := tx.Exec(ctx, `DELETE FROM sessions WHERE id_pengguna = $1`, id)
	return err
}

func (h *Handler) ubahStatusKaryawan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !bacaForm(w, r) {
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	if user.ID == id {
		http.Error(w, "Anda tidak bisa menonaktifkan akun sendiri.", http.StatusForbidden)
		return
	}
	aktif := r.PostForm.Get("aktif") == "true"

	err := pgx.BeginFunc(r.Context(), h.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(), `
			UPDATE pengguna SET status_aktif = $2, updated_at = now()
			WHERE id_pengguna = $1`, id, aktif)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		if !aktif {
			return akhiriSesi(r.Context(), tx, id)
		}
		return nil
	})
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		gagal(w, "status karyawan", err)
		return
	}
	pesan := "akun-nonaktif"
	if aktif {
		pesan = "akun-aktif"
	}
	http.Redirect(w, r, "/karyawan?pesan="+pesan, http.StatusSeeOther)
}

type akunSayaData struct {
	Halaman
	Akun   karyawan
	Errors map[string]string
}

func (h *Handler) tampilAkunSaya(w http.ResponseWriter, r *http.Request, status int, errs map[string]string) {
	user, _ := middleware.CurrentUser(r.Context())
	var akun karyawan
	err := h.pool.QueryRow(r.Context(), `
		SELECT u.id_pengguna, u.nama_pengguna, u.username, COALESCE(u.email, ''), r.nama_role,
		       u.status_aktif, u.last_login
		FROM pengguna u JOIN role r USING (id_role)
		WHERE u.id_pengguna = $1`, user.ID).
		Scan(&akun.ID, &akun.Nama, &akun.Username, &akun.Email, &akun.Role, &akun.Aktif, &akun.LastLogin)
	if err != nil {
		gagal(w, "akun saya", err)
		return
	}
	halaman := h.halaman(r, "Akun saya", "akun")
	if errs != nil {
		halaman.Pesan = ""
	}
	h.render(w, status, "karyawan/akun", akunSayaData{Halaman: halaman, Akun: akun, Errors: errs})
}

func (h *Handler) akunSaya(w http.ResponseWriter, r *http.Request) {
	h.tampilAkunSaya(w, r, http.StatusOK, nil)
}

func (h *Handler) gantiPasswordSendiri(w http.ResponseWriter, r *http.Request) {
	if !bacaForm(w, r) {
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	errs := map[string]string{}
	lama := r.PostForm.Get("password_lama")
	baru := bacaPassword(r, true, errs)

	var hashLama string
	err := h.pool.QueryRow(r.Context(), `SELECT password_hash FROM pengguna WHERE id_pengguna = $1`, user.ID).Scan(&hashLama)
	if err != nil {
		gagal(w, "password lama", err)
		return
	}
	if lama == "" {
		errs["password_lama"] = "Isi password saat ini."
	} else if len(lama) > 72 || bcrypt.CompareHashAndPassword([]byte(hashLama), []byte(lama)) != nil {
		errs["password_lama"] = "Password saat ini salah."
	}
	if len(errs) > 0 {
		h.tampilAkunSaya(w, r, http.StatusUnprocessableEntity, errs)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(baru), bcrypt.DefaultCost)
	if err != nil {
		gagal(w, "hash password", err)
		return
	}
	// Compare against the old hash so a concurrent reset by the owner is not overwritten.
	tag, err := h.pool.Exec(r.Context(), `
		UPDATE pengguna SET password_hash = $3, updated_at = now()
		WHERE id_pengguna = $1 AND password_hash = $2`, user.ID, hashLama, string(hash))
	if err != nil {
		gagal(w, "ganti password", err)
		return
	}
	if tag.RowsAffected() == 0 {
		errs["password_lama"] = "Password Anda baru saja diubah. Muat ulang halaman lalu coba lagi."
		h.tampilAkunSaya(w, r, http.StatusConflict, errs)
		return
	}
	http.Redirect(w, r, "/akun?pesan=password-ubah", http.StatusSeeOther)
}
