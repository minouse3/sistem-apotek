package obat

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
)

const (
	maksHarga  = 999_999_999
	maksFaktor = 100_000
)

type ringkasObat struct {
	ID            int
	Kode          string
	Nama          string
	Kandungan     string
	SatuanDasar   string
	Rak           string
	Aktif         bool
	JumlahKemasan int
}

type daftarObatData struct {
	Halaman
	Q      string
	Status string
	Daftar []ringkasObat
}

func (h *Handler) daftarObat(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if utf8.RuneCountInString(q) > 100 {
		q = string([]rune(q)[:100])
	}
	status := r.URL.Query().Get("status")
	if status != "nonaktif" && status != "semua" {
		status = "aktif"
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT o.id_obat, o.kode_obat, o.nama_obat, COALESCE(o.kandungan, ''), s.nama_satuan,
		       o.lokasi_rak, o.status_aktif,
		       (SELECT count(*) FROM obat_satuan k WHERE k.id_obat = o.id_obat AND k.aktif)
		FROM obat o
		JOIN satuan s ON s.id_satuan = o.satuan_dasar_id
		WHERE ($1 = '' OR o.kode_obat ILIKE '%' || $1 || '%' OR o.nama_obat ILIKE '%' || $1 || '%')
		  AND ($2 = 'semua' OR o.status_aktif = ($2 = 'aktif'))
		ORDER BY o.nama_obat, o.kode_obat
		LIMIT 200`, escapeLike(q), status)
	if err != nil {
		gagal(w, "daftar obat", err)
		return
	}
	daftar, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ringkasObat, error) {
		var o ringkasObat
		err := row.Scan(&o.ID, &o.Kode, &o.Nama, &o.Kandungan, &o.SatuanDasar, &o.Rak, &o.Aktif, &o.JumlahKemasan)
		return o, err
	})
	if err != nil {
		gagal(w, "daftar obat", err)
		return
	}

	h.render(w, http.StatusOK, "obat/daftar", daftarObatData{
		Halaman: h.halaman(r, "Data obat", "obat"),
		Q:       q,
		Status:  status,
		Daftar:  daftar,
	})
}

type satuan struct {
	ID   int
	Nama string
}

func (h *Handler) semuaSatuan(ctx context.Context) ([]satuan, error) {
	rows, err := h.pool.Query(ctx, `SELECT id_satuan, nama_satuan FROM satuan ORDER BY nama_satuan`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[satuan])
}

type isianObat struct {
	Kode          string
	Nama          string
	Kandungan     string
	Rak           string
	SatuanDasarID int
	HargaDasar    string
}

type formObatData struct {
	Halaman
	Baru             bool
	Aksi             string
	ID               int
	Isian            isianObat
	Satuan           []satuan
	KunciSatuanDasar bool
	Errors           map[string]string
}

func (h *Handler) formObatBaru(w http.ResponseWriter, r *http.Request) {
	h.tampilFormObat(w, r, http.StatusOK, formObatData{
		Baru:  true,
		Aksi:  "/obat",
		Isian: isianObat{Rak: "RAK-UMUM", HargaDasar: "0"},
	})
}

func (h *Handler) tampilFormObat(w http.ResponseWriter, r *http.Request, status int, data formObatData) {
	daftar, err := h.semuaSatuan(r.Context())
	if err != nil {
		gagal(w, "daftar satuan", err)
		return
	}
	title := "Ubah obat"
	if data.Baru {
		title = "Tambah obat"
	}
	data.Halaman = h.halaman(r, title, "obat")
	data.Halaman.Pesan = ""
	data.Satuan = daftar
	h.render(w, status, "obat/form", data)
}

func bacaIsianObat(r *http.Request) (isianObat, map[string]string) {
	isian := isianObat{
		Kode:       strings.ToUpper(strings.TrimSpace(r.PostForm.Get("kode_obat"))),
		Nama:       strings.TrimSpace(r.PostForm.Get("nama_obat")),
		Kandungan:  strings.TrimSpace(r.PostForm.Get("kandungan")),
		Rak:        strings.ToUpper(strings.TrimSpace(r.PostForm.Get("lokasi_rak"))),
		HargaDasar: strings.TrimSpace(r.PostForm.Get("harga_jual")),
	}
	isian.SatuanDasarID, _ = bilanganBulat(r.PostForm.Get("satuan_dasar_id"), 1, 1<<31-1)

	errs := map[string]string{}
	switch {
	case isian.Kode == "":
		errs["kode_obat"] = "Isi kode obat."
	case utf8.RuneCountInString(isian.Kode) > 30:
		errs["kode_obat"] = "Kode obat maksimal 30 karakter."
	}
	switch {
	case isian.Nama == "":
		errs["nama_obat"] = "Isi nama obat."
	case utf8.RuneCountInString(isian.Nama) > 150:
		errs["nama_obat"] = "Nama obat maksimal 150 karakter."
	}
	if utf8.RuneCountInString(isian.Kandungan) > 1000 {
		errs["kandungan"] = "Kandungan maksimal 1.000 karakter."
	}
	if isian.Rak == "" {
		isian.Rak = "RAK-UMUM"
	}
	if utf8.RuneCountInString(isian.Rak) > 50 {
		errs["lokasi_rak"] = "Lokasi rak maksimal 50 karakter."
	}
	if isian.SatuanDasarID == 0 {
		errs["satuan_dasar_id"] = "Pilih satuan dasar."
	}
	return isian, errs
}

func (h *Handler) simpanObatBaru(w http.ResponseWriter, r *http.Request) {
	if !bacaForm(w, r) {
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	isian, errs := bacaIsianObat(r)

	harga := 0
	if user.Role == auth.RolePemilik {
		var ok bool
		if isian.HargaDasar == "" {
			isian.HargaDasar = "0"
		}
		if harga, ok = bilanganBulat(isian.HargaDasar, 0, maksHarga); !ok {
			errs["harga_jual"] = "Isi harga jual berupa angka bulat dari 0 sampai 999.999.999."
		}
	}
	if len(errs) > 0 {
		h.tampilFormObat(w, r, http.StatusUnprocessableEntity, formObatData{Baru: true, Aksi: "/obat", Isian: isian, Errors: errs})
		return
	}

	var id int
	err := pgx.BeginFunc(r.Context(), h.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(r.Context(), `
			INSERT INTO obat (kode_obat, nama_obat, kandungan, satuan_dasar_id, lokasi_rak)
			VALUES ($1, $2, NULLIF($3, ''), $4, $5)
			RETURNING id_obat`,
			isian.Kode, isian.Nama, isian.Kandungan, isian.SatuanDasarID, isian.Rak).Scan(&id)
		if err != nil {
			return err
		}
		// Every medicine is also sold and received in its base unit at factor 1.
		_, err = tx.Exec(r.Context(), `
			INSERT INTO obat_satuan (id_obat, id_satuan, faktor_konversi, harga_jual)
			VALUES ($1, $2, 1, $3)`, id, isian.SatuanDasarID, harga)
		return err
	})
	if pelanggaranUnik(err, "obat_kode_obat_key") {
		errs["kode_obat"] = "Kode obat ini sudah dipakai. Gunakan kode lain."
		h.tampilFormObat(w, r, http.StatusConflict, formObatData{Baru: true, Aksi: "/obat", Isian: isian, Errors: errs})
		return
	}
	if fkGagal(err) {
		errs["satuan_dasar_id"] = "Satuan dasar tidak ditemukan. Pilih ulang."
		h.tampilFormObat(w, r, http.StatusUnprocessableEntity, formObatData{Baru: true, Aksi: "/obat", Isian: isian, Errors: errs})
		return
	}
	if err != nil {
		gagal(w, "tambah obat", err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/obat/%d?pesan=obat-baru", id), http.StatusSeeOther)
}

// punyaRiwayat reports whether stock or transactions already depend on the
// medicine's conversion factors.
func punyaRiwayat(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, idObat int) (bool, error) {
	var ada bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM batch WHERE id_obat = $1)
		    OR EXISTS (SELECT 1 FROM detail_penerimaan d JOIN obat_satuan k USING (id_obat_satuan) WHERE k.id_obat = $1)
		    OR EXISTS (SELECT 1 FROM item_penjualan i JOIN obat_satuan k USING (id_obat_satuan) WHERE k.id_obat = $1)`,
		idObat).Scan(&ada)
	return ada, err
}

