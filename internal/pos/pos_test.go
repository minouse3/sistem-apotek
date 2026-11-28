package pos

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
)

func TestAllocateFEFO_WholePackagePrefersLaterBatch(t *testing.T) {
	lines := []CartLineRequest{
		{
			IDObatSatuan:   1,
			IDObat:         10,
			NamaObat:       "Paracetamol",
			NamaSatuan:     "Strip",
			SatuanDasar:    "Tablet",
			Jumlah:         1,
			FaktorKonversi: 10,
			HargaJual:      12000,
		},
	}
	batches := []CandidateBatch{
		{IDBatch: 1, IDObat: 10, NomorBatch: "B-EARLY", ExpiryDate: "10-05-2027", StokTersisa: 6, CostTenThousand: 8000000},
		{IDBatch: 2, IDObat: 10, NomorBatch: "B-LATER", ExpiryDate: "10-08-2027", StokTersisa: 12, CostTenThousand: 8500000},
	}

	allocs, err := AllocateFEFO(lines, batches)
	if err != nil {
		t.Fatalf("AllocateFEFO failed: %v", err)
	}
	if len(allocs) != 1 || len(allocs[0].Batches) != 1 {
		t.Fatalf("expected single batch allocation, got %+v", allocs)
	}
	got := allocs[0].Batches[0]
	if got.NomorBatch != "B-LATER" || got.JumlahDasar != 10 || got.UrutanFEFO != 1 {
		t.Fatalf("unexpected allocation: %+v", got)
	}
	if got.HPP != 8500 {
		t.Fatalf("got HPP %d, want 8500", got.HPP)
	}
}

func TestAllocateFEFO_SplitsRemainderInExpiryOrder(t *testing.T) {
	lines := []CartLineRequest{
		{
			IDObatSatuan:   1,
			IDObat:         10,
			NamaObat:       "Paracetamol",
			NamaSatuan:     "Strip",
			SatuanDasar:    "Tablet",
			Jumlah:         1,
			FaktorKonversi: 10,
			HargaJual:      12000,
		},
	}
	batches := []CandidateBatch{
		{IDBatch: 1, IDObat: 10, NomorBatch: "B-EARLY", ExpiryDate: "10-05-2027", StokTersisa: 6, CostTenThousand: 8000000},
		{IDBatch: 2, IDObat: 10, NomorBatch: "B-LATER", ExpiryDate: "10-08-2027", StokTersisa: 4, CostTenThousand: 8500000},
	}

	allocs, err := AllocateFEFO(lines, batches)
	if err != nil {
		t.Fatalf("AllocateFEFO failed: %v", err)
	}
	if len(allocs[0].Batches) != 2 {
		t.Fatalf("expected 2 split batches, got %d", len(allocs[0].Batches))
	}
	first := allocs[0].Batches[0]
	second := allocs[0].Batches[1]
	if first.NomorBatch != "B-EARLY" || first.JumlahDasar != 6 || first.UrutanFEFO != 1 {
		t.Fatalf("unexpected first batch: %+v", first)
	}
	if second.NomorBatch != "B-LATER" || second.JumlahDasar != 4 || second.UrutanFEFO != 2 {
		t.Fatalf("unexpected second batch: %+v", second)
	}
}

func TestAllocateFEFO_SharedMedicineAcrossCartLines(t *testing.T) {
	lines := []CartLineRequest{
		{IDObatSatuan: 1, IDObat: 10, NamaObat: "Amoxicillin", Jumlah: 1, FaktorKonversi: 10, HargaJual: 20000},
		{IDObatSatuan: 2, IDObat: 10, NamaObat: "Amoxicillin", Jumlah: 3, FaktorKonversi: 1, HargaJual: 2500},
	}
	batches := []CandidateBatch{
		{IDBatch: 1, IDObat: 10, NomorBatch: "B-1", StokTersisa: 12, CostTenThousand: 15000000},
	}

	_, err := AllocateFEFO(lines, batches)
	if !errors.Is(err, ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock when combined lines exceed stock, got %v", err)
	}
}

