package penerimaan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
)

const (
	maxReceivingLines = 50
	maxReceivingBytes = 1 << 20
	maxPostgresInt    = int64(math.MaxInt32)
)

var (
	ErrExpiryBeforeReceipt = errors.New("batch expiry is before receipt date")
	errInvalidLineAmounts  = errors.New("invalid or overflowing receiving line amounts")
	ErrInvoiceExists       = errors.New("receiving invoice already exists")
	ErrSupplierUnavailable = errors.New("receiving distributor is unavailable")
	ErrUnitUnavailable     = errors.New("receiving medicine unit is unavailable")
	ErrInvoiceNotFound     = errors.New("receiving invoice not found")
	ErrInvoiceCanceled     = errors.New("receiving invoice is already canceled")
	ErrInvoiceInUse        = errors.New("receiving invoice batches have been used")
)

type receivingDraft struct {
	DistributorID int
	InvoiceNumber string
	InvoiceDate   time.Time
	ReceivedDate  time.Time
	Lines         []receivingLineDraft
}

type receivingLineDraft struct {
	UnitID       int
	BatchNumber  string
	Expiry       time.Time
	Quantity     int
	InvoicePrice int
}

type receivingDistributor struct {
	ID   int
	Name string
}

type receivingUnit struct {
	ID         int
	MedicineID int
	Medicine   string
	Unit       string
	BaseUnit   string
	Factor     int
}

type receivingOptions struct {
	Distributors []receivingDistributor
	Units        []receivingUnit
}

type receivingInvoice struct {
	ID            int
	Distributor   string
	InvoiceNumber string
	InvoiceDate   string
	ReceivedDate  string
	Status        string
	StatusLabel   string
	LineCount     int
	BaseQuantity  int64
	CanCancel     bool
}

type receivingLine struct {
	DetailID     int
	Medicine     string
	Unit         string
	BaseUnit     string
	BatchNumber  string
	Expiry       string
	Quantity     int
	Factor       int
	BaseQuantity int
	BatchStatus  string
	StatusLabel  string
}

type ownerLineFinancial struct {
	InvoicePrice      int
	InvoicePriceLabel string
	Subtotal          int
	SubtotalLabel     string
	BaseUnitCost      string
	BaseUnitCostLabel string
}

type ownerInvoiceFinancial struct {
	Total      int
	TotalLabel string
	Lines      map[int]ownerLineFinancial
}

type receivingStore interface {
	Options(context.Context) (receivingOptions, error)
	List(context.Context) ([]receivingInvoice, error)
	OwnerTotals(context.Context) (map[int]string, error)
	Invoice(context.Context, int) (receivingInvoice, []receivingLine, error)
	OwnerFinancials(context.Context, int) (ownerInvoiceFinancial, error)
	Create(context.Context, int, receivingDraft) error
	Cancel(context.Context, int, int, string) error
}

type postgresReceivingStore struct {
	pool *pgxpool.Pool
}

type receivingHandler struct {
	store receivingStore
	page  *template.Template
}

type receivingFormLine struct {
	UnitID        string
	BatchNumber   string
	Expiry        string
	Quantity      string
	InvoicePrice  string
	UnitError     string
	BatchError    string
	ExpiryError   string
	QuantityError string
	PriceError    string
}

type receivingFormErrors struct {
	Distributor  string
	Invoice      string
	InvoiceDate  string
	ReceivedDate string
}

type receivingForm struct {
	DistributorID string
	InvoiceNumber string
	InvoiceDate   string
	ReceivedDate  string
	Lines         []receivingFormLine
	Errors        receivingFormErrors
}

type receivingListPage struct {
	IsOwner      bool
	Distributors []receivingDistributor
	Units        []receivingUnit
	Invoices     []receivingInvoice
	OwnerTotals  map[int]string
	Form         receivingForm
	Empty        bool
	ListError    string
	OptionsError string
	FormError    string
	Notice       string
	NearExpiry   bool
}

type receivingDetailPage struct {
	IsOwner      bool
	Invoice      receivingInvoice
	Lines        []receivingLine
	Financial    ownerInvoiceFinancial
	CancelError  string
	CancelReason string
	Notice       string
}

type receivingLineStoreError struct {
	Index int
	Err   error
}

func (e receivingLineStoreError) Error() string {
	return e.Err.Error()
}

func (e receivingLineStoreError) Unwrap() error {
	return e.Err
}

func NewReceiving(pool *pgxpool.Pool, page *template.Template) *ReceivingHandler {
	return &ReceivingHandler{handler: newReceivingHandler(postgresReceivingStore{pool: pool}, page)}
}

type ReceivingHandler struct {
	handler *receivingHandler
}

func (h *ReceivingHandler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /penerimaan", h.handler.list)
	mux.HandleFunc("GET /penerimaan/{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/penerimaan", http.StatusSeeOther)
	})
	mux.HandleFunc("POST /penerimaan", h.handler.create)
	mux.HandleFunc("GET /penerimaan/{id}", h.handler.detail)
	return mux
}