type kemasan struct {
	ID          int
	IDSatuan    int
	NamaSatuan  string
	Faktor      int
	Harga       int
	BolehDijual bool
	Aktif       bool
	Dasar       bool
}

type detailObat struct {
	ID          int
	Kode        string
	Nama        string
	Kandungan   string
	SatuanDasar string
	IDDasar     int
	Rak         string
	Aktif       bool
}

type detailObatData struct {
	Halaman
	Obat         detailObat
	Kemasan      []kemasan
	PunyaRiwayat bool
}

func (h *Handler) ambilObat(ctx context.Context, id int) (detailObat, error) {
	var o detailObat
	err := h.pool.QueryRow(ctx, `
		SELECT o.id_obat, o.kode_obat, o.nama_obat, COALESCE(o.kandungan, ''), s.nama_satuan, s.id_satuan,
		       o.lokasi_rak, o.status_aktif
		FROM obat o JOIN satuan s ON s.id_satuan = o.satuan_dasar_id
		WHERE o.id_obat = $1`, id).
		Scan(&o.ID, &o.Kode, &o.Nama, &o.Kandungan, &o.SatuanDasar, &o.IDDasar, &o.Rak, &o.Aktif)
	return o, err
}

func (h *Handler) detailObat(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	o, err := h.ambilObat(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		gagal(w, "detail obat", err)
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT k.id_obat_satuan, k.id_satuan, s.nama_satuan, k.faktor_konversi, k.harga_jual,
		       k.boleh_dijual, k.aktif, k.id_satuan = $2
		FROM obat_satuan k JOIN satuan s USING (id_satuan)
		WHERE k.id_obat = $1
		ORDER BY k.faktor_konversi, s.nama_satuan`, id, o.IDDasar)
	if err != nil {
		gagal(w, "kemasan obat", err)
		return
	}
	daftar, err := pgx.CollectRows(rows, pgx.RowToStructByPos[kemasan])
	if err != nil {
		gagal(w, "kemasan obat", err)
		return
	}
	riwayat, err := punyaRiwayat(r.Context(), h.pool, id)
	if err != nil {
		gagal(w, "riwayat obat", err)
		return
	}

	h.render(w, http.StatusOK, "obat/detail", detailObatData{
		Halaman:      h.halaman(r, o.Nama, "obat"),
		Obat:         o,
		Kemasan:      daftar,
		PunyaRiwayat: riwayat,
	})
}

func (h *Handler) formUbahObat(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	o, err := h.ambilObat(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		gagal(w, "ubah obat", err)
		return
	}
	riwayat, err := punyaRiwayat(r.Context(), h.pool, id)
	if err != nil {
		gagal(w, "riwayat obat", err)
		return
	}
	h.tampilFormObat(w, r, http.StatusOK, formObatData{
		Aksi: fmt.Sprintf("/obat/%d/ubah", id),
		ID:   id,
		Isian: isianObat{
			Kode: o.Kode, Nama: o.Nama, Kandungan: o.Kandungan, Rak: o.Rak, SatuanDasarID: o.IDDasar,
		},
		KunciSatuanDasar: riwayat,
	})
}

var errRiwayat = errors.New("medicine has stock history")

func (h *Handler) simpanUbahObat(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !bacaForm(w, r) {
		return
	}
	isian, errs := bacaIsianObat(r)
	data := formObatData{Aksi: fmt.Sprintf("/obat/%d/ubah", id), ID: id, Isian: isian, Errors: errs}
	if len(errs) > 0 {
		h.tampilFormObat(w, r, http.StatusUnprocessableEntity, data)
		return
	}

	err := pgx.BeginFunc(r.Context(), h.pool, func(tx pgx.Tx) error {
		var dasarLama int
		err := tx.QueryRow(r.Context(), `SELECT satuan_dasar_id FROM obat WHERE id_obat = $1 FOR UPDATE`, id).Scan(&dasarLama)
		if err != nil {
			return err
		}
		if dasarLama != isian.SatuanDasarID {
			// Stock is stored in base units, so the base unit is fixed once stock exists.
			riwayat, err := punyaRiwayat(r.Context(), tx, id)
			if err != nil {
				return err
			}
			if riwayat {
				return errRiwayat
			}
			if _, err := tx.Exec(r.Context(), `
				UPDATE obat_satuan SET id_satuan = $3
				WHERE id_obat = $1 AND id_satuan = $2`, id, dasarLama, isian.SatuanDasarID); err != nil {
				return err
			}
		}
		_, err = tx.Exec(r.Context(), `
			UPDATE obat SET kode_obat = $2, nama_obat = $3, kandungan = NULLIF($4, ''),
			                satuan_dasar_id = $5, lokasi_rak = $6
			WHERE id_obat = $1`,
			id, isian.Kode, isian.Nama, isian.Kandungan, isian.SatuanDasarID, isian.Rak)
		return err
	})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		http.NotFound(w, r)
	case errors.Is(err, errRiwayat):
		data.KunciSatuanDasar = true
		errs["satuan_dasar_id"] = "Satuan dasar tidak bisa diganti karena obat ini sudah punya stok atau transaksi."
		h.tampilFormObat(w, r, http.StatusConflict, data)
	case pelanggaranUnik(err, "obat_kode_obat_key"):
		errs["kode_obat"] = "Kode obat ini sudah dipakai. Gunakan kode lain."
		h.tampilFormObat(w, r, http.StatusConflict, data)
	case pelanggaranUnik(err, "obat_satuan_id_obat_id_satuan_key"):
		errs["satuan_dasar_id"] = "Satuan ini sudah terdaftar sebagai kemasan obat ini. Pilih satuan lain."
		h.tampilFormObat(w, r, http.StatusConflict, data)
	case fkGagal(err):
		errs["satuan_dasar_id"] = "Satuan dasar tidak ditemukan. Pilih ulang."
		h.tampilFormObat(w, r, http.StatusUnprocessableEntity, data)
	case err != nil:
		gagal(w, "ubah obat", err)
	default:
		http.Redirect(w, r, fmt.Sprintf("/obat/%d?pesan=obat-ubah", id), http.StatusSeeOther)
	}
}

func (h *Handler) ubahStatusObat(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !bacaForm(w, r) {
		return
	}
	aktif := r.PostForm.Get("aktif") == "true"
	tag, err := h.pool.Exec(r.Context(), `UPDATE obat SET status_aktif = $2 WHERE id_obat = $1`, id, aktif)
	if err != nil {
		gagal(w, "status obat", err)
		return
	}
	if tag.RowsAffected() == 0 {
		http.NotFound(w, r)
		return
	}
	pesan := "obat-nonaktif"
	if aktif {
		pesan = "obat-aktif"
	}
	http.Redirect(w, r, fmt.Sprintf("/obat/%d?pesan=%s", id, pesan), http.StatusSeeOther)
}
