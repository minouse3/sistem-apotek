package penerimaan

import (
	"context"
	"errors"
	"html/template"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/view"
)

func TestReceivingBatchStatusUsesThirtyDayCutoff(t *testing.T) {
	received := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		days   int
		status string
	}{
		{name: "expiry on receipt date", days: 0, status: "HAMPIR_ED"},
		{name: "expiry in thirty days", days: 30, status: "HAMPIR_ED"},
		{name: "expiry after thirty days", days: 31, status: "AKTIF"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expiry := received.AddDate(0, 0, test.days)
			status, err := receivingBatchStatus(expiry, received)
			if err != nil {
				t.Fatalf("receivingBatchStatus() error = %v", err)
			}
			if status != test.status {
				t.Fatalf("receivingBatchStatus() = %q, want %q", status, test.status)
			}
		})
	}
}

func TestReceivingBatchStatusRejectsExpiryBeforeReceipt(t *testing.T) {
	received := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	_, err := receivingBatchStatus(received.AddDate(0, 0, -1), received)

	if !errors.Is(err, ErrExpiryBeforeReceipt) {
		t.Fatalf("receivingBatchStatus() error = %v, want %v", err, ErrExpiryBeforeReceipt)
	}
}

func TestReceivingLineAmountsConvertsToBaseUnits(t *testing.T) {
	baseQuantity, subtotal, err := receivingLineAmounts(3, 1250, 10)
	if err != nil {
		t.Fatalf("receivingLineAmounts() error = %v", err)
	}
	if baseQuantity != 30 || subtotal != 3750 {
		t.Fatalf("amounts = (%d, %d), want (30, 3750)", baseQuantity, subtotal)
	}
}

func TestReceivingLineAmountsRejectsInvalidAndOverflowingValues(t *testing.T) {
	tests := []struct {
		name     string
		quantity int
		price    int
		factor   int
	}{
		{name: "zero quantity", quantity: 0, price: 10, factor: 1},
		{name: "negative price", quantity: 1, price: -1, factor: 1},
		{name: "zero conversion", quantity: 1, price: 10, factor: 0},
		{name: "base quantity over PostgreSQL integer", quantity: math.MaxInt32, price: 0, factor: 2},
		{name: "subtotal over PostgreSQL integer", quantity: math.MaxInt32, price: math.MaxInt32, factor: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := receivingLineAmounts(test.quantity, test.price, test.factor); err == nil {
				t.Fatal("receivingLineAmounts() error = nil, want an error")
			}
		})
	}
}

func TestReceivingRupiahFormattingGroupsThousandsAndKeepsFourDecimals(t *testing.T) {
	if got := formatRupiah(1250000); got != "Rp1.250.000" {
		t.Fatalf("formatRupiah() = %q, want %q", got, "Rp1.250.000")
	}
	if got := formatRupiahDecimal("1250.0000"); got != "Rp1.250,0000" {
		t.Fatalf("formatRupiahDecimal() = %q, want %q", got, "Rp1.250,0000")
	}
}

type memoryReceivingStore struct {
	options               receivingOptions
	invoices              []receivingInvoice
	invoice               receivingInvoice
	lines                 []receivingLine
	financial             ownerInvoiceFinancial
	createErr             error
	cancelErr             error
	ownerTotalsCalled     bool
	ownerFinancialsCalled bool
	createdBy             int
	createdDraft          receivingDraft
	cancelledBy           int
	cancelledInvoice      int
	cancelReason          string
	cancelCalls           int
}

func (s *memoryReceivingStore) Options(context.Context) (receivingOptions, error) {
	return s.options, nil
}

func (s *memoryReceivingStore) List(context.Context) ([]receivingInvoice, error) {
	return s.invoices, nil
}

func (s *memoryReceivingStore) OwnerTotals(context.Context) (map[int]string, error) {
	s.ownerTotalsCalled = true
	return map[int]string{1: "Rp1.200"}, nil
}

func (s *memoryReceivingStore) Invoice(context.Context, int) (receivingInvoice, []receivingLine, error) {
	return s.invoice, s.lines, nil
}

func (s *memoryReceivingStore) OwnerFinancials(context.Context, int) (ownerInvoiceFinancial, error) {
	s.ownerFinancialsCalled = true
	return s.financial, nil
}