func (h *ReceivingHandler) CancelHandler() http.Handler {
	return http.HandlerFunc(h.handler.cancel)
}

func newReceivingHandler(store receivingStore, page *template.Template) *receivingHandler {
	return &receivingHandler{store: store, page: page}
}

func receivingBatchStatus(expiry, received time.Time) (string, error) {
	expiry = dateOnly(expiry)
	received = dateOnly(received)
	if expiry.Before(received) {
		return "", ErrExpiryBeforeReceipt
	}
	if !expiry.After(received.AddDate(0, 0, 30)) {
		return "HAMPIR_ED", nil
	}
	return "AKTIF", nil
}

func receivingLineAmounts(quantity, invoicePrice, factor int) (int64, int64, error) {
	if quantity < 1 || invoicePrice < 0 || factor < 1 {
		return 0, 0, errInvalidLineAmounts
	}
	quantityValue := int64(quantity)
	priceValue := int64(invoicePrice)
	factorValue := int64(factor)
	if quantityValue > maxPostgresInt/factorValue || quantityValue > maxPostgresInt || priceValue > maxPostgresInt {
		return 0, 0, errInvalidLineAmounts
	}
	baseQuantity := quantityValue * factorValue
	if priceValue != 0 && quantityValue > maxPostgresInt/priceValue {
		return 0, 0, errInvalidLineAmounts
	}
	return baseQuantity, quantityValue * priceValue, nil
}

