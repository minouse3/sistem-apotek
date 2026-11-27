package penerimaan

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/minouse3/sistem-apotek/internal/middleware"
	"github.com/minouse3/sistem-apotek/internal/view"
)

var (
	ErrNameTaken = errors.New("PBF distributor name already exists")
	ErrNotFound  = errors.New("PBF distributor not found")
)

type Distributor struct {
	ID     int
	Name   string
	Active bool
}

type distributorStore interface {
	List(context.Context) ([]Distributor, error)
	Create(context.Context, string) error
	Rename(context.Context, int, string) error
	SetActive(context.Context, int, bool) error
}

type postgresDistributorStore struct {
	pool *pgxpool.Pool
}

type pbfHandler struct {
	store    distributorStore
	renderer pageRenderer
}

type pbfPageData struct {
	view.PageMeta
	Distributors []Distributor
	Name         string
	NameError    string
	EditID       int
	EditName     string
	EditError    string
	Error        string
	Notice       string
	Empty        bool
}

func newHandler(store distributorStore, renderer pageRenderer) http.Handler {
	handler := &pbfHandler{store: store, renderer: renderer}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /pbf", handler.list)
	mux.HandleFunc("GET /pbf/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/pbf", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /pbf", handler.create)
	mux.HandleFunc("POST /pbf/{id}/edit", handler.rename)
	mux.HandleFunc("POST /pbf/{id}/status", handler.setActive)
	return mux
}

func (s postgresDistributorStore) List(ctx context.Context) ([]Distributor, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id_pbf, nama_pbf, aktif
		FROM pbf
		ORDER BY aktif DESC, nama_pbf, id_pbf`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	distributors := make([]Distributor, 0)
	for rows.Next() {
		var distributor Distributor
		if err := rows.Scan(&distributor.ID, &distributor.Name, &distributor.Active); err != nil {
			return nil, err
		}
		distributors = append(distributors, distributor)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return distributors, nil
}

func (s postgresDistributorStore) Create(ctx context.Context, name string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO pbf (nama_pbf) VALUES ($1)`, name)
	return distributorWriteError(err)
}

