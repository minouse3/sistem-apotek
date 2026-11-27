package obat

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
	"github.com/minouse3/sistem-apotek/internal/view"
)

type Handler struct {
	pool     *pgxpool.Pool
	renderer *view.Renderer
}

type Halaman struct {
	view.PageMeta
	User  auth.User
	Pesan string
}

func (h Halaman) Pemilik() bool { return h.User.Role == auth.RolePemilik }

func (h Halaman) BolehKelolaObat() bool {
	return h.User.Role == auth.RolePemilik || h.User.Role == auth.RoleStaf
}

func NewHandler(pool *pgxpool.Pool, renderer *view.Renderer) *Handler {
	return &Handler{pool: pool, renderer: renderer}
}

func (h *Handler) Register(mux *http.ServeMux, accounts *auth.Service) {
	semua := func(fn http.HandlerFunc) http.Handler { return middleware.Require(accounts, fn) }
	kelola := func(fn http.HandlerFunc) http.Handler {
		return middleware.Require(accounts, fn, auth.RoleStaf, auth.RolePemilik)
	}
	pemilik := func(fn http.HandlerFunc) http.Handler {
		return middleware.Require(accounts, fn, auth.RolePemilik)
	}

	mux.Handle("GET /obat", semua(h.daftarObat))
	mux.Handle("GET /obat/baru", kelola(h.formObatBaru))
	mux.Handle("POST /obat", kelola(h.simpanObatBaru))
	mux.Handle("GET /obat/{id}", semua(h.detailObat))
	mux.Handle("GET /obat/{id}/ubah", kelola(h.formUbahObat))
	mux.Handle("POST /obat/{id}/ubah", kelola(h.simpanUbahObat))
	mux.Handle("POST /obat/{id}/status", kelola(h.ubahStatusObat))
	mux.Handle("GET /obat/{id}/kemasan/baru", kelola(h.formKemasanBaru))
	mux.Handle("POST /obat/{id}/kemasan", kelola(h.simpanKemasanBaru))
	mux.Handle("GET /obat/{id}/kemasan/{kid}/ubah", kelola(h.formUbahKemasan))
	mux.Handle("POST /obat/{id}/kemasan/{kid}/ubah", kelola(h.simpanUbahKemasan))

	mux.Handle("GET /satuan", kelola(h.daftarSatuan))
	mux.Handle("POST /satuan", kelola(h.simpanSatuan))
	mux.Handle("GET /satuan/{id}/ubah", kelola(h.formUbahSatuan))
	mux.Handle("POST /satuan/{id}/ubah", kelola(h.simpanUbahSatuan))

	mux.Handle("GET /akun", semua(h.akunSaya))
	mux.Handle("POST /akun/password", semua(h.gantiPasswordSendiri))

	mux.Handle("GET /karyawan", pemilik(h.daftarKaryawan))
	mux.Handle("GET /karyawan/baru", pemilik(h.formKaryawanBaru))
	mux.Handle("POST /karyawan", pemilik(h.simpanKaryawanBaru))
	mux.Handle("GET /karyawan/{id}/ubah", pemilik(h.formUbahKaryawan))
	mux.Handle("POST /karyawan/{id}/ubah", pemilik(h.simpanUbahKaryawan))
	mux.Handle("POST /karyawan/{id}/status", pemilik(h.ubahStatusKaryawan))
}

func (h *Handler) halaman(r *http.Request, title, active string) Halaman {
	user, _ := middleware.CurrentUser(r.Context())
	return Halaman{
		PageMeta: view.NewPageMeta(user, title, active),
		User:     user,
		Pesan:    pesanSukses[r.URL.Query().Get("pesan")],
	}
}

// Redirect messages are looked up by key so the query string cannot inject text.
var pesanSukses = map[string]string{
	"obat-baru":     "Obat berhasil ditambahkan.",
	"obat-ubah":     "Perubahan data obat tersimpan.",
	"obat-aktif":    "Obat diaktifkan kembali.",
	"obat-nonaktif": "Obat dinonaktifkan. Riwayatnya tetap tersimpan.",
	"kemasan-baru":  "Satuan kemasan ditambahkan.",
	"kemasan-ubah":  "Perubahan satuan kemasan tersimpan.",
	"satuan-baru":   "Satuan ditambahkan.",
	"satuan-ubah":   "Nama satuan diperbarui.",
	"password-ubah": "Password Anda berhasil diganti.",
	"karyawan-baru": "Akun karyawan berhasil dibuat.",
	"karyawan-ubah": "Perubahan akun tersimpan.",
	"akun-aktif":    "Akun diaktifkan kembali.",
	"akun-nonaktif": "Akun dinonaktifkan dan sesinya diakhiri.",
}

func (h *Handler) render(w http.ResponseWriter, status int, page string, data any) {
	h.renderer.Render(w, status, "templates/"+page+".html", data)
}

func gagal(w http.ResponseWriter, where string, err error) {
	log.Printf("%s: %v", where, err)
	http.Error(w, "Terjadi kesalahan pada server. Coba lagi.", http.StatusInternalServerError)
}

func pathID(r *http.Request, name string) (int, bool) {
	id, err := strconv.Atoi(r.PathValue(name))
	return id, err == nil && id > 0
}

func bacaForm(w http.ResponseWriter, r *http.Request) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulir tidak dapat dibaca. Isi kembali lalu coba lagi.", http.StatusBadRequest)
		return false
	}
	return true
}

func pelanggaranUnik(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}

// escapeLike makes user input match literally inside ILIKE patterns.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// bilanganBulat parses a whole number typed with optional thousands dots.
func bilanganBulat(s string, min, max int) (int, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ".", "")
	n, err := strconv.Atoi(s)
	return n, err == nil && n >= min && n <= max
}

func fkGagal(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