func (s *memoryReceivingStore) Create(_ context.Context, userID int, draft receivingDraft) error {
	s.createdBy = userID
	s.createdDraft = draft
	return s.createErr
}

func (s *memoryReceivingStore) Cancel(_ context.Context, userID, invoiceID int, reason string) error {
	s.cancelledBy = userID
	s.cancelledInvoice = invoiceID
	s.cancelReason = reason
	s.cancelCalls++
	return s.cancelErr
}

func testReceivingHandler(t *testing.T, store receivingStore) *receivingHandler {
	t.Helper()
	page, err := template.New("receiving").Parse(`
		{{define "templates/penerimaan/receiving_list.html"}}
		{{if .FormError}}<p>{{.FormError}}</p>{{end}}
		{{if .Form.Errors.Invoice}}<p>{{.Form.Errors.Invoice}}</p>{{end}}
		<p>{{.Form.InvoiceNumber}}</p>
		{{range .Form.Lines}}<p>{{.ExpiryError}}</p>{{end}}
		{{if .IsOwner}}{{range $id, $total := .OwnerTotals}}<p>{{$total}}</p>{{end}}{{end}}
		{{end}}
		{{define "templates/penerimaan/receiving_detail.html"}}
		{{if .IsOwner}}<p>{{.Financial.Total}}</p>{{end}}
		{{if .CancelError}}<p>{{.CancelError}}</p>{{end}}
		{{end}}`)
	if err != nil {
		t.Fatal(err)
	}
	return newReceivingHandler(store, stubRenderer{tmpl: page})
}

func receivingFormValues() url.Values {
	return url.Values{
		"id_pbf":              {"1"},
		"no_faktur":           {"INV-001"},
		"tanggal_faktur":      {"2026-04-01"},
		"tanggal_terima":      {"2026-04-02"},
		"id_obat_satuan":      {"3"},
		"nomor_batch":         {"BATCH-001"},
		"tanggal_kedaluwarsa": {"2026-05-02"},
		"jumlah_faktur":       {"2"},
		"harga_faktur":        {"1250"},
	}
}

func TestParseReceivingFormRejectsExpiryBeforeReceiptInline(t *testing.T) {
	values := receivingFormValues()
	values.Set("tanggal_kedaluwarsa", "2026-04-01")
	request := httptest.NewRequest(http.MethodPost, "/penerimaan", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	form, draft, _, valid, err := parseReceivingForm(httptest.NewRecorder(), request)
	if err != nil {
		t.Fatalf("parseReceivingForm() error = %v", err)
	}
	if valid {
		t.Fatal("parseReceivingForm() valid = true, want false")
	}
	if len(draft.Lines) != 0 {
		t.Fatalf("draft lines = %#v, want none", draft.Lines)
	}
	if form.Lines[0].ExpiryError == "" {
		t.Fatal("expiry validation error is missing")
	}
}

func TestCreateReceivingInvoiceRedirectsWithoutReturningPurchasePrice(t *testing.T) {
	store := &memoryReceivingStore{}
	handler := testReceivingHandler(t, store)
	values := receivingFormValues()
	request := httptest.NewRequest(http.MethodPost, "/penerimaan", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.createForUser(response, request, auth.User{ID: 8, Role: auth.RoleStaf})

	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/penerimaan?status=created&near=1" {
		t.Fatalf("response = %d, Location %q", response.Code, response.Header().Get("Location"))
	}
	if store.createdBy != 8 || len(store.createdDraft.Lines) != 1 || store.createdDraft.Lines[0].InvoicePrice != 1250 {
		t.Fatalf("saved draft = %#v by user %d", store.createdDraft, store.createdBy)
	}
	if strings.Contains(response.Body.String(), "1250") {
		t.Fatalf("purchase price appeared in the save response: %q", response.Body.String())
	}
}

func TestCreateReceivingInvoiceShowsDuplicateNumberError(t *testing.T) {
	store := &memoryReceivingStore{createErr: ErrInvoiceExists}
	handler := testReceivingHandler(t, store)
	values := receivingFormValues()
	request := httptest.NewRequest(http.MethodPost, "/penerimaan", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()

	handler.createForUser(response, request, auth.User{ID: 8, Role: auth.RoleStaf})

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
	for _, expected := range []string{"Nomor faktur sudah terdaftar untuk distributor ini.", "INV-001"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("response missing %q: %s", expected, response.Body.String())
		}
	}
}

func TestOnlyOwnerCanCancelReceivingInvoice(t *testing.T) {
	store := &memoryReceivingStore{}
	handler := testReceivingHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/penerimaan/7/cancel", strings.NewReader("alasan_batal=Koreksi"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetPathValue("id", "7")
	response := httptest.NewRecorder()

	handler.cancelForUser(response, request, auth.User{ID: 8, Role: auth.RoleStaf})

	if response.Code != http.StatusForbidden || store.cancelCalls != 0 {
		t.Fatalf("staff cancellation response = %d, store calls = %d", response.Code, store.cancelCalls)
	}
}

func TestOwnerCancellationTrimsReasonAndRedirects(t *testing.T) {
	store := &memoryReceivingStore{}
	handler := testReceivingHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/penerimaan/7/cancel", strings.NewReader("alasan_batal=+++Koreksi+faktur+++"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetPathValue("id", "7")
	response := httptest.NewRecorder()

	handler.cancelForUser(response, request, auth.User{ID: 1, Role: auth.RolePemilik})

	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/penerimaan/7?status=cancelled" {
		t.Fatalf("owner cancellation response = %d, Location %q", response.Code, response.Header().Get("Location"))
	}
	if store.cancelCalls != 1 || store.cancelledBy != 1 || store.cancelledInvoice != 7 || store.cancelReason != "Koreksi faktur" {
		t.Fatalf("cancel call = %#v", store)
	}
}

