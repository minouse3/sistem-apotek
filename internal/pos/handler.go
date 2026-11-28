package pos

import (
	"encoding/json"
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

type pageRenderer interface {
	Render(w http.ResponseWriter, status int, page string, data any)
}

type Handler struct {
	store    Store
	renderer pageRenderer
}

func NewHandler(pool *pgxpool.Pool, renderer *view.Renderer) *Handler {
	return NewHandlerWithStore(NewPostgresStore(pool), renderer)
}

func NewHandlerWithStore(store Store, renderer pageRenderer) *Handler {
	return &Handler{
		store:    store,
		renderer: renderer,
	}
}

func (h *Handler) Register(mux *http.ServeMux, accounts *auth.Service) {
	posAccess := func(fn http.HandlerFunc) http.Handler {
		return middleware.Require(accounts, fn, auth.RoleKasir, auth.RolePemilik)
	}

	mux.Handle("GET /pos", posAccess(h.kasirPage))
	mux.Handle("GET /pos/produk", posAccess(h.searchProducts))
	mux.Handle("POST /pos/preview", posAccess(h.previewCart))
	mux.Handle("POST /pos/checkout", posAccess(h.checkout))
	mux.Handle("GET /pos/riwayat", posAccess(h.receiptListPage))
	mux.Handle("GET /pos/struk/{id}", posAccess(h.receiptDetailPage))
	mux.Handle("POST /pos/struk/{id}/void", posAccess(h.voidReceipt))
}

var posNotices = map[string]string{
	"checkout-sukses": "Transaksi berhasil disimpan dan stok batch telah diperbarui.",
	"void-sukses":     "Transaksi dibatalkan dan stok batch telah dikembalikan.",
}

func (h *Handler) kasirPage(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r.Context())
	products, err := h.store.ListSellableProducts(r.Context(), "")
	loadError := ""
	if err != nil {
		log.Printf("pos kasir list products: %v", err)
		loadError = "Gagal memuat daftar obat layak jual. Muat ulang halaman."
	}

	h.renderer.Render(w, http.StatusOK, "templates/pos/kasir.html", struct {
		view.PageMeta
		Products  []SellableProduct
		LoadError string
	}{
		PageMeta:  view.NewPageMeta(user, "Kasir (POS)", "pos"),
		Products:  products,
		LoadError: loadError,
	})
}

