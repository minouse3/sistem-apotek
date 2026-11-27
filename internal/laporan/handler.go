package laporan

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
	"github.com/minouse3/sistem-apotek/internal/view"
)

type Handler struct {
	pool     *pgxpool.Pool
	renderer *view.Renderer
}

func NewHandler(pool *pgxpool.Pool, renderer *view.Renderer) *Handler {
	return &Handler{pool: pool, renderer: renderer}
}

func (h *Handler) Register(mux *http.ServeMux, accounts *auth.Service) {
	mux.Handle("GET /{$}", middleware.Require(accounts, http.HandlerFunc(h.Dashboard)))
	mux.Handle("GET /laporan/stok", middleware.Require(accounts, http.HandlerFunc(h.BatchStock)))
	mux.Handle("GET /laporan/stok/export.csv", middleware.Require(accounts, http.HandlerFunc(h.ExportBatchStockCSV), auth.RolePemilik))
	mux.Handle("GET /laporan/hampir-ed", middleware.Require(accounts, http.HandlerFunc(h.ExpiringBatches)))
	mux.Handle("GET /laporan/batch/{id}", middleware.Require(accounts, http.HandlerFunc(h.BatchAudit)))
	mux.Handle("GET /laporan/laba-kotor", middleware.Require(accounts, http.HandlerFunc(h.GrossProfit), auth.RolePemilik))
	mux.Handle("GET /laporan/laba-kotor/export.csv", middleware.Require(accounts, http.HandlerFunc(h.ExportGrossProfitCSV), auth.RolePemilik))
}

type dashboardPageData struct {
	view.PageMeta
	Dashboard OperationalDashboard
	Finance   *OwnerFinanceSummary
}

func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Sesi pengguna tidak ditemukan.", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ops, err := GetOperationalDashboard(ctx, h.pool)
	if err != nil {
		log.Printf("get operational dashboard: %v", err)
		http.Error(w, "Gagal memuat ringkasan beranda. Coba lagi.", http.StatusInternalServerError)
		return
	}

	var finance *OwnerFinanceSummary
	if user.Role == auth.RolePemilik {
		summary, err := GetOwnerFinanceSummary(ctx, h.pool)
		if err != nil {
			log.Printf("get owner finance summary: %v", err)
			http.Error(w, "Gagal memuat ringkasan keuangan. Coba lagi.", http.StatusInternalServerError)
			return
		}
		finance = &summary
	}

	h.renderer.Render(w, http.StatusOK, "templates/home.html", dashboardPageData{
		PageMeta:  view.NewPageMeta(user, "Beranda", "dashboard"),
		Dashboard: ops,
		Finance:   finance,
	})
}

type batchStockPageData struct {
	view.PageMeta
	Filter         StockFilter
	Zones          ZoneSummary
	Racks          []string
	Batches        []BatchStock
	OwnerBatches   []OwnerBatchStock
	TotalNilaiAset int64
	IsOwner        bool
	FilterError    string
}

func parseStockFilter(r *http.Request) (StockFilter, string) {
	filter := StockFilter{
		Keyword: strings.TrimSpace(r.URL.Query().Get("q")),
		Zona:    strings.TrimSpace(r.URL.Query().Get("zona")),
		Status:  strings.TrimSpace(r.URL.Query().Get("status")),
		Rak:     strings.TrimSpace(r.URL.Query().Get("rak")),
	}

	switch filter.Zona {
	case "", ZoneMerah, ZoneKuning, ZoneHijau, ZoneHabis:
	default:
		filter.Zona = ""
		return filter, "Filter zona ED tidak dikenali. Pilih Merah, Kuning, Hijau, atau Habis."
	}

	switch filter.Status {
	case "", "AKTIF", "HAMPIR_ED", "KEDALUWARSA", "HABIS":
	default:
		filter.Status = ""
		return filter, "Filter status batch tidak valid."
	}

	return filter, ""
}

