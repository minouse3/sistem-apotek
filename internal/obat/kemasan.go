package obat

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
)

const pesanKunciKonversi = "Obat ini sudah punya stok atau transaksi. Satuan kemasan dan faktor konversinya hanya bisa diatur oleh Pemilik."

type isianKemasan struct {
	IDSatuan    int
	NamaSatuan  string
	Faktor      string
	Harga       string
	BolehDijual bool
	Aktif       bool
}

type formKemasanData struct {
	Halaman
	Obat        detailObat
	Baru        bool
	Aksi        string
	Isian       isianKemasan
	Dasar       bool
	KunciFaktor bool
	Ditolak     string
	Satuan      []satuan
	Errors      map[string]string
}

func (h *Handler) tampilFormKemasan(w http.ResponseWriter, r *http.Request, status int, data formKemasanData) {
	if data.Baru {
		daftar, err := h.semuaSatuan(r.Context())
		if err != nil {
			gagal(w, "daftar satuan", err)
			return
		}
		data.Satuan = daftar
	}
	title := "Ubah satuan kemasan"
	if data.Baru {
		title = "Tambah satuan kemasan"
	}
	data.Halaman = h.halaman(r, title, "obat")
	data.Halaman.Pesan = ""
	h.render(w, status, "obat/kemasan", data)
}

// stafTerkunci applies the "Terbatas" rule: staff may set conversions only
// while the medicine has no stock or transaction history.
func stafTerkunci(r *http.Request, riwayat bool) bool {
	user, _ := middleware.CurrentUser(r.Context())
	return riwayat && user.Role != auth.RolePemilik
}

func (h *Handler) obatUntukKemasan(w http.ResponseWriter, r *http.Request) (detailObat, bool, bool) {
	id, ok := pathID(r, "id")
	if !ok {
		http.NotFound(w, r)
		return detailObat{}, false, false
	}
	o, err := h.ambilObat(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return o, false, false
	}
	if err != nil {
		gagal(w, "obat", err)
		return o, false, false
	}
	riwayat, err := punyaRiwayat(r.Context(), h.pool, id)
	if err != nil {
		gagal(w, "riwayat obat", err)
		return o, false, false
	}
	return o, riwayat, true
}

func (h *Handler) formKemasanBaru(w http.ResponseWriter, r *http.Request) {
	o, riwayat, ok := h.obatUntukKemasan(w, r)
	if !ok {
		return
	}
	data := formKemasanData{
		Obat:  o,
		Baru:  true,
		Aksi:  fmt.Sprintf("/obat/%d/kemasan", o.ID),
		Isian: isianKemasan{Harga: "0", BolehDijual: true, Aktif: true},
	}
	if stafTerkunci(r, riwayat) {
		data.Ditolak = pesanKunciKonversi
		h.tampilFormKemasan(w, r, http.StatusForbidden, data)
		return
	}
	h.tampilFormKemasan(w, r, http.StatusOK, data)
}