func (s postgresReceivingStore) Options(ctx context.Context) (receivingOptions, error) {
	options := receivingOptions{
		Distributors: make([]receivingDistributor, 0),
		Units:        make([]receivingUnit, 0),
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id_pbf, nama_pbf
		FROM pbf
		WHERE aktif
		ORDER BY nama_pbf, id_pbf`)
	if err != nil {
		return receivingOptions{}, err
	}
	for rows.Next() {
		var distributor receivingDistributor
		if err := rows.Scan(&distributor.ID, &distributor.Name); err != nil {
			rows.Close()
			return receivingOptions{}, err
		}
		options.Distributors = append(options.Distributors, distributor)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return receivingOptions{}, err
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT os.id_obat_satuan, o.id_obat, o.nama_obat, s.nama_satuan,
			base.nama_satuan, os.faktor_konversi
		FROM obat_satuan os
		JOIN obat o ON o.id_obat = os.id_obat
		JOIN satuan s ON s.id_satuan = os.id_satuan
		JOIN satuan base ON base.id_satuan = o.satuan_dasar_id
		WHERE o.status_aktif AND os.aktif
		ORDER BY o.nama_obat, s.nama_satuan, os.id_obat_satuan`)
	if err != nil {
		return receivingOptions{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var unit receivingUnit
		if err := rows.Scan(&unit.ID, &unit.MedicineID, &unit.Medicine, &unit.Unit, &unit.BaseUnit, &unit.Factor); err != nil {
			return receivingOptions{}, err
		}
		options.Units = append(options.Units, unit)
	}
	if err := rows.Err(); err != nil {
		return receivingOptions{}, err
	}
	return options, nil
}

func (s postgresReceivingStore) List(ctx context.Context) ([]receivingInvoice, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.id_penerimaan, f.nama_pbf, p.no_faktur, p.tanggal_faktur, p.tanggal_terima,
			p.status, COUNT(d.id_detail_penerimaan)::int,
			COALESCE(SUM(d.jumlah_dasar), 0)::bigint
		FROM penerimaan p
		JOIN pbf f ON f.id_pbf = p.id_pbf
		LEFT JOIN detail_penerimaan d ON d.id_penerimaan = p.id_penerimaan
		GROUP BY p.id_penerimaan, f.nama_pbf
		ORDER BY p.tanggal_terima DESC, p.id_penerimaan DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invoices := make([]receivingInvoice, 0)
	for rows.Next() {
		var invoice receivingInvoice
		var invoiceDate, receivedDate time.Time
		if err := rows.Scan(
			&invoice.ID,
			&invoice.Distributor,
			&invoice.InvoiceNumber,
			&invoiceDate,
			&receivedDate,
			&invoice.Status,
			&invoice.LineCount,
			&invoice.BaseQuantity,
		); err != nil {
			return nil, err
		}
		invoice.InvoiceDate = invoiceDate.Format(time.DateOnly)
		invoice.ReceivedDate = receivedDate.Format(time.DateOnly)
		invoice.StatusLabel = receivingStatusLabel(invoice.Status)
		invoices = append(invoices, invoice)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invoices, nil
}

func (s postgresReceivingStore) OwnerTotals(ctx context.Context) (map[int]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id_penerimaan, total_faktur
		FROM penerimaan
		ORDER BY id_penerimaan`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	totals := make(map[int]string)
	for rows.Next() {
		var invoiceID, total int
		if err := rows.Scan(&invoiceID, &total); err != nil {
			return nil, err
		}
		totals[invoiceID] = formatRupiah(total)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return totals, nil
}

func (s postgresReceivingStore) Invoice(ctx context.Context, id int) (receivingInvoice, []receivingLine, error) {
	var invoice receivingInvoice
	var invoiceDate, receivedDate time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT p.id_penerimaan, f.nama_pbf, p.no_faktur, p.tanggal_faktur,
			p.tanggal_terima, p.status,
			(
				p.status = 'DITERIMA'
				AND NOT EXISTS (
					SELECT 1
					FROM detail_penerimaan d
					JOIN batch b ON b.id_batch = d.id_batch
					WHERE d.id_penerimaan = p.id_penerimaan
						AND (
							b.stok_tersisa <> d.jumlah_dasar
							OR EXISTS (
								SELECT 1 FROM detail_penjualan ds
								WHERE ds.id_batch = b.id_batch
							)
							OR EXISTS (
								SELECT 1 FROM detail_stock_opname so
								WHERE so.id_batch = b.id_batch
							)
							OR EXISTS (
								SELECT 1 FROM mutasi_stok m
								WHERE m.id_batch = b.id_batch
									AND m.jenis_mutasi <> 'PENERIMAAN'
							)
						)
				)
			)
		FROM penerimaan p
		JOIN pbf f ON f.id_pbf = p.id_pbf
		WHERE p.id_penerimaan = $1`, id).Scan(
		&invoice.ID,
		&invoice.Distributor,
		&invoice.InvoiceNumber,
		&invoiceDate,
		&receivedDate,
		&invoice.Status,
		&invoice.CanCancel,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return receivingInvoice{}, nil, ErrInvoiceNotFound
	}
	if err != nil {
		return receivingInvoice{}, nil, err
	}
	invoice.InvoiceDate = invoiceDate.Format(time.DateOnly)
	invoice.ReceivedDate = receivedDate.Format(time.DateOnly)
	invoice.StatusLabel = receivingStatusLabel(invoice.Status)

	rows, err := s.pool.Query(ctx, `
		SELECT d.id_detail_penerimaan, o.nama_obat, u.nama_satuan, base.nama_satuan, b.nomor_batch,
			b.tanggal_kedaluwarsa, d.jumlah_faktur, d.faktor_konversi,
			d.jumlah_dasar, b.status_batch
		FROM detail_penerimaan d
		JOIN obat_satuan os ON os.id_obat_satuan = d.id_obat_satuan
		JOIN obat o ON o.id_obat = os.id_obat
		JOIN satuan u ON u.id_satuan = os.id_satuan
		JOIN satuan base ON base.id_satuan = o.satuan_dasar_id
		JOIN batch b ON b.id_batch = d.id_batch
		WHERE d.id_penerimaan = $1
		ORDER BY d.id_detail_penerimaan`, id)
	if err != nil {
		return receivingInvoice{}, nil, err
	}
	defer rows.Close()

	lines := make([]receivingLine, 0)
	for rows.Next() {
		var line receivingLine
		var expiry time.Time
		if err := rows.Scan(
			&line.DetailID,
			&line.Medicine,
			&line.Unit,
			&line.BaseUnit,
			&line.BatchNumber,
			&expiry,
			&line.Quantity,
			&line.Factor,
			&line.BaseQuantity,
			&line.BatchStatus,
		); err != nil {
			return receivingInvoice{}, nil, err
		}
		line.Expiry = expiry.Format(time.DateOnly)
		line.StatusLabel = receivingStatusLabel(line.BatchStatus)
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return receivingInvoice{}, nil, err
	}
	invoice.LineCount = len(lines)
	return invoice, lines, nil
}

func (s postgresReceivingStore) OwnerFinancials(ctx context.Context, id int) (ownerInvoiceFinancial, error) {
	financial := ownerInvoiceFinancial{Lines: make(map[int]ownerLineFinancial)}
	rows, err := s.pool.Query(ctx, `
		SELECT p.total_faktur, d.id_detail_penerimaan, d.harga_faktur,
			d.subtotal, b.harga_beli_dasar::text
		FROM penerimaan p
		LEFT JOIN detail_penerimaan d ON d.id_penerimaan = p.id_penerimaan
		LEFT JOIN batch b ON b.id_batch = d.id_batch
		WHERE p.id_penerimaan = $1
		ORDER BY d.id_detail_penerimaan`, id)
	if err != nil {
		return ownerInvoiceFinancial{}, err
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		found = true
		var detailID *int
		var invoicePrice, subtotal *int
		var baseUnitCost *string
		if err := rows.Scan(&financial.Total, &detailID, &invoicePrice, &subtotal, &baseUnitCost); err != nil {
			return ownerInvoiceFinancial{}, err
		}
		if detailID != nil && invoicePrice != nil && subtotal != nil && baseUnitCost != nil {
			financial.Lines[*detailID] = ownerLineFinancial{
				InvoicePrice:      *invoicePrice,
				InvoicePriceLabel: formatRupiah(*invoicePrice),
				Subtotal:          *subtotal,
				SubtotalLabel:     formatRupiah(*subtotal),
				BaseUnitCost:      *baseUnitCost,
				BaseUnitCostLabel: formatRupiahDecimal(*baseUnitCost),
			}
		}
	}
	if err := rows.Err(); err != nil {
		return ownerInvoiceFinancial{}, err
	}
	if !found {
		return ownerInvoiceFinancial{}, ErrInvoiceNotFound
	}
	financial.TotalLabel = formatRupiah(financial.Total)
	return financial, nil
}

func (s postgresReceivingStore) Create(ctx context.Context, userID int, draft receivingDraft) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var activeDistributor int
	err = tx.QueryRow(ctx, `
		SELECT id_pbf
		FROM pbf
		WHERE id_pbf = $1 AND aktif
		FOR SHARE`, draft.DistributorID).Scan(&activeDistributor)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSupplierUnavailable
	}
	if err != nil {
		return err
	}

	type preparedLine struct {
		receivingLineDraft
		MedicineID int
		Factor     int
		BaseAmount int64
		Subtotal   int64
		Status     string
	}
	prepared := make([]preparedLine, 0, len(draft.Lines))
	total := int64(0)
	for i, line := range draft.Lines {
		var medicineID, factor int
		err := tx.QueryRow(ctx, `
			SELECT os.id_obat, os.faktor_konversi
			FROM obat_satuan os
			JOIN obat o ON o.id_obat = os.id_obat
			WHERE os.id_obat_satuan = $1 AND os.aktif AND o.status_aktif
			FOR SHARE OF os, o`, line.UnitID).Scan(&medicineID, &factor)
		if errors.Is(err, pgx.ErrNoRows) {
			return receivingLineStoreError{Index: i, Err: ErrUnitUnavailable}
		}
		if err != nil {
			return err
		}
		baseQuantity, subtotal, err := receivingLineAmounts(line.Quantity, line.InvoicePrice, factor)
		if err != nil {
			return receivingLineStoreError{Index: i, Err: err}
		}
		status, err := receivingBatchStatus(line.Expiry, draft.ReceivedDate)
		if err != nil {
			return receivingLineStoreError{Index: i, Err: err}
		}
		if total > maxPostgresInt-subtotal {
			return errInvalidLineAmounts
		}
		total += subtotal
		prepared = append(prepared, preparedLine{
			receivingLineDraft: line,
			MedicineID:         medicineID,
			Factor:             factor,
			BaseAmount:         baseQuantity,
			Subtotal:           subtotal,
			Status:             status,
		})
	}

	var invoiceID int
	err = tx.QueryRow(ctx, `
		INSERT INTO penerimaan
			(id_pbf, no_faktur, tanggal_faktur, tanggal_terima, id_pengguna, total_faktur)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id_penerimaan`,
		draft.DistributorID,
		draft.InvoiceNumber,
		draft.InvoiceDate,
		draft.ReceivedDate,
		userID,
		int(total),
	).Scan(&invoiceID)
	if err != nil {
		return receivingWriteError(err)
	}

	for _, line := range prepared {
		var batchID int
		err := tx.QueryRow(ctx, `
			INSERT INTO batch
				(id_obat, nomor_batch, tanggal_kedaluwarsa, harga_beli_dasar,
					stok_tersisa, status_batch)
			VALUES ($1, $2, $3, ROUND($4::numeric / $5, 4), $6, $7)
			RETURNING id_batch`,
			line.MedicineID,
			line.BatchNumber,
			line.Expiry,
			line.InvoicePrice,
			line.Factor,
			line.BaseAmount,
			line.Status,
		).Scan(&batchID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO detail_penerimaan
				(id_penerimaan, id_batch, id_obat_satuan, jumlah_faktur, harga_faktur,
					faktor_konversi, jumlah_dasar, subtotal)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			invoiceID,
			batchID,
			line.UnitID,
			line.Quantity,
			line.InvoicePrice,
			line.Factor,
			line.BaseAmount,
			line.Subtotal,
		)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO mutasi_stok
				(id_batch, jenis_mutasi, jumlah_dasar, jenis_referensi, id_referensi, id_pengguna)
			VALUES ($1, 'PENERIMAAN', $2, 'penerimaan', $3, $4)`,
			batchID,
			line.BaseAmount,
			invoiceID,
			userID,
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s postgresReceivingStore) Cancel(ctx context.Context, userID, invoiceID int, reason string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx, `
		SELECT status
		FROM penerimaan
		WHERE id_penerimaan = $1
		FOR UPDATE`, invoiceID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvoiceNotFound
	}
	if err != nil {
		return err
	}
	if status == "DIBATALKAN" {
		return ErrInvoiceCanceled
	}

	type batchToCancel struct {
		ID       int
		Stock    int
		Received int
	}
	rows, err := tx.Query(ctx, `
		SELECT b.id_batch, b.stok_tersisa, d.jumlah_dasar
		FROM detail_penerimaan d
		JOIN batch b ON b.id_batch = d.id_batch
		WHERE d.id_penerimaan = $1
		ORDER BY b.id_batch
		FOR UPDATE OF b`, invoiceID)
	if err != nil {
		return err
	}
	batches := make([]batchToCancel, 0)
	for rows.Next() {
		var batch batchToCancel
		if err := rows.Scan(&batch.ID, &batch.Stock, &batch.Received); err != nil {
			rows.Close()
			return err
		}
		batches = append(batches, batch)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(batches) == 0 {
		return ErrInvoiceInUse
	}

	var used bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM detail_penerimaan d
			JOIN batch b ON b.id_batch = d.id_batch
			WHERE d.id_penerimaan = $1
				AND (
					b.stok_tersisa <> d.jumlah_dasar
					OR EXISTS (
						SELECT 1 FROM detail_penjualan ds
						WHERE ds.id_batch = b.id_batch
					)
					OR EXISTS (
						SELECT 1 FROM detail_stock_opname so
						WHERE so.id_batch = b.id_batch
					)
					OR EXISTS (
						SELECT 1 FROM mutasi_stok m
						WHERE m.id_batch = b.id_batch
							AND m.jenis_mutasi <> 'PENERIMAAN'
					)
				)
		)`, invoiceID).Scan(&used)
	if err != nil {
		return err
	}
	if used {
		return ErrInvoiceInUse
	}

	for _, batch := range batches {
		_, err := tx.Exec(ctx, `
			INSERT INTO mutasi_stok
				(id_batch, jenis_mutasi, jumlah_dasar, jenis_referensi, id_referensi, id_pengguna)
			VALUES ($1, 'PEMBATALAN_PENERIMAAN', $2, 'penerimaan', $3, $4)`,
			batch.ID,
			-batch.Received,
			invoiceID,
			userID,
		)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `
			UPDATE batch SET stok_tersisa = 0, status_batch = 'HABIS'
			WHERE id_batch = $1`, batch.ID)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `
		UPDATE penerimaan
		SET status = 'DIBATALKAN', alasan_batal = $2, waktu_batal = now(), id_pengguna_batal = $3
		WHERE id_penerimaan = $1`, invoiceID, reason, userID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func receivingWriteError(err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) &&
		postgresError.Code == "23505" &&
		postgresError.ConstraintName == "penerimaan_id_pbf_no_faktur_key" {
		return ErrInvoiceExists
	}
	return err
}