func (s postgresDistributorStore) Rename(ctx context.Context, id int, name string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE pbf SET nama_pbf = $2 WHERE id_pbf = $1`, id, name)
	if err != nil {
		return distributorWriteError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s postgresDistributorStore) SetActive(ctx context.Context, id int, active bool) error {
	tag, err := s.pool.Exec(ctx, `UPDATE pbf SET aktif = $2 WHERE id_pbf = $1`, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func distributorWriteError(err error) error {
	if err == nil {
		return nil
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) &&
		postgresError.Code == "23505" &&
		postgresError.ConstraintName == "pbf_nama_pbf_key" {
		return ErrNameTaken
	}
	return err
}

func (h *pbfHandler) list(w http.ResponseWriter, r *http.Request) {
	data := pbfPageData{Notice: noticeFor(r.URL.Query().Get("status"))}
	h.renderPage(w, r, http.StatusOK, data)
}

func (h *pbfHandler) create(w http.ResponseWriter, r *http.Request) {
	name, err := readDistributorName(w, r)
	if err != nil {
		h.renderPage(w, r, http.StatusBadRequest, pbfPageData{
			Name:  name,
			Error: "Formulir tidak dapat dibaca. Isi kembali lalu coba lagi.",
		})
		return
	}
	if message := validateDistributorName(name); message != "" {
		h.renderPage(w, r, http.StatusUnprocessableEntity, pbfPageData{Name: name, NameError: message})
		return
	}
	if err := h.store.Create(r.Context(), name); errors.Is(err, ErrNameTaken) {
		h.renderPage(w, r, http.StatusConflict, pbfPageData{
			Name:      name,
			NameError: "Nama distributor sudah terdaftar.",
		})
		return
	} else if err != nil {
		log.Printf("create PBF distributor: %v", err)
		h.renderPage(w, r, http.StatusInternalServerError, pbfPageData{
			Name:  name,
			Error: "Tidak dapat menyimpan distributor PBF. Coba lagi.",
		})
		return
	}
	http.Redirect(w, r, "/pbf?status=created", http.StatusSeeOther)
}

func (h *pbfHandler) rename(w http.ResponseWriter, r *http.Request) {
	id, err := parseDistributorID(r)
	if err != nil {
		http.Error(w, "ID distributor tidak valid.", http.StatusBadRequest)
		return
	}
	name, err := readDistributorName(w, r)
	if err != nil {
		h.renderPage(w, r, http.StatusBadRequest, pbfPageData{
			Error: "Formulir tidak dapat dibaca. Isi kembali lalu coba lagi.",
		})
		return
	}
	if message := validateDistributorName(name); message != "" {
		h.renderPage(w, r, http.StatusUnprocessableEntity, pbfPageData{
			EditID:    id,
			EditName:  name,
			EditError: message,
			Error:     "Perubahan nama belum disimpan.",
		})
		return
	}
	if err := h.store.Rename(r.Context(), id, name); errors.Is(err, ErrNameTaken) {
		h.renderPage(w, r, http.StatusConflict, pbfPageData{
			EditID:    id,
			EditName:  name,
			EditError: "Nama distributor sudah terdaftar.",
			Error:     "Perubahan nama belum disimpan.",
		})
		return
	} else if errors.Is(err, ErrNotFound) {
		http.Error(w, "Distributor PBF tidak ditemukan.", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("rename PBF distributor: %v", err)
		h.renderPage(w, r, http.StatusInternalServerError, pbfPageData{
			Error: "Tidak dapat mengubah distributor PBF. Coba lagi.",
		})
		return
	}
	http.Redirect(w, r, "/pbf?status=updated", http.StatusSeeOther)
}

func (h *pbfHandler) setActive(w http.ResponseWriter, r *http.Request) {
	id, err := parseDistributorID(r)
	if err != nil {
		http.Error(w, "ID distributor tidak valid.", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		h.renderPage(w, r, http.StatusBadRequest, pbfPageData{
			Error: "Formulir tidak dapat dibaca. Isi kembali lalu coba lagi.",
		})
		return
	}
	var active bool
	switch r.PostForm.Get("aktif") {
	case "true":
		active = true
	case "false":
	default:
		h.renderPage(w, r, http.StatusUnprocessableEntity, pbfPageData{
			Error: "Status distributor tidak valid. Muat ulang halaman lalu coba lagi.",
		})
		return
	}
	if err := h.store.SetActive(r.Context(), id, active); errors.Is(err, ErrNotFound) {
		http.Error(w, "Distributor PBF tidak ditemukan.", http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("change PBF distributor status: %v", err)
		h.renderPage(w, r, http.StatusInternalServerError, pbfPageData{
			Error: "Tidak dapat mengubah status distributor PBF. Coba lagi.",
		})
		return
	}
	status := "deactivated"
	if active {
		status = "activated"
	}
	http.Redirect(w, r, "/pbf?status="+status, http.StatusSeeOther)
}

func readDistributorName(w http.ResponseWriter, r *http.Request) (string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return strings.TrimSpace(r.PostForm.Get("nama_pbf")), nil
}

func validateDistributorName(name string) string {
	switch {
	case name == "":
		return "Nama distributor wajib diisi."
	case !utf8.ValidString(name):
		return "Nama distributor mengandung karakter yang tidak valid."
	case utf8.RuneCountInString(name) > 150:
		return "Nama distributor maksimal 150 karakter."
	case strings.ContainsRune(name, 0):
		return "Nama distributor mengandung karakter yang tidak valid."
	default:
		return ""
	}
}

func parseDistributorID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, errors.New("invalid distributor ID")
	}
	return id, nil
}

func noticeFor(status string) string {
	switch status {
	case "created":
		return "Distributor PBF berhasil ditambahkan."
	case "updated":
		return "Perubahan distributor berhasil disimpan."
	case "deactivated":
		return "Distributor PBF dinonaktifkan."
	case "activated":
		return "Distributor PBF diaktifkan kembali."
	default:
		return ""
	}
}

func (h *pbfHandler) renderPage(w http.ResponseWriter, r *http.Request, status int, data pbfPageData) {
	user, _ := middleware.CurrentUser(r.Context())
	data.PageMeta = view.NewPageMeta(user, "Distributor PBF", "pbf")

	distributors, err := h.store.List(r.Context())
	if err != nil {
		log.Printf("list PBF distributors: %v", err)
		status = http.StatusInternalServerError
		data.Error = "Tidak dapat memuat distributor PBF."
		data.Distributors = nil
		data.Empty = false
	} else {
		data.Distributors = distributors
		data.Empty = len(distributors) == 0
	}

	h.renderer.Render(w, status, "templates/penerimaan/pbf.html", data)
}