func (h *Handler) BatchStock(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Sesi pengguna tidak ditemukan.", http.StatusInternalServerError)
		return
	}

	filter, filterErr := parseStockFilter(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	zones, err := GetZoneSummary(ctx, h.pool)
	if err != nil {
		log.Printf("get zone summary: %v", err)
		http.Error(w, "Gagal memuat ringkasan zona ED.", http.StatusInternalServerError)
		return
	}

	racks, err := GetRackLocations(ctx, h.pool)
	if err != nil {
		log.Printf("get rack locations: %v", err)
		http.Error(w, "Gagal memuat daftar lokasi rak.", http.StatusInternalServerError)
		return
	}

	data := batchStockPageData{
		PageMeta:    view.NewPageMeta(user, "Laporan Stok Batch", "stok"),
		Filter:      filter,
		Zones:       zones,
		Racks:       racks,
		IsOwner:     user.Role == auth.RolePemilik,
		FilterError: filterErr,
	}

	statusCode := http.StatusOK
	if filterErr != "" {
		statusCode = http.StatusBadRequest
	} else if data.IsOwner {
		ownerRows, totalAsset, err := GetOwnerBatchStock(ctx, h.pool, filter)
		if err != nil {
			log.Printf("get owner batch stock: %v", err)
			http.Error(w, "Gagal memuat laporan stok batch.", http.StatusInternalServerError)
			return
		}
		data.OwnerBatches = ownerRows
		data.TotalNilaiAset = totalAsset
	} else {
		rows, err := GetBatchStock(ctx, h.pool, filter)
		if err != nil {
			log.Printf("get batch stock: %v", err)
			http.Error(w, "Gagal memuat laporan stok batch.", http.StatusInternalServerError)
			return
		}
		data.Batches = rows
	}

	h.renderer.Render(w, statusCode, "templates/laporan/stok.html", data)
}

