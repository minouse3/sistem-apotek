package obat

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

type satuanTerpakai struct {
	ID         int
	Nama       string
	JumlahObat int
}

type daftarSatuanData struct {
	Halaman
	Daftar []satuanTerpakai
	Nama   string
	Error  string
}

func (h *Handler) tampilSatuan(w http.ResponseWriter, r *http.Request, status int, nama, pesanError string) {
	rows, err := h.pool.Query(r.Context(), `
		SELECT s.id_satuan, s.nama_satuan, count(DISTINCT k.id_obat)
		FROM satuan s LEFT JOIN obat_satuan k USING (id_satuan)
		GROUP BY s.id_satuan
		ORDER BY s.nama_satuan`)
	if err != nil {
		gagal(w, "daftar satuan", err)
		return
	}
	daftar, err := pgx.CollectRows(rows, pgx.RowToStructByPos[satuanTerpakai])
	if err != nil {
		gagal(w, "daftar satuan", err)
		return
	}
	halaman := h.halaman(r, "Satuan", "satuan")
	if pesanError != "" {
		halaman.Pesan = ""
	}
	h.render(w, status, "obat/satuan", daftarSatuanData{Halaman: halaman, Daftar: daftar, Nama: nama, Error: pesanError})
}

func (h *Handler) daftarSatuan(w http.ResponseWriter, r *http.Request) {
	h.tampilSatuan(w, r, http.StatusOK, "", "")
}

func (h *Handler) simpanSatuan(w http.ResponseWriter, r *http.Request) {
	if !bacaForm(w, r) {
		return
	}
	nama, pesan := bacaNamaSatuan(r)
	if pesan != "" {
		h.tampilSatuan(w, r, http.StatusUnprocessableEntity, nama, pesan)
		return
	}

	_, err := h.pool.Exec(r.Context(), `INSERT INTO satuan (nama_satuan) VALUES ($1)`, nama)
	if pelanggaranUnik(err, "satuan_nama_satuan_key") {
		h.tampilSatuan(w, r, http.StatusConflict, nama, "Satuan ini sudah ada.")
		return
	}
	if err != nil {
		gagal(w, "tambah satuan", err)
		return
	}
	http.Redirect(w, r, "/satuan?pesan=satuan-baru", http.StatusSeeOther)
}

type ubahSatuanData struct {
	Halaman
	ID    int
	Nama  string
	Error string
}

func bacaNamaSatuan(r *http.Request) (string, string) {
	nama := strings.ToLower(strings.Join(strings.Fields(r.PostForm.Get("nama_satuan")), " "))
	switch {
	case nama == "":
		return nama, "Isi nama satuan."
	case utf8.RuneCountInString(nama) > 50:
		return nama, "Nama satuan maksimal 50 karakter."
	}
	return nama, ""
}

func (h *Handler) formUbahSatuan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	var nama string
	err := h.pool.QueryRow(r.Context(), `SELECT nama_satuan FROM satuan WHERE id_satuan = $1`, id).Scan(&nama)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		gagal(w, "ubah satuan", err)
		return
	}
	h.render(w, http.StatusOK, "obat/satuan_ubah", ubahSatuanData{
		Halaman: h.halaman(r, "Ubah satuan", "satuan"), ID: id, Nama: nama,
	})
}

func (h *Handler) simpanUbahSatuan(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !bacaForm(w, r) {
		return
	}
	nama, pesan := bacaNamaSatuan(r)
	tampil := func(status int, pesan string) {
		h.render(w, status, "obat/satuan_ubah", ubahSatuanData{
			Halaman: h.halaman(r, "Ubah satuan", "satuan"), ID: id, Nama: nama, Error: pesan,
		})
	}
	if pesan != "" {
		tampil(http.StatusUnprocessableEntity, pesan)
		return
	}

	tag, err := h.pool.Exec(r.Context(), `UPDATE satuan SET nama_satuan = $2 WHERE id_satuan = $1`, id, nama)
	switch {
	case pelanggaranUnik(err, "satuan_nama_satuan_key"):
		tampil(http.StatusConflict, "Nama satuan ini sudah dipakai.")
	case err != nil:
		gagal(w, "ubah satuan", err)
	case tag.RowsAffected() == 0:
		http.NotFound(w, r)
	default:
		http.Redirect(w, r, "/satuan?pesan=satuan-ubah", http.StatusSeeOther)
	}
}