func (h *receivingHandler) list(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Akses ditolak", http.StatusUnauthorized)
		return
	}
	h.listForUser(
		w,
		r,
		user,
		defaultReceivingForm(),
		receivingFormErrors{},
		"",
		http.StatusOK,
		r.URL.Query().Get("near") == "1",
	)
}

func (h *receivingHandler) listForUser(
	w http.ResponseWriter,
	r *http.Request,
	user auth.User,
	form receivingForm,
	formErrors receivingFormErrors,
	formError string,
	status int,
	nearExpiry bool,
) {
	data := receivingListPage{
		IsOwner:    user.Role == auth.RolePemilik,
		Form:       form,
		FormError:  formError,
		NearExpiry: nearExpiry,
		Notice:     receivingNotice(r.URL.Query().Get("status")),
	}
	if len(data.Form.Lines) == 0 {
		data.Form = defaultReceivingForm()
	}
	data.Form.Errors = formErrors

	options, err := h.store.Options(r.Context())
	if err != nil {
		log.Printf("load receiving options: %v", err)
		data.OptionsError = "Tidak dapat memuat distributor dan obat untuk formulir penerimaan."
		status = http.StatusInternalServerError
	} else {
		data.Distributors = options.Distributors
		data.Units = options.Units
	}

	invoices, err := h.store.List(r.Context())
	if err != nil {
		log.Printf("list receiving invoices: %v", err)
		data.ListError = "Tidak dapat memuat daftar faktur penerimaan."
		status = http.StatusInternalServerError
	} else {
		data.Invoices = invoices
		data.Empty = len(invoices) == 0
	}
	if data.IsOwner && data.ListError == "" {
		totals, err := h.store.OwnerTotals(r.Context())
		if err != nil {
			log.Printf("load owner receiving totals: %v", err)
			data.ListError = "Tidak dapat memuat daftar faktur penerimaan."
			status = http.StatusInternalServerError
		} else {
			data.OwnerTotals = totals
		}
	}
	h.render(w, "receiving-list.html", data, status)
}