func TestWithinKasirVoidWindow(t *testing.T) {
	sale := time.Date(2026, 11, 27, 23, 57, 0, 0, wibLocation)

	if !withinKasirVoidWindow(sale, sale.Add(2*time.Minute)) {
		t.Fatal("expected void within 2 minutes on same day to be allowed")
	}
	if withinKasirVoidWindow(sale, sale.Add(5*time.Minute+time.Second)) {
		t.Fatal("expected void after 5 minutes to be rejected")
	}
	if withinKasirVoidWindow(sale, sale.Add(4*time.Minute)) {
		t.Fatal("expected void across midnight WIB calendar day to be rejected")
	}
}

func TestCostConversionAndHPP(t *testing.T) {
	cost, err := ParseCostTenThousandths("1234.5678")
	if err != nil {
		t.Fatalf("ParseCostTenThousandths: %v", err)
	}
	if cost != 12345678 {
		t.Fatalf("got %d, want 12345678", cost)
	}
	if got := FormatCostTenThousandths(cost); got != "1234.5678" {
		t.Fatalf("got %q, want 1234.5678", got)
	}
	if hpp := ComputeHPP(3, 1005000); hpp != 302 {
		t.Fatalf("got HPP %d, want 302", hpp)
	}
}

type fakeStore struct {
	products    []SellableProduct
	preview     CartPreview
	previewErr  error
	checkoutID  int
	checkoutErr error
	receipt     ReceiptDetail
	finance     *OwnerReceiptFinance
	voidErr     error
}

func (f *fakeStore) ListSellableProducts(context.Context, string) ([]SellableProduct, error) {
	return f.products, nil
}

func (f *fakeStore) PreviewCart(context.Context, []CartItemInput) (CartPreview, error) {
	return f.preview, f.previewErr
}

func (f *fakeStore) Checkout(context.Context, auth.User, CheckoutRequest) (int, error) {
	return f.checkoutID, f.checkoutErr
}

func (f *fakeStore) ListReceipts(context.Context, ReceiptFilter, bool) ([]ReceiptSummary, map[int]int, error) {
	return nil, nil, nil
}

func (f *fakeStore) GetReceipt(context.Context, int, bool) (ReceiptDetail, *OwnerReceiptFinance, error) {
	return f.receipt, f.finance, nil
}

func (f *fakeStore) VoidReceipt(context.Context, int, auth.User, string) error {
	return f.voidErr
}

type noopRenderer struct {
	lastStatus int
	lastPage   string
}

func (n *noopRenderer) Render(w http.ResponseWriter, status int, page string, _ any) {
	n.lastStatus = status
	n.lastPage = page
	w.WriteHeader(status)
}

func TestHandlerCheckoutAndVoidErrors(t *testing.T) {
	store := &fakeStore{
		checkoutID: 42,
		receipt:    ReceiptDetail{IDPenjualan: 42, NomorTransaksi: "TRX-20261127-00042", Status: "SELESAI"},
	}
	renderer := &noopRenderer{}
	h := NewHandlerWithStore(store, renderer)

	t.Run("checkout returns created redirect", func(t *testing.T) {
		body, _ := json.Marshal(CheckoutRequest{
			MetodePembayaran: "TUNAI",
			JumlahBayar:      20000,
			Items:            []CartItemInput{{IDObatSatuan: 1, Jumlah: 1}},
		})
		req := httptest.NewRequest(http.MethodPost, "/pos/checkout", bytes.NewReader(body))
		rec := httptest.NewRecorder()
		h.checkout(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "/pos/struk/42") {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})

	t.Run("void returns 409 when active return exists", func(t *testing.T) {
		store.voidErr = ErrHasActiveReturn
		form := url.Values{"alasan_batal": {"Salah input"}}
		req := httptest.NewRequest(http.MethodPost, "/pos/struk/42/void", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "42")
		rec := httptest.NewRecorder()
		h.voidReceipt(rec, req)

		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409", rec.Code)
		}
	})

	t.Run("void returns 403 when kasir window expired", func(t *testing.T) {
		store.voidErr = ErrVoidWindowExpired
		form := url.Values{"alasan_batal": {"Telat"}}
		req := httptest.NewRequest(http.MethodPost, "/pos/struk/42/void", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.SetPathValue("id", "42")
		rec := httptest.NewRecorder()
		h.voidReceipt(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	})

	_ = middleware.CurrentUser
}