func (h *Handler) searchProducts(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	products, err := h.store.ListSellableProducts(r.Context(), q)
	if err != nil {
		log.Printf("pos search products: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Gagal mencari obat."})
		return
	}
	if products == nil {
		products = []SellableProduct{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

func (h *Handler) previewCart(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	var req struct {
		Items []CartItemInput `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Format keranjang tidak valid."})
		return
	}

	preview, err := h.store.PreviewCart(r.Context(), req.Items)
	if err != nil {
		status, msg := mapPOSDomainError(err)
		if status == http.StatusInternalServerError {
			log.Printf("pos preview: %v", err)
		}
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (h *Handler) checkout(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)

	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Data pembayaran tidak valid."})
		return
	}

	saleID, err := h.store.Checkout(r.Context(), user, req)
	if err != nil {
		status, msg := mapPOSDomainError(err)
		if status == http.StatusInternalServerError {
			log.Printf("pos checkout: %v", err)
		}
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id_penjualan": saleID,
		"redirect":     fmt.Sprintf("/pos/struk/%d?pesan=checkout-sukses", saleID),
	})
}

func (h *Handler) receiptListPage(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r.Context())
	isOwner := user.Role == auth.RolePemilik

	filter := ReceiptFilter{
		Keyword: strings.TrimSpace(r.URL.Query().Get("q")),
		Tanggal: strings.TrimSpace(r.URL.Query().Get("tanggal")),
		Status:  strings.TrimSpace(r.URL.Query().Get("status")),
	}
	filterError := ""
	if filter.Tanggal != "" {
		if _, err := time.Parse("2006-01-02", filter.Tanggal); err != nil {
			filterError = "Format tanggal tidak valid."
			filter.Tanggal = ""
		}
	}
	if filter.Status != "" && filter.Status != "SELESAI" && filter.Status != "DIBATALKAN" {
		filterError = "Filter status tidak dikenal."
		filter.Status = ""
	}

	receipts, ownerHPP, err := h.store.ListReceipts(r.Context(), filter, isOwner)
	if err != nil {
		log.Printf("pos list receipts: %v", err)
		filterError = "Gagal memuat riwayat transaksi. Coba muat ulang."
	}

	h.renderer.Render(w, http.StatusOK, "templates/pos/riwayat.html", struct {
		view.PageMeta
		Filter      ReceiptFilter
		Receipts    []ReceiptSummary
		OwnerHPP    map[int]int
		IsOwner     bool
		FilterError string
	}{
		PageMeta:    view.NewPageMeta(user, "Riwayat Transaksi", "pos-riwayat"),
		Filter:      filter,
		Receipts:    receipts,
		OwnerHPP:    ownerHPP,
		IsOwner:     isOwner,
		FilterError: filterError,
	})
}

func (h *Handler) receiptDetailPage(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}

	isOwner := user.Role == auth.RolePemilik
	receipt, finance, err := h.store.GetReceipt(r.Context(), id, isOwner)
	if errors.Is(err, ErrReceiptNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("pos get receipt %d: %v", id, err)
		http.Error(w, "Gagal memuat struk transaksi.", http.StatusInternalServerError)
		return
	}

	h.renderReceiptDetail(w, http.StatusOK, user, receipt, finance, posNotices[r.URL.Query().Get("pesan")], "", "")
}

func (h *Handler) voidReceipt(w http.ResponseWriter, r *http.Request) {
	user, _ := middleware.CurrentUser(r.Context())
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.NotFound(w, r)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Formulir tidak dapat dibaca.", http.StatusBadRequest)
		return
	}

	reason := strings.TrimSpace(r.PostForm.Get("alasan_batal"))
	isOwner := user.Role == auth.RolePemilik

	err = h.store.VoidReceipt(r.Context(), id, user, reason)
	if err == nil {
		http.Redirect(w, r, fmt.Sprintf("/pos/struk/%d?pesan=void-sukses", id), http.StatusSeeOther)
		return
	}
	if errors.Is(err, ErrReceiptNotFound) {
		http.NotFound(w, r)
		return
	}

	status, msg := mapPOSDomainError(err)
	if status == http.StatusInternalServerError {
		log.Printf("pos void receipt %d: %v", id, err)
	}

	receipt, finance, loadErr := h.store.GetReceipt(r.Context(), id, isOwner)
	if loadErr != nil {
		http.Error(w, msg, status)
		return
	}
	h.renderReceiptDetail(w, status, user, receipt, finance, "", msg, reason)
}

func (h *Handler) renderReceiptDetail(
	w http.ResponseWriter,
	status int,
	user auth.User,
	receipt ReceiptDetail,
	finance *OwnerReceiptFinance,
	notice string,
	voidError string,
	voidReason string,
) {
	h.renderer.Render(w, status, "templates/pos/struk.html", struct {
		view.PageMeta
		Receipt    ReceiptDetail
		Finance    *OwnerReceiptFinance
		IsOwner    bool
		Notice     string
		VoidError  string
		VoidReason string
	}{
		PageMeta:   view.NewPageMeta(user, "Struk "+receipt.NomorTransaksi, "pos-riwayat"),
		Receipt:    receipt,
		Finance:    finance,
		IsOwner:    user.Role == auth.RolePemilik,
		Notice:     notice,
		VoidError:  voidError,
		VoidReason: voidReason,
	})
}

func mapPOSDomainError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrEmptyCart):
		return http.StatusBadRequest, "Tambahkan minimal satu obat ke keranjang."
	case errors.Is(err, ErrTooManyItems):
		return http.StatusBadRequest, "Jumlah baris keranjang melebihi batas maksimal."
	case errors.Is(err, ErrInvalidItem):
		return http.StatusBadRequest, "Obat atau satuan kemasan tidak tersedia untuk dijual."
	case errors.Is(err, ErrInvalidPayment):
		return http.StatusBadRequest, "Pilih metode pembayaran Tunai, QRIS, atau Debit."
	case errors.Is(err, ErrInsufficientPay):
		return http.StatusUnprocessableEntity, "Jumlah bayar tunai kurang dari total belanja."
	case errors.Is(err, ErrInsufficientStock):
		return http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, ErrReasonRequired):
		return http.StatusBadRequest, "Masukkan alasan pembatalan maksimal 1.000 karakter."
	case errors.Is(err, ErrAlreadyCanceled):
		return http.StatusConflict, "Transaksi ini sudah dibatalkan sebelumnya."
	case errors.Is(err, ErrHasActiveReturn):
		return http.StatusConflict, "Transaksi tidak dapat dibatalkan karena sudah memiliki retur aktif atau disetujui."
	case errors.Is(err, ErrVoidWindowExpired):
		return http.StatusForbidden, "Batas waktu void kasir (5 menit pada hari yang sama) sudah lewat. Hubungi pemilik."
	default:
		return http.StatusInternalServerError, "Terjadi kesalahan pada server. Coba lagi."
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode pos json: %v", err)
	}
}