func (h *receivingHandler) create(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Akses ditolak", http.StatusUnauthorized)
		return
	}
	h.createForUser(w, r, user)
}

func (h *receivingHandler) createForUser(w http.ResponseWriter, r *http.Request, user auth.User) {
	form, draft, formErrors, valid, err := parseReceivingForm(w, r)
	if err != nil {
		h.listForUser(w, r, user, form, formErrors, "Formulir tidak dapat dibaca. Isi kembali lalu coba lagi.", http.StatusBadRequest, false)
		return
	}
	if !valid {
		h.listForUser(w, r, user, form, formErrors, "", http.StatusUnprocessableEntity, false)
		return
	}
	if err := h.store.Create(r.Context(), user.ID, draft); err != nil {
		switch {
		case errors.Is(err, ErrInvoiceExists):
			formErrors.Invoice = "Nomor faktur sudah terdaftar untuk distributor ini."
			h.listForUser(w, r, user, form, formErrors, "", http.StatusConflict, false)
		case errors.Is(err, ErrSupplierUnavailable):
			formErrors.Distributor = "Pilih distributor yang masih aktif."
			h.listForUser(w, r, user, form, formErrors, "", http.StatusUnprocessableEntity, false)
		case errors.Is(err, ErrExpiryBeforeReceipt):
			var lineError receivingLineStoreError
			if errors.As(err, &lineError) {
				setReceivingLineFieldError(&form, &lineError, "expiry", "Tanggal kedaluwarsa tidak boleh sebelum tanggal terima.")
			}
			h.listForUser(w, r, user, form, formErrors, "", http.StatusUnprocessableEntity, false)
		case errors.Is(err, ErrUnitUnavailable):
			var lineError receivingLineStoreError
			if errors.As(err, &lineError) {
				setReceivingLineFieldError(&form, &lineError, "unit", "Pilih obat dan satuan yang masih aktif.")
			}
			h.listForUser(w, r, user, form, formErrors, "", http.StatusUnprocessableEntity, false)
		case errors.Is(err, errInvalidLineAmounts):
			var lineError receivingLineStoreError
			if errors.As(err, &lineError) {
				setReceivingLineFieldError(&form, &lineError, "quantity", "Jumlah setelah konversi melebihi batas penyimpanan.")
			}
			h.listForUser(w, r, user, form, formErrors, "", http.StatusUnprocessableEntity, false)
		default:
			log.Printf("create receiving invoice: %v", err)
			h.listForUser(w, r, user, form, formErrors, "Tidak dapat menyimpan faktur penerimaan. Coba lagi.", http.StatusInternalServerError, false)
		}
		return
	}

	nearExpiry := false
	for _, line := range draft.Lines {
		status, err := receivingBatchStatus(line.Expiry, draft.ReceivedDate)
		if err == nil && status == "HAMPIR_ED" {
			nearExpiry = true
			break
		}
	}
	location := "/penerimaan?status=created"
	if nearExpiry {
		location += "&near=1"
	}
	http.Redirect(w, r, location, http.StatusSeeOther)
}