func (h *Handler) simpanKemasanBaru(w http.ResponseWriter, r *http.Request) {
	o, riwayat, ok := h.obatUntukKemasan(w, r)
	if !ok || !bacaForm(w, r) {
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	data := formKemasanData{Obat: o, Baru: true, Aksi: fmt.Sprintf("/obat/%d/kemasan", o.ID), Errors: map[string]string{}}
	if stafTerkunci(r, riwayat) {
		data.Ditolak = pesanKunciKonversi
		h.tampilFormKemasan(w, r, http.StatusForbidden, data)
		return
	}

	isian := isianKemasan{
		Faktor:      r.PostForm.Get("faktor_konversi"),
		Harga:       r.PostForm.Get("harga_jual"),
		BolehDijual: r.PostForm.Get("boleh_dijual") == "on",
		Aktif:       true,
	}
	isian.IDSatuan, _ = strconv.Atoi(r.PostForm.Get("id_satuan"))
	data.Isian = isian
	errs := data.Errors

	if isian.IDSatuan <= 0 {
		errs["id_satuan"] = "Pilih satuan kemasan."
	} else if isian.IDSatuan == o.IDDasar {
		errs["id_satuan"] = "Ini satuan dasar obat. Pilih satuan kemasan lain."
	}
	faktor, ok := bilanganBulat(isian.Faktor, 2, maksFaktor)
	if !ok {
		errs["faktor_konversi"] = fmt.Sprintf("Isi angka bulat dari 2 sampai 100.000 (jumlah %s dalam satu kemasan).", o.SatuanDasar)
	}
	harga := 0
	if user.Role == auth.RolePemilik {
		if harga, ok = bilanganBulat(isian.Harga, 0, maksHarga); !ok {
			errs["harga_jual"] = "Isi harga jual berupa angka bulat dari 0 sampai 999.999.999."
		}
	}
	if len(errs) > 0 {
		h.tampilFormKemasan(w, r, http.StatusUnprocessableEntity, data)
		return
	}

	_, err := h.pool.Exec(r.Context(), `
		INSERT INTO obat_satuan (id_obat, id_satuan, faktor_konversi, harga_jual, boleh_dijual)
		VALUES ($1, $2, $3, $4, $5)`, o.ID, isian.IDSatuan, faktor, harga, isian.BolehDijual)
	switch {
	case pelanggaranUnik(err, "obat_satuan_id_obat_id_satuan_key"):
		errs["id_satuan"] = "Satuan ini sudah terdaftar untuk obat ini. Ubah dari daftar kemasan."
		h.tampilFormKemasan(w, r, http.StatusConflict, data)
	case fkGagal(err):
		errs["id_satuan"] = "Satuan tidak ditemukan. Pilih ulang."
		h.tampilFormKemasan(w, r, http.StatusUnprocessableEntity, data)
	case err != nil:
		gagal(w, "tambah kemasan", err)
	default:
		http.Redirect(w, r, fmt.Sprintf("/obat/%d?pesan=kemasan-baru", o.ID), http.StatusSeeOther)
	}
}

func (h *Handler) formUbahKemasan(w http.ResponseWriter, r *http.Request) {
	o, riwayat, ok := h.obatUntukKemasan(w, r)
	if !ok {
		return
	}
	kid, ok := pathID(r, "kid")
	if !ok {
		http.NotFound(w, r)
		return
	}
	var k kemasan
	err := h.pool.QueryRow(r.Context(), `
		SELECT k.id_obat_satuan, k.id_satuan, s.nama_satuan, k.faktor_konversi, k.harga_jual,
		       k.boleh_dijual, k.aktif, k.id_satuan = $3
		FROM obat_satuan k JOIN satuan s USING (id_satuan)
		WHERE k.id_obat = $1 AND k.id_obat_satuan = $2`, o.ID, kid, o.IDDasar).
		Scan(&k.ID, &k.IDSatuan, &k.NamaSatuan, &k.Faktor, &k.Harga, &k.BolehDijual, &k.Aktif, &k.Dasar)
	if errors.Is(err, pgx.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		gagal(w, "kemasan", err)
		return
	}
	h.tampilFormKemasan(w, r, http.StatusOK, formKemasanData{
		Obat: o,
		Aksi: fmt.Sprintf("/obat/%d/kemasan/%d/ubah", o.ID, k.ID),
		Isian: isianKemasan{
			IDSatuan: k.IDSatuan, NamaSatuan: k.NamaSatuan,
			Faktor: strconv.Itoa(k.Faktor), Harga: strconv.Itoa(k.Harga),
			BolehDijual: k.BolehDijual, Aktif: k.Aktif,
		},
		Dasar:       k.Dasar,
		KunciFaktor: k.Dasar || stafTerkunci(r, riwayat),
	})
}

func (h *Handler) simpanUbahKemasan(w http.ResponseWriter, r *http.Request) {
	o, _, ok := h.obatUntukKemasan(w, r)
	if !ok {
		return
	}
	kid, ok := pathID(r, "kid")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !bacaForm(w, r) {
		return
	}
	user, _ := middleware.CurrentUser(r.Context())
	pemilik := user.Role == auth.RolePemilik

	isian := isianKemasan{
		Faktor:      r.PostForm.Get("faktor_konversi"),
		Harga:       r.PostForm.Get("harga_jual"),
		BolehDijual: r.PostForm.Get("boleh_dijual") == "on",
		Aktif:       r.PostForm.Get("aktif") == "on",
	}
	data := formKemasanData{Obat: o, Aksi: fmt.Sprintf("/obat/%d/kemasan/%d/ubah", o.ID, kid), Errors: map[string]string{}}
	errs := data.Errors

	err := pgx.BeginFunc(r.Context(), h.pool, func(tx pgx.Tx) error {
		var faktorLama, hargaLama int
		err := tx.QueryRow(r.Context(), `
			SELECT k.id_satuan, s.nama_satuan, k.faktor_konversi, k.harga_jual, k.id_satuan = o.satuan_dasar_id
			FROM obat_satuan k
			JOIN satuan s USING (id_satuan)
			JOIN obat o USING (id_obat)
			WHERE k.id_obat = $1 AND k.id_obat_satuan = $2
			FOR UPDATE OF k, o`, o.ID, kid).
			Scan(&isian.IDSatuan, &isian.NamaSatuan, &faktorLama, &hargaLama, &data.Dasar)
		if err != nil {
			return err
		}
		riwayat, err := punyaRiwayat(r.Context(), tx, o.ID)
		if err != nil {
			return err
		}
		data.KunciFaktor = data.Dasar || (riwayat && !pemilik)

		faktor := faktorLama
		if data.Dasar {
			// The base unit always converts 1:1 and must stay available.
			isian.Faktor = "1"
			isian.Aktif = true
		} else if f, ok := bilanganBulat(isian.Faktor, 2, maksFaktor); !ok {
			errs["faktor_konversi"] = fmt.Sprintf("Isi angka bulat dari 2 sampai 100.000 (jumlah %s dalam satu kemasan).", o.SatuanDasar)
		} else if f != faktorLama && data.KunciFaktor {
			errs["faktor_konversi"] = pesanKunciKonversi
			isian.Faktor = strconv.Itoa(faktorLama)
		} else {
			faktor = f
		}

		harga := hargaLama
		if !pemilik {
			isian.Harga = strconv.Itoa(hargaLama)
		} else if hg, ok := bilanganBulat(isian.Harga, 0, maksHarga); !ok {
			errs["harga_jual"] = "Isi harga jual berupa angka bulat dari 0 sampai 999.999.999."
		} else {
			harga = hg
		}
		if len(errs) > 0 {
			return errValidasi
		}

		_, err = tx.Exec(r.Context(), `
			UPDATE obat_satuan
			SET faktor_konversi = $3, harga_jual = $4, boleh_dijual = $5, aktif = $6
			WHERE id_obat = $1 AND id_obat_satuan = $2`,
			o.ID, kid, faktor, harga, isian.BolehDijual, isian.Aktif)
		return err
	})
	data.Isian = isian
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		http.NotFound(w, r)
	case errors.Is(err, errValidasi):
		h.tampilFormKemasan(w, r, http.StatusUnprocessableEntity, data)
	case err != nil:
		gagal(w, "ubah kemasan", err)
	default:
		http.Redirect(w, r, fmt.Sprintf("/obat/%d?pesan=kemasan-ubah", o.ID), http.StatusSeeOther)
	}
}

var errValidasi = errors.New("validation failed")