func (h *Handler) ExportBatchStockCSV(w http.ResponseWriter, r *http.Request) {
	filter, filterErr := parseStockFilter(r)
	if filterErr != "" {
		http.Error(w, filterErr, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, _, err := GetOwnerBatchStock(ctx, h.pool, filter)
	if err != nil {
		log.Printf("export batch stock csv: %v", err)
		http.Error(w, "Gagal mengekspor laporan stok.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="laporan-stok-batch.csv"`)
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"Kode Obat",
		"Nama Obat",
		"Nomor Batch",
		"Lokasi Rak",
		"Tanggal Kedaluwarsa",
		"Zona ED",
		"Status",
		"Stok Tersisa",
		"Satuan Dasar",
		"Harga Beli Dasar",
		"Nilai Aset",
	})
	for _, b := range rows {
		_ = writer.Write([]string{
			safeCSVCell(b.KodeObat),
			safeCSVCell(b.NamaObat),
			safeCSVCell(b.NomorBatch),
			safeCSVCell(b.LokasiRak),
			b.TanggalKedaluwarsa,
			b.ZonaED,
			b.Status,
			strconv.Itoa(b.Stok),
			safeCSVCell(b.SatuanDasar),
			strconv.FormatFloat(b.HargaBeliDasar, 'f', 4, 64),
			strconv.FormatInt(b.NilaiAset, 10),
		})
	}
	writer.Flush()
}

type expiryPageData struct {
	view.PageMeta
	Zona        string
	Zones       ZoneSummary
	Batches     []ExpiringBatch
	FilterError string
}

func (h *Handler) ExpiringBatches(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Sesi pengguna tidak ditemukan.", http.StatusInternalServerError)
		return
	}

	zona := strings.TrimSpace(r.URL.Query().Get("zona"))
	var filterErr string
	switch zona {
	case "", "SEMUA", ZoneMerah, ZoneKuning, ZoneHijau:
	default:
		zona = ""
		filterErr = "Pilihan zona ED tidak valid. Gunakan Merah, Kuning, Hijau, atau Semua."
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	zones, err := GetZoneSummary(ctx, h.pool)
	if err != nil {
		log.Printf("get expiry zone summary: %v", err)
		http.Error(w, "Gagal memuat ringkasan zona ED.", http.StatusInternalServerError)
		return
	}

	data := expiryPageData{
		PageMeta:    view.NewPageMeta(user, "Zonasi & Batch Hampir ED", "hampir-ed"),
		Zona:        zona,
		Zones:       zones,
		FilterError: filterErr,
	}

	statusCode := http.StatusOK
	if filterErr != "" {
		statusCode = http.StatusBadRequest
	} else {
		batches, err := GetExpiringBatches(ctx, h.pool, zona)
		if err != nil {
			log.Printf("get expiring batches: %v", err)
			http.Error(w, "Gagal memuat peringatan kedaluwarsa.", http.StatusInternalServerError)
			return
		}
		data.Batches = batches
	}

	h.renderer.Render(w, statusCode, "templates/laporan/hampir_ed.html", data)
}

type batchAuditPageData struct {
	view.PageMeta
	Audit   BatchAudit
	Finance *OwnerBatchFinance
}

func (h *Handler) BatchAudit(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Sesi pengguna tidak ditemukan.", http.StatusInternalServerError)
		return
	}

	idBatch, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || idBatch <= 0 {
		http.Error(w, "ID batch tidak valid.", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	audit, err := GetBatchAudit(ctx, h.pool, idBatch)
	if errors.Is(err, ErrBatchNotFound) {
		http.Error(w, "Batch tidak ditemukan.", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("get batch audit: %v", err)
		http.Error(w, "Gagal memuat jejak mutasi batch.", http.StatusInternalServerError)
		return
	}

	var finance *OwnerBatchFinance
	if user.Role == auth.RolePemilik {
		fin, err := GetOwnerBatchFinance(ctx, h.pool, idBatch)
		if err != nil {
			log.Printf("get owner batch finance: %v", err)
			http.Error(w, "Gagal memuat data finansial batch.", http.StatusInternalServerError)
			return
		}
		finance = &fin
	}

	h.renderer.Render(w, http.StatusOK, "templates/laporan/batch_audit.html", batchAuditPageData{
		PageMeta: view.NewPageMeta(user, "Jejak Batch "+audit.NomorBatch, "stok"),
		Audit:    audit,
		Finance:  finance,
	})
}

type grossProfitPageData struct {
	view.PageMeta
	TanggalMulai string
	TanggalAkhir string
	Keyword      string
	FilterError  string
	Summary      GrossProfitReport
	Transactions []GrossProfitTransaction
}

func parseDateRange(r *http.Request, now time.Time) (string, string, string, string) {
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	mulai := strings.TrimSpace(r.URL.Query().Get("mulai"))
	akhir := strings.TrimSpace(r.URL.Query().Get("akhir"))
	keyword := strings.TrimSpace(r.URL.Query().Get("q"))

	if mulai == "" {
		mulai = firstDay.Format("2006-01-02")
	}
	if akhir == "" {
		akhir = now.Format("2006-01-02")
	}

	tMulai, err := time.Parse("2006-01-02", mulai)
	if err != nil {
		return mulai, akhir, keyword, "Format tanggal mulai tidak valid. Gunakan format YYYY-MM-DD."
	}
	tAkhir, err := time.Parse("2006-01-02", akhir)
	if err != nil {
		return mulai, akhir, keyword, "Format tanggal akhir tidak valid. Gunakan format YYYY-MM-DD."
	}
	if tMulai.After(tAkhir) {
		return mulai, akhir, keyword, "Tanggal mulai tidak boleh melebihi tanggal akhir."
	}

	return mulai, akhir, keyword, ""
}

func (h *Handler) GrossProfit(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Sesi pengguna tidak ditemukan.", http.StatusInternalServerError)
		return
	}

	mulai, akhir, keyword, filterErr := parseDateRange(r, time.Now())

	data := grossProfitPageData{
		PageMeta:     view.NewPageMeta(user, "Laporan Laba Kotor", "laba-kotor"),
		TanggalMulai: mulai,
		TanggalAkhir: akhir,
		Keyword:      keyword,
		FilterError:  filterErr,
	}

	if filterErr != "" {
		h.renderer.Render(w, http.StatusBadRequest, "templates/laporan/laba_kotor.html", data)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	summary, err := GetGrossProfit(ctx, h.pool, mulai, akhir, keyword)
	if err != nil {
		log.Printf("get gross profit: %v", err)
		http.Error(w, "Gagal memuat ringkasan laba kotor.", http.StatusInternalServerError)
		return
	}

	transactions, err := GetGrossProfitTransactions(ctx, h.pool, mulai, akhir, keyword)
	if err != nil {
		log.Printf("get gross profit transactions: %v", err)
		http.Error(w, "Gagal memuat rincian transaksi.", http.StatusInternalServerError)
		return
	}

	data.Summary = summary
	data.Transactions = transactions
	h.renderer.Render(w, http.StatusOK, "templates/laporan/laba_kotor.html", data)
}

func (h *Handler) ExportGrossProfitCSV(w http.ResponseWriter, r *http.Request) {
	mulai, akhir, keyword, filterErr := parseDateRange(r, time.Now())
	if filterErr != "" {
		http.Error(w, filterErr, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	transactions, err := GetGrossProfitTransactions(ctx, h.pool, mulai, akhir, keyword)
	if err != nil {
		log.Printf("export gross profit csv: %v", err)
		http.Error(w, "Gagal mengekspor laporan laba kotor.", http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("laporan-laba-kotor-%s-sd-%s.csv", mulai, akhir)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")

	writer := csv.NewWriter(w)
	_ = writer.Write([]string{
		"Nomor Transaksi",
		"Tanggal",
		"Kasir",
		"Metode Pembayaran",
		"Penjualan",
		"HPP",
		"Pengembalian Retur",
		"HPP Retur Masuk",
		"Laba Kotor",
	})
	for _, tx := range transactions {
		_ = writer.Write([]string{
			safeCSVCell(tx.NomorTransaksi),
			tx.TanggalTransaksi,
			safeCSVCell(tx.NamaKasir),
			safeCSVCell(tx.MetodePembayaran),
			strconv.FormatInt(tx.TotalPenjualan, 10),
			strconv.FormatInt(tx.TotalHPP, 10),
			strconv.FormatInt(tx.TotalPengembalian, 10),
			strconv.FormatInt(tx.HPPReturMasuk, 10),
			strconv.FormatInt(tx.LabaKotor, 10),
		})
	}
	writer.Flush()
}