func (h *receivingHandler) detail(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Akses ditolak", http.StatusUnauthorized)
		return
	}
	id, err := parseReceivingID(r)
	if err != nil {
		http.Error(w, "ID faktur tidak valid.", http.StatusBadRequest)
		return
	}
	h.detailForUser(w, r, user, id, "", "", http.StatusOK)
}

func (h *receivingHandler) detailForUser(
	w http.ResponseWriter,
	r *http.Request,
	user auth.User,
	id int,
	cancelError string,
	cancelReason string,
	status int,
) {
	invoice, lines, err := h.store.Invoice(r.Context(), id)
	if errors.Is(err, ErrInvoiceNotFound) {
		http.Error(w, "Faktur penerimaan tidak ditemukan.", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("load receiving invoice %d: %v", id, err)
		http.Error(w, "Tidak dapat memuat faktur penerimaan.", http.StatusInternalServerError)
		return
	}
	data := receivingDetailPage{
		IsOwner:      user.Role == auth.RolePemilik,
		Invoice:      invoice,
		Lines:        lines,
		CancelError:  cancelError,
		CancelReason: cancelReason,
		Notice:       receivingNotice(r.URL.Query().Get("status")),
	}
	if data.IsOwner {
		financial, err := h.store.OwnerFinancials(r.Context(), id)
		if err != nil {
			log.Printf("load owner receiving financials for invoice %d: %v", id, err)
			http.Error(w, "Tidak dapat memuat rincian keuangan faktur.", http.StatusInternalServerError)
			return
		}
		data.Financial = financial
	}
	h.render(w, "receiving-detail.html", data, status)
}

func (h *receivingHandler) cancel(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r.Context())
	if !ok {
		http.Error(w, "Akses ditolak", http.StatusUnauthorized)
		return
	}
	h.cancelForUser(w, r, user)
}