func TestOwnerCancellationRequiresReason(t *testing.T) {
	store := &memoryReceivingStore{}
	handler := testReceivingHandler(t, store)
	request := httptest.NewRequest(http.MethodPost, "/penerimaan/7/cancel", strings.NewReader("alasan_batal=+++"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetPathValue("id", "7")
	response := httptest.NewRecorder()

	handler.cancelForUser(response, request, auth.User{ID: 1, Role: auth.RolePemilik})

	if response.Code != http.StatusUnprocessableEntity || store.cancelCalls != 0 {
		t.Fatalf("blank-reason response = %d, store calls = %d", response.Code, store.cancelCalls)
	}
	if !strings.Contains(response.Body.String(), "Alasan pembatalan wajib diisi") {
		t.Fatalf("reason validation missing from %q", response.Body.String())
	}
}

func TestStaffReceivingReadsDoNotLoadFinancialData(t *testing.T) {
	store := &memoryReceivingStore{
		invoices: []receivingInvoice{{ID: 1, InvoiceNumber: "INV-001"}},
		invoice:  receivingInvoice{ID: 1, InvoiceNumber: "INV-001"},
		financial: ownerInvoiceFinancial{
			Total:      1200,
			TotalLabel: "Rp2.500",
			Lines:      map[int]ownerLineFinancial{1: {InvoicePrice: 1200, Subtotal: 1200, BaseUnitCost: "1200.0000"}},
		},
	}
	handler := testReceivingHandler(t, store)
	request := httptest.NewRequest(http.MethodGet, "/penerimaan", nil)
	response := httptest.NewRecorder()
	handler.listForUser(response, request, auth.User{ID: 8, Role: auth.RoleStaf}, defaultReceivingForm(), receivingFormErrors{}, "", http.StatusOK, false)

	detailRequest := httptest.NewRequest(http.MethodGet, "/penerimaan/1", nil)
	detailRequest.SetPathValue("id", "1")
	detailResponse := httptest.NewRecorder()
	handler.detailForUser(detailResponse, detailRequest, auth.User{ID: 8, Role: auth.RoleStaf}, 1, "", "", http.StatusOK)

	if store.ownerTotalsCalled || store.ownerFinancialsCalled {
		t.Fatalf("staff read financial data: totals=%v detail=%v", store.ownerTotalsCalled, store.ownerFinancialsCalled)
	}
	if strings.Contains(response.Body.String(), "1200") || strings.Contains(detailResponse.Body.String(), "1200") {
		t.Fatal("staff response contains owner-only financial values")
	}
}

func TestReceivingTemplatesRenderStaffAndOwnerPages(t *testing.T) {
	renderer, err := view.NewRenderer(
		os.DirFS("../.."),
		"templates/penerimaan/receiving_list.html",
		"templates/penerimaan/receiving_detail.html",
	)
	if err != nil {
		t.Fatal(err)
	}
	store := &memoryReceivingStore{
		options: receivingOptions{
			Distributors: []receivingDistributor{{ID: 2, Name: "PBF Contoh"}},
			Units:        []receivingUnit{{ID: 3, Medicine: "Obat Contoh", Unit: "box", BaseUnit: "tablet", Factor: 10}},
		},
		invoices: []receivingInvoice{{
			ID: 1, Distributor: "PBF Contoh", InvoiceNumber: "INV-001",
			ReceivedDate: "2026-04-02", Status: "DITERIMA", StatusLabel: "Diterima",
			LineCount: 1, BaseQuantity: 20,
		}},
		invoice: receivingInvoice{
			ID: 1, Distributor: "PBF Contoh", InvoiceNumber: "INV-001",
			InvoiceDate: "2026-04-01", ReceivedDate: "2026-04-02",
			Status: "DITERIMA", StatusLabel: "Diterima",
		},
		lines: []receivingLine{{
			DetailID: 4, Medicine: "Obat Contoh", Unit: "box", BaseUnit: "tablet", BatchNumber: "BATCH-1",
			Expiry: "2027-01-01", Quantity: 2, Factor: 10, BaseQuantity: 20,
			BatchStatus: "AKTIF", StatusLabel: "Aktif",
		}},
		financial: ownerInvoiceFinancial{
			Total:      2500,
			TotalLabel: "Rp2.500",
			Lines: map[int]ownerLineFinancial{4: {
				InvoicePrice: 1250, InvoicePriceLabel: "Rp1.250",
				Subtotal: 2500, SubtotalLabel: "Rp2.500",
				BaseUnitCost: "125.0000", BaseUnitCostLabel: "Rp125,0000",
			}},
		},
	}
	handler := newReceivingHandler(store, renderer)

	staffList := httptest.NewRecorder()
	handler.listForUser(staffList, httptest.NewRequest(http.MethodGet, "/penerimaan", nil),
		auth.User{ID: 8, Role: auth.RoleStaf}, defaultReceivingForm(), receivingFormErrors{}, "", http.StatusOK, false)
	if staffList.Code != http.StatusOK || !strings.Contains(staffList.Body.String(), "Penerimaan Barang") {
		t.Fatalf("staff list response = %d, %s", staffList.Code, staffList.Body.String())
	}
	if !strings.Contains(staffList.Body.String(), "/static/js/receiving.js") {
		t.Fatal("staff list missing receiving.js script")
	}
	if strings.Contains(staffList.Body.String(), "Rp2.500") || strings.Contains(staffList.Body.String(), "Total Faktur") {
		t.Fatal("staff list contains owner-only financial fields")
	}

	ownerList := httptest.NewRecorder()
	handler.listForUser(ownerList, httptest.NewRequest(http.MethodGet, "/penerimaan", nil),
		auth.User{ID: 1, Role: auth.RolePemilik}, defaultReceivingForm(), receivingFormErrors{}, "", http.StatusOK, false)
	if !strings.Contains(ownerList.Body.String(), "Total Faktur") || !strings.Contains(ownerList.Body.String(), "Rp1.200") {
		t.Fatal("owner list is missing its financial total")
	}

	staffDetail := httptest.NewRecorder()
	handler.detailForUser(staffDetail, httptest.NewRequest(http.MethodGet, "/penerimaan/1", nil),
		auth.User{ID: 8, Role: auth.RoleStaf}, 1, "", "", http.StatusOK)
	if strings.Contains(staffDetail.Body.String(), "Harga Beli Dasar") ||
		strings.Contains(staffDetail.Body.String(), "Rp1.250") ||
		strings.Contains(staffDetail.Body.String(), "Rp125,0000") {
		t.Fatal("staff detail contains owner-only financial fields")
	}

	ownerDetail := httptest.NewRecorder()
	handler.detailForUser(ownerDetail, httptest.NewRequest(http.MethodGet, "/penerimaan/1", nil),
		auth.User{ID: 1, Role: auth.RolePemilik}, 1, "", "", http.StatusOK)
	if !strings.Contains(ownerDetail.Body.String(), "Harga Beli Dasar") ||
		!strings.Contains(ownerDetail.Body.String(), "Rp125,0000") {
		t.Fatal("owner detail is missing line financial data")
	}
}
