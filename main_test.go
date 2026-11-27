package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/laporan"
	"github.com/minouse3/sistem-apotek/internal/obat"
	"github.com/minouse3/sistem-apotek/internal/view"
)

func TestStaticAssetHandler(t *testing.T) {
	staticFiles, err := fs.Sub(appFiles, "static")
	if err != nil {
		t.Fatalf("sub static: %v", err)
	}
	handler := staticAssetHandler(staticFiles)

	allowed := []string{
		"/static/css/custom.css",
		"/static/js/app.js",
		"/static/vendor/bootstrap/bootstrap.min.css",
		"/static/vendor/bootstrap/bootstrap.bundle.min.js",
		"/static/vendor/phosphor/icons.svg",
		"/static/vendor/roboto/roboto-latin-wght-normal.woff2",
	}
	for _, target := range allowed {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want 200", target, rec.Code)
		}
	}

	blocked := []string{
		"/static/",
		"/static/css/",
		"/static/css/.gitkeep",
		"/static/vendor/bootstrap/LICENSE",
		"/static/../main.go",
	}
	for _, target := range blocked {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Errorf("GET %s = %d, want 404", target, rec.Code)
		}
	}
}

func TestTemplatesRenderAndEnforceRoleVisibility(t *testing.T) {
	renderer, err := view.NewRenderer(appFiles, appTemplates...)
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	kasir := auth.User{ID: 2, Name: "Nadya Kasir", Role: auth.RoleKasir}
	staf := auth.User{ID: 3, Name: "Fachreza Staf", Role: auth.RoleStaf}
	pemilik := auth.User{ID: 1, Name: "Pemilik Demo", Role: auth.RolePemilik}

	t.Run("home omits finance and owner-only nav for kasir and keeps mobile logout and active nav", func(t *testing.T) {
		rec := httptest.NewRecorder()
		renderer.Render(rec, http.StatusOK, "templates/home.html", struct {
			view.PageMeta
			Dashboard laporan.OperationalDashboard
			Finance   *laporan.OwnerFinanceSummary
		}{
			PageMeta: view.NewPageMeta(kasir, "Beranda", "dashboard"),
			Finance:  nil,
		})
		body := rec.Body.String()
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if strings.Contains(body, "Laporan Laba Kotor") || strings.Contains(body, "Keuangan Bulan Ini") ||
			strings.Contains(body, "/satuan") || strings.Contains(body, "/karyawan") {
			t.Fatal("kasir dashboard leaked restricted sections or nav links")
		}
		if !strings.Contains(body, "/obat") || !strings.Contains(body, "/akun") {
			t.Fatal("kasir dashboard missing allowed master data links")
		}
		if strings.Count(body, `aria-current="page"`) != 2 {
			t.Fatalf("expected active nav state in both desktop sidebar and mobile drawer, got %d", strings.Count(body, `aria-current="page"`))
		}
		if strings.Count(body, `action="/logout"`) != 2 {
			t.Fatalf("expected logout form in both desktop sidebar and mobile drawer, got %d", strings.Count(body, `action="/logout"`))
		}
		if strings.Contains(body, "—") {
			t.Fatal("rendered HTML contains forbidden em dash")
		}
	})

	t.Run("stok shows finance only for pemilik", func(t *testing.T) {
		recKasir := httptest.NewRecorder()
		renderer.Render(recKasir, http.StatusOK, "templates/laporan/stok.html", struct {
			view.PageMeta
			Filter         laporan.StockFilter
			Zones          laporan.ZoneSummary
			Racks          []string
			Batches        []laporan.BatchStock
			OwnerBatches   []laporan.OwnerBatchStock
			TotalNilaiAset int64
			IsOwner        bool
			FilterError    string
		}{
			PageMeta: view.NewPageMeta(kasir, "Laporan Stok Batch", "stok"),
			IsOwner:  false,
		})
		if strings.Contains(recKasir.Body.String(), "Harga Beli") || strings.Contains(recKasir.Body.String(), "export.csv") {
			t.Fatal("kasir stok page leaked owner-only finance or CSV export")
		}

		recOwner := httptest.NewRecorder()
		renderer.Render(recOwner, http.StatusOK, "templates/laporan/stok.html", struct {
			view.PageMeta
			Filter         laporan.StockFilter
			Zones          laporan.ZoneSummary
			Racks          []string
			Batches        []laporan.BatchStock
			OwnerBatches   []laporan.OwnerBatchStock
			TotalNilaiAset int64
			IsOwner        bool
			FilterError    string
		}{
			PageMeta:       view.NewPageMeta(pemilik, "Laporan Stok Batch", "stok"),
			IsOwner:        true,
			TotalNilaiAset: 2500000,
		})
		body := recOwner.Body.String()
		if !strings.Contains(body, "Harga Beli") || !strings.Contains(body, "Rp2.500.000") {
			t.Fatal("owner stok page missing expected financial valuation")
		}
	})

	t.Run("laporan pages render cleanly", func(t *testing.T) {
		recED := httptest.NewRecorder()
		renderer.Render(recED, http.StatusOK, "templates/laporan/hampir_ed.html", struct {
			view.PageMeta
			Zona        string
			Zones       laporan.ZoneSummary
			Batches     []laporan.ExpiringBatch
			FilterError string
		}{
			PageMeta: view.NewPageMeta(kasir, "Batch Hampir ED", "hampir-ed"),
		})
		if recED.Code != http.StatusOK || strings.Contains(recED.Body.String(), "—") {
			t.Fatalf("hampir_ed render failed or contains em dash (status %d)", recED.Code)
		}

		recAudit := httptest.NewRecorder()
		renderer.Render(recAudit, http.StatusOK, "templates/laporan/batch_audit.html", struct {
			view.PageMeta
			Audit   laporan.BatchAudit
			Finance *laporan.OwnerBatchFinance
		}{
			PageMeta: view.NewPageMeta(pemilik, "Jejak Batch PCM-01", "stok"),
			Audit:    laporan.BatchAudit{NomorBatch: "PCM-01", NamaObat: "Paracetamol", SatuanDasar: "tablet", BadgeClass: "sw-badge-sage"},
			Finance:  &laporan.OwnerBatchFinance{HargaBeliDasar: 180.25, NilaiAset: 18025},
		})
		if recAudit.Code != http.StatusOK || !strings.Contains(recAudit.Body.String(), "Rp180,2500") {
			t.Fatalf("batch_audit render failed (status %d)", recAudit.Code)
		}

		recLaba := httptest.NewRecorder()
		renderer.Render(recLaba, http.StatusOK, "templates/laporan/laba_kotor.html", struct {
			view.PageMeta
			TanggalMulai string
			TanggalAkhir string
			Keyword      string
			FilterError  string
			Summary      laporan.GrossProfitReport
			Transactions []laporan.GrossProfitTransaction
		}{
			PageMeta:     view.NewPageMeta(pemilik, "Laporan Laba Kotor", "laba-kotor"),
			TanggalMulai: "2026-11-01",
			TanggalAkhir: "2026-11-27",
			Summary:      laporan.GrossProfitReport{PenjualanBersih: 1200000, HPPBersih: 800000, LabaKotor: 400000, MarginPersen: 33.3, NilaiAsetStok: 5400000},
		})
		body := recLaba.Body.String()
		if recLaba.Code != http.StatusOK || !strings.Contains(body, "Rp400.000") || !strings.Contains(body, "Rp5.400.000") || strings.Contains(body, "—") {
			t.Fatalf("laba_kotor render failed (status %d)", recLaba.Code)
		}
	})

	t.Run("obat and karyawan pages render cleanly in shared shell", func(t *testing.T) {
		now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
		stafPage := obat.Halaman{PageMeta: view.NewPageMeta(staf, "Data Obat", "obat"), User: staf}
		ownerPage := obat.Halaman{PageMeta: view.NewPageMeta(pemilik, "Akun Karyawan", "karyawan"), User: pemilik}

		recDaftar := httptest.NewRecorder()
		renderer.Render(recDaftar, http.StatusOK, "templates/obat/daftar.html", struct {
			obat.Halaman
			Q      string
			Status string
			Daftar []struct {
				ID            int
				Kode          string
				Nama          string
				Kandungan     string
				SatuanDasar   string
				Rak           string
				Aktif         bool
				JumlahKemasan int
			}
		}{
			Halaman: stafPage,
			Status:  "aktif",
		})
		if recDaftar.Code != http.StatusOK || strings.Contains(recDaftar.Body.String(), "—") {
			t.Fatalf("obat/daftar render failed (status %d)", recDaftar.Code)
		}

		recDetail := httptest.NewRecorder()
		renderer.Render(recDetail, http.StatusOK, "templates/obat/detail.html", struct {
			obat.Halaman
			Obat struct {
				ID            int
				Kode          string
				Nama          string
				Kandungan     string
				SatuanDasar   string
				IDDasar       int
				Rak           string
				Aktif         bool
				StokLayak     int
				BatchTerdekat string
				EDTerdekat    string
			}
			Kemasan []struct {
				ID          int
				IDSatuan    int
				NamaSatuan  string
				Faktor      int
				Harga       int
				BolehDijual bool
				Aktif       bool
				Dasar       bool
			}
			PunyaRiwayat bool
		}{
			Halaman:      stafPage,
			PunyaRiwayat: true,
		})
		if recDetail.Code != http.StatusOK || !strings.Contains(recDetail.Body.String(), "Faktor konversi terkunci") {
			t.Fatalf("obat/detail render failed (status %d)", recDetail.Code)
		}

		for _, page := range []string{
			"templates/obat/form.html",
			"templates/obat/kemasan.html",
			"templates/obat/satuan.html",
			"templates/obat/satuan_ubah.html",
			"templates/karyawan/daftar.html",
			"templates/karyawan/form.html",
			"templates/karyawan/akun.html",
		} {
			rec := httptest.NewRecorder()
			renderer.Render(rec, http.StatusOK, page, map[string]any{
				"Title":            ownerPage.Title,
				"Active":           ownerPage.Active,
				"Role":             ownerPage.Role,
				"RoleLabel":        ownerPage.RoleLabel,
				"UserName":         ownerPage.UserName,
				"Initial":          ownerPage.Initial,
				"User":             pemilik,
				"Pesan":            "",
				"Pemilik":          true,
				"BolehKelolaObat":  true,
				"Baru":             true,
				"Aksi":             "/test",
				"ID":               1,
				"Nama":             "tablet",
				"Error":            "",
				"Errors":           map[string]string{},
				"Ditolak":          "",
				"Dasar":            false,
				"KunciFaktor":      false,
				"KunciSatuanDasar": false,
				"AkunSendiri":      false,
				"Obat":             map[string]any{"ID": 1, "Nama": "Paracetamol", "SatuanDasar": "tablet", "IDDasar": 1},
				"Isian":            map[string]any{"Kode": "PCT-500", "Nama": "Paracetamol", "Kandungan": "", "Rak": "RAK-A1", "SatuanDasarID": 1, "HargaDasar": "500", "IDSatuan": 2, "NamaSatuan": "strip", "Faktor": "10", "Harga": "5000", "BolehDijual": true, "Aktif": true, "Username": "kasir1", "Email": "", "IDRole": 1},
				"Akun":             map[string]any{"ID": 1, "Nama": "Pemilik Demo", "Username": "pemilik", "Email": "", "Role": auth.RolePemilik, "LastLogin": &now},
				"Daftar":           nil,
				"Satuan":           nil,
				"Roles":            nil,
			})
			if rec.Code != http.StatusOK || strings.Contains(rec.Body.String(), "—") {
				t.Fatalf("render %s failed (status %d)", page, rec.Code)
			}
		}
	})
}