func (h *receivingHandler) cancelForUser(w http.ResponseWriter, r *http.Request, user auth.User) {
	if user.Role != auth.RolePemilik {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
		return
	}
	id, err := parseReceivingID(r)
	if err != nil {
		http.Error(w, "ID faktur tidak valid.", http.StatusBadRequest)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
	if err := r.ParseForm(); err != nil {
		h.detailForUser(w, r, user, id, "Formulir pembatalan tidak dapat dibaca. Coba lagi.", "", http.StatusBadRequest)
		return
	}
	reason := strings.TrimSpace(r.PostForm.Get("alasan_batal"))
	if reason == "" || !utf8.ValidString(reason) || utf8.RuneCountInString(reason) > 1000 || strings.ContainsRune(reason, 0) {
		h.detailForUser(w, r, user, id, "Alasan pembatalan wajib diisi dan maksimal 1.000 karakter.", reason, http.StatusUnprocessableEntity)
		return
	}
	if err := h.store.Cancel(r.Context(), user.ID, id, reason); err != nil {
		switch {
		case errors.Is(err, ErrInvoiceNotFound):
			http.Error(w, "Faktur penerimaan tidak ditemukan.", http.StatusNotFound)
		case errors.Is(err, ErrInvoiceCanceled):
			h.detailForUser(w, r, user, id, "Faktur ini sudah dibatalkan.", reason, http.StatusConflict)
		case errors.Is(err, ErrInvoiceInUse):
			h.detailForUser(w, r, user, id, "Faktur tidak dapat dibatalkan karena batch telah digunakan atau stoknya berubah.", reason, http.StatusConflict)
		default:
			log.Printf("cancel receiving invoice %d: %v", id, err)
			h.detailForUser(w, r, user, id, "Tidak dapat membatalkan faktur penerimaan. Coba lagi.", reason, http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/penerimaan/%d?status=cancelled", id), http.StatusSeeOther)
}

func parseReceivingForm(w http.ResponseWriter, r *http.Request) (receivingForm, receivingDraft, receivingFormErrors, bool, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReceivingBytes)
	if err := r.ParseForm(); err != nil {
		return receivingForm{}, receivingDraft{}, receivingFormErrors{}, false, err
	}
	form := receivingForm{
		DistributorID: r.PostForm.Get("id_pbf"),
		InvoiceNumber: strings.TrimSpace(r.PostForm.Get("no_faktur")),
		InvoiceDate:   r.PostForm.Get("tanggal_faktur"),
		ReceivedDate:  r.PostForm.Get("tanggal_terima"),
	}
	values := []string{
		"id_obat_satuan",
		"nomor_batch",
		"tanggal_kedaluwarsa",
		"jumlah_faktur",
		"harga_faktur",
	}
	count := len(r.PostForm[values[0]])
	if count < 1 || count > maxReceivingLines {
		form.Lines = []receivingFormLine{{}}
		return form, receivingDraft{}, receivingFormErrors{}, false, fmt.Errorf("receiving line count must be between 1 and %d", maxReceivingLines)
	}
	for _, name := range values[1:] {
		if len(r.PostForm[name]) != count {
			form.Lines = []receivingFormLine{{}}
			return form, receivingDraft{}, receivingFormErrors{}, false, errors.New("receiving form lines do not match")
		}
	}
	form.Lines = make([]receivingFormLine, count)
	for i := range form.Lines {
		form.Lines[i] = receivingFormLine{
			UnitID:       r.PostForm[values[0]][i],
			BatchNumber:  strings.TrimSpace(r.PostForm[values[1]][i]),
			Expiry:       r.PostForm[values[2]][i],
			Quantity:     r.PostForm[values[3]][i],
			InvoicePrice: r.PostForm[values[4]][i],
		}
	}

	var formErrors receivingFormErrors
	draft := receivingDraft{InvoiceNumber: form.InvoiceNumber}
	valid := true
	distributorID, err := strconv.ParseInt(form.DistributorID, 10, 32)
	if err != nil || distributorID < 1 {
		formErrors.Distributor = "Pilih distributor PBF."
		valid = false
	} else {
		draft.DistributorID = int(distributorID)
	}
	if form.InvoiceNumber == "" || !validReceivingText(form.InvoiceNumber, 100) {
		formErrors.Invoice = "Nomor faktur wajib diisi dan maksimal 100 karakter."
		valid = false
	}
	var dateValid bool
	draft.InvoiceDate, dateValid = parseReceivingDate(form.InvoiceDate)
	if !dateValid {
		formErrors.InvoiceDate = "Masukkan tanggal faktur yang valid."
		valid = false
	}
	draft.ReceivedDate, dateValid = parseReceivingDate(form.ReceivedDate)
	if !dateValid {
		formErrors.ReceivedDate = "Masukkan tanggal terima yang valid."
		valid = false
	}
	if len(form.Lines) == 0 {
		valid = false
	}

	draft.Lines = make([]receivingLineDraft, 0, len(form.Lines))
	var total int64
	for i := range form.Lines {
		line := &form.Lines[i]
		lineValid := true
		unitID, err := strconv.ParseInt(line.UnitID, 10, 32)
		if err != nil || unitID < 1 {
			line.UnitError = "Pilih obat dan satuan."
			lineValid = false
		}
		if line.BatchNumber == "" || !validReceivingText(line.BatchNumber, 100) {
			line.BatchError = "Nomor batch wajib diisi dan maksimal 100 karakter."
			lineValid = false
		}
		expiry, expiryValid := parseReceivingDate(line.Expiry)
		if !expiryValid {
			line.ExpiryError = "Masukkan tanggal kedaluwarsa yang valid."
			lineValid = false
		} else if dateValid && !draft.ReceivedDate.IsZero() {
			if _, err := receivingBatchStatus(expiry, draft.ReceivedDate); errors.Is(err, ErrExpiryBeforeReceipt) {
				line.ExpiryError = "Tanggal kedaluwarsa tidak boleh sebelum tanggal terima."
				lineValid = false
			}
		}
		quantity, err := strconv.ParseInt(line.Quantity, 10, 32)
		if err != nil || quantity < 1 {
			line.QuantityError = "Jumlah faktur harus berupa bilangan bulat lebih dari 0."
			lineValid = false
		}
		price, err := strconv.ParseInt(line.InvoicePrice, 10, 32)
		if err != nil || price < 0 {
			line.PriceError = "Harga faktur harus berupa bilangan bulat 0 atau lebih."
			lineValid = false
		}
		if lineValid {
			_, subtotal, err := receivingLineAmounts(int(quantity), int(price), 1)
			if err != nil || total > maxPostgresInt-subtotal {
				line.PriceError = "Jumlah dan harga menghasilkan subtotal di luar batas penyimpanan."
				lineValid = false
			} else {
				total += subtotal
				draft.Lines = append(draft.Lines, receivingLineDraft{
					UnitID:       int(unitID),
					BatchNumber:  line.BatchNumber,
					Expiry:       expiry,
					Quantity:     int(quantity),
					InvoicePrice: int(price),
				})
			}
		}
		if !lineValid {
			valid = false
		}
	}
	return form, draft, formErrors, valid, nil
}

func defaultReceivingForm() receivingForm {
	today := time.Now().Format(time.DateOnly)
	return receivingForm{
		InvoiceDate:  today,
		ReceivedDate: today,
		Lines:        []receivingFormLine{{}},
	}
}

func parseReceivingDate(value string) (time.Time, bool) {
	date, err := time.Parse(time.DateOnly, value)
	return date, err == nil && date.Format(time.DateOnly) == value
}

func dateOnly(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func validReceivingText(value string, maxRunes int) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && utf8.RuneCountInString(value) <= maxRunes
}

func formatRupiah(amount int) string {
	return "Rp " + groupRupiahDigits(strconv.Itoa(amount))
}

func formatRupiahDecimal(amount string) string {
	whole, fraction, hasFraction := strings.Cut(amount, ".")
	if !hasFraction {
		return "Rp " + groupRupiahDigits(whole)
	}
	return "Rp " + groupRupiahDigits(whole) + "," + fraction
}

func groupRupiahDigits(value string) string {
	if len(value) <= 3 {
		return value
	}
	firstGroup := len(value) % 3
	if firstGroup == 0 {
		firstGroup = 3
	}
	groups := []string{value[:firstGroup]}
	for start := firstGroup; start < len(value); start += 3 {
		groups = append(groups, value[start:start+3])
	}
	return strings.Join(groups, ".")
}

func parseReceivingID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, errors.New("invalid receiving invoice ID")
	}
	return id, nil
}

func receivingStatusLabel(status string) string {
	switch status {
	case "AKTIF":
		return "Aktif"
	case "HAMPIR_ED":
		return "Hampir kedaluwarsa"
	case "KEDALUWARSA":
		return "Kedaluwarsa"
	case "HABIS":
		return "Habis"
	case "DIBATALKAN":
		return "Dibatalkan"
	case "DITERIMA":
		return "Diterima"
	default:
		return status
	}
}

func receivingNotice(status string) string {
	switch status {
	case "created":
		return "Faktur penerimaan berhasil disimpan."
	case "cancelled":
		return "Faktur penerimaan dibatalkan dan stoknya dikurangi."
	default:
		return ""
	}
}

func setReceivingLineFieldError(form *receivingForm, lineError *receivingLineStoreError, field, message string) {
	if lineError.Index < 0 || lineError.Index >= len(form.Lines) {
		return
	}
	line := &form.Lines[lineError.Index]
	switch field {
	case "expiry":
		line.ExpiryError = message
	case "quantity":
		line.QuantityError = message
	default:
		line.UnitError = message
	}
}

func (h *receivingHandler) render(w http.ResponseWriter, name string, data any, status int) {
	var body bytes.Buffer
	if err := h.page.ExecuteTemplate(&body, name, data); err != nil {
		log.Printf("render receiving page %s: %v", name, err)
		http.Error(w, "Halaman penerimaan tidak dapat ditampilkan.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if _, err := w.Write(body.Bytes()); err != nil {
		log.Printf("write receiving page %s: %v", name, err)
	}
}
