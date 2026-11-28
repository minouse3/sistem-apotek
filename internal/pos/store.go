package pos

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/minouse3/sistem-apotek/internal/auth"
)

var wibLocation = time.FixedZone("WIB", 7*60*60)

var (
	ErrEmptyCart         = errors.New("keranjang kosong")
	ErrTooManyItems      = errors.New("jumlah item melebihi batas")
	ErrInvalidItem       = errors.New("item penjualan tidak valid")
	ErrInvalidPayment    = errors.New("pembayaran tidak valid")
	ErrInsufficientPay   = errors.New("jumlah bayar kurang dari total")
	ErrReceiptNotFound   = errors.New("transaksi tidak ditemukan")
	ErrAlreadyCanceled   = errors.New("transaksi sudah dibatalkan")
	ErrHasActiveReturn   = errors.New("transaksi memiliki retur aktif atau disetujui")
	ErrVoidWindowExpired = errors.New("batas waktu void kasir sudah lewat")
	ErrReasonRequired    = errors.New("alasan pembatalan wajib diisi")
)

const (
	maxCartItems   = 100
	maxPostgresInt = int64(math.MaxInt32)
)

type SellableProduct struct {
	IDObatSatuan   int    `json:"id_obat_satuan"`
	IDObat         int    `json:"id_obat"`
	KodeObat       string `json:"kode_obat"`
	NamaObat       string `json:"nama_obat"`
	LokasiRak      string `json:"lokasi_rak"`
	NamaSatuan     string `json:"nama_satuan"`
	SatuanDasar    string `json:"satuan_dasar"`
	FaktorKonversi int    `json:"faktor_konversi"`
	HargaJual      int    `json:"harga_jual"`
	StokLayak      int    `json:"stok_layak"`
	MaxPaket       int    `json:"max_paket"`
	BatchTerdekat  string `json:"batch_terdekat"`
}

type CartItemInput struct {
	IDObatSatuan int `json:"id_obat_satuan"`
	Jumlah       int `json:"jumlah"`
}

type CheckoutRequest struct {
	MetodePembayaran string          `json:"metode_pembayaran"`
	JumlahBayar      int             `json:"jumlah_bayar"`
	Items            []CartItemInput `json:"items"`
}

type PreviewBatch struct {
	NomorBatch  string `json:"nomor_batch"`
	ExpiryDate  string `json:"expiry_date"`
	JumlahDasar int    `json:"jumlah_dasar"`
}

type PreviewItem struct {
	IDObatSatuan   int            `json:"id_obat_satuan"`
	NamaObat       string         `json:"nama_obat"`
	NamaSatuan     string         `json:"nama_satuan"`
	SatuanDasar    string         `json:"satuan_dasar"`
	Jumlah         int            `json:"jumlah"`
	FaktorKonversi int            `json:"faktor_konversi"`
	JumlahDasar    int            `json:"jumlah_dasar"`
	HargaJual      int            `json:"harga_jual"`
	Subtotal       int            `json:"subtotal"`
	Batches        []PreviewBatch `json:"batches"`
}

type CartPreview struct {
	Items          []PreviewItem `json:"items"`
	TotalPenjualan int           `json:"total_penjualan"`
}

type ReceiptFilter struct {
	Keyword string
	Tanggal string
	Status  string
}

type ReceiptSummary struct {
	IDPenjualan      int
	NomorTransaksi   string
	TanggalLabel     string
	NamaKasir        string
	MetodePembayaran string
	TotalPenjualan   int
	Status           string
	StatusLabel      string
}

type ReceiptBatchDetail struct {
	IDDetail    int
	IDBatch     int
	NomorBatch  string
	ExpiryDate  string
	UrutanFEFO  int
	JumlahDasar int
}

type ReceiptLine struct {
	IDItem         int
	KodeObat       string
	NamaObat       string
	NamaSatuan     string
	SatuanDasar    string
	Jumlah         int
	FaktorKonversi int
	JumlahDasar    int
	HargaJual      int
	Subtotal       int
	Batches        []ReceiptBatchDetail
}

type ReceiptDetail struct {
	IDPenjualan      int
	NomorTransaksi   string
	TanggalLabel     string
	TanggalWIB       time.Time
	NamaKasir        string
	MetodePembayaran string
	TotalPenjualan   int
	JumlahBayar      int
	Kembalian        int
	Status           string
	StatusLabel      string
	AlasanBatal      string
	WaktuBatalLabel  string
	NamaPembatal     string
	HasActiveReturn  bool
	CanKasirVoid     bool
	CanOwnerCancel   bool
	Lines            []ReceiptLine
}

// OwnerReceiptFinance holds HPP and gross profit exclusively for pemilik.
type OwnerReceiptFinance struct {
	TotalHPP  int
	LabaKotor int
	LineHPP   map[int]int // keyed by id_item
}

type Store interface {
	ListSellableProducts(ctx context.Context, keyword string) ([]SellableProduct, error)
	PreviewCart(ctx context.Context, items []CartItemInput) (CartPreview, error)
	Checkout(ctx context.Context, user auth.User, req CheckoutRequest) (int, error)
	ListReceipts(ctx context.Context, filter ReceiptFilter, includeOwnerFinance bool) ([]ReceiptSummary, map[int]int, error)
	GetReceipt(ctx context.Context, id int, includeOwnerFinance bool) (ReceiptDetail, *OwnerReceiptFinance, error)
	VoidReceipt(ctx context.Context, id int, user auth.User, reason string) error
}

type PostgresStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		pool: pool,
		now:  time.Now,
	}
}

func (s *PostgresStore) ListSellableProducts(ctx context.Context, keyword string) ([]SellableProduct, error) {
	keyword = strings.TrimSpace(keyword)
	rows, err := s.pool.Query(ctx, `
		SELECT
			os.id_obat_satuan,
			o.id_obat,
			o.kode_obat,
			o.nama_obat,
			o.lokasi_rak,
			st.nama_satuan,
			sd.nama_satuan,
			os.faktor_konversi,
			os.harga_jual,
			COALESCE(SUM(b.stok_tersisa), 0)::INT AS stok_layak,
			COALESCE(TO_CHAR(MIN(b.tanggal_kedaluwarsa), 'DD-MM-YYYY'), '') AS batch_terdekat
		FROM obat_satuan os
		JOIN obat o ON o.id_obat = os.id_obat
		JOIN satuan st ON st.id_satuan = os.id_satuan
		JOIN satuan sd ON sd.id_satuan = o.satuan_dasar_id
		LEFT JOIN batch b ON b.id_obat = o.id_obat
			AND b.status_batch = 'AKTIF'
			AND b.stok_tersisa > 0
			AND b.tanggal_kedaluwarsa > CURRENT_DATE + 30
		WHERE o.status_aktif = TRUE
		  AND os.aktif = TRUE
		  AND os.boleh_dijual = TRUE
		  AND ($1 = '' OR o.kode_obat ILIKE '%' || $1 || '%' OR o.nama_obat ILIKE '%' || $1 || '%')
		GROUP BY os.id_obat_satuan, o.id_obat, o.kode_obat, o.nama_obat, o.lokasi_rak,
		         st.nama_satuan, sd.nama_satuan, os.faktor_konversi, os.harga_jual
		HAVING COALESCE(SUM(b.stok_tersisa), 0) >= os.faktor_konversi
		ORDER BY o.nama_obat ASC, os.faktor_konversi ASC
		LIMIT 100
	`, keyword)
	if err != nil {
		return nil, fmt.Errorf("list sellable products: %w", err)
	}
	defer rows.Close()

	var products []SellableProduct
	for rows.Next() {
		var p SellableProduct
		if err := rows.Scan(
			&p.IDObatSatuan,
			&p.IDObat,
			&p.KodeObat,
			&p.NamaObat,
			&p.LokasiRak,
			&p.NamaSatuan,
			&p.SatuanDasar,
			&p.FaktorKonversi,
			&p.HargaJual,
			&p.StokLayak,
			&p.BatchTerdekat,
		); err != nil {
			return nil, fmt.Errorf("scan sellable product: %w", err)
		}
		if p.FaktorKonversi > 0 {
			p.MaxPaket = p.StokLayak / p.FaktorKonversi
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func normalizeCartItems(items []CartItemInput) ([]CartItemInput, error) {
	if len(items) == 0 {
		return nil, ErrEmptyCart
	}
	if len(items) > maxCartItems {
		return nil, ErrTooManyItems
	}

	merged := make([]CartItemInput, 0, len(items))
	indexByUnit := make(map[int]int, len(items))
	for _, item := range items {
		if item.IDObatSatuan <= 0 || item.Jumlah <= 0 || int64(item.Jumlah) > maxPostgresInt {
			return nil, ErrInvalidItem
		}
		if idx, exists := indexByUnit[item.IDObatSatuan]; exists {
			sum := int64(merged[idx].Jumlah) + int64(item.Jumlah)
			if sum > maxPostgresInt {
				return nil, ErrInvalidItem
			}
			merged[idx].Jumlah = int(sum)
			continue
		}
		indexByUnit[item.IDObatSatuan] = len(merged)
		merged = append(merged, item)
	}
	return merged, nil
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func loadCartRequestsAndBatches(ctx context.Context, q querier, items []CartItemInput, lockBatches bool) ([]CartLineRequest, []CandidateBatch, error) {
	unitIDs := make([]int, len(items))
	for i, it := range items {
		unitIDs[i] = it.IDObatSatuan
	}

	unitRows, err := q.Query(ctx, `
		SELECT
			os.id_obat_satuan,
			o.id_obat,
			o.nama_obat,
			st.nama_satuan,
			sd.nama_satuan,
			os.faktor_konversi,
			os.harga_jual
		FROM obat_satuan os
		JOIN obat o ON o.id_obat = os.id_obat
		JOIN satuan st ON st.id_satuan = os.id_satuan
		JOIN satuan sd ON sd.id_satuan = o.satuan_dasar_id
		WHERE os.id_obat_satuan = ANY($1)
		  AND o.status_aktif = TRUE
		  AND os.aktif = TRUE
		  AND os.boleh_dijual = TRUE
	`, unitIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("load cart units: %w", err)
	}
	defer unitRows.Close()

	unitMap := make(map[int]CartLineRequest, len(items))
	obatSet := make(map[int]struct{})
	for unitRows.Next() {
		var req CartLineRequest
		if err := unitRows.Scan(
			&req.IDObatSatuan,
			&req.IDObat,
			&req.NamaObat,
			&req.NamaSatuan,
			&req.SatuanDasar,
			&req.FaktorKonversi,
			&req.HargaJual,
		); err != nil {
			return nil, nil, fmt.Errorf("scan cart unit: %w", err)
		}
		unitMap[req.IDObatSatuan] = req
		obatSet[req.IDObat] = struct{}{}
	}
	if err := unitRows.Err(); err != nil {
		return nil, nil, err
	}

	cartLines := make([]CartLineRequest, 0, len(items))
	for _, it := range items {
		req, ok := unitMap[it.IDObatSatuan]
		if !ok {
			return nil, nil, ErrInvalidItem
		}
		req.Jumlah = it.Jumlah
		if int64(req.Jumlah)*int64(req.FaktorKonversi) > maxPostgresInt ||
			int64(req.Jumlah)*int64(req.HargaJual) > maxPostgresInt {
			return nil, nil, ErrInvalidItem
		}
		cartLines = append(cartLines, req)
	}

	obatIDs := make([]int, 0, len(obatSet))
	for id := range obatSet {
		obatIDs = append(obatIDs, id)
	}
	sort.Ints(obatIDs)

	batchSQL := `
		SELECT
			b.id_batch,
			b.id_obat,
			b.nomor_batch,
			TO_CHAR(b.tanggal_kedaluwarsa, 'DD-MM-YYYY'),
			b.stok_tersisa,
			b.harga_beli_dasar::text
		FROM batch b
		WHERE b.id_obat = ANY($1)
		  AND b.status_batch = 'AKTIF'
		  AND b.stok_tersisa > 0
		  AND b.tanggal_kedaluwarsa > CURRENT_DATE + 30
		ORDER BY b.id_obat ASC, b.tanggal_kedaluwarsa ASC, b.id_batch ASC
	`
	if lockBatches {
		batchSQL += " FOR UPDATE"
	}

	batchRows, err := q.Query(ctx, batchSQL, obatIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("load candidate batches: %w", err)
	}
	defer batchRows.Close()

	var batches []CandidateBatch
	for batchRows.Next() {
		var b CandidateBatch
		var costText string
		if err := batchRows.Scan(
			&b.IDBatch,
			&b.IDObat,
			&b.NomorBatch,
			&b.ExpiryDate,
			&b.StokTersisa,
			&costText,
		); err != nil {
			return nil, nil, fmt.Errorf("scan candidate batch: %w", err)
		}
		cost, err := ParseCostTenThousandths(costText)
		if err != nil {
			return nil, nil, err
		}
		b.CostTenThousand = cost
		batches = append(batches, b)
	}
	if err := batchRows.Err(); err != nil {
		return nil, nil, err
	}

	return cartLines, batches, nil
}

func (s *PostgresStore) PreviewCart(ctx context.Context, items []CartItemInput) (CartPreview, error) {
	normalized, err := normalizeCartItems(items)
	if err != nil {
		return CartPreview{}, err
	}

	cartLines, batches, err := loadCartRequestsAndBatches(ctx, s.pool, normalized, false)
	if err != nil {
		return CartPreview{}, err
	}

	allocs, err := AllocateFEFO(cartLines, batches)
	if err != nil {
		return CartPreview{}, err
	}

	preview := CartPreview{Items: make([]PreviewItem, 0, len(allocs))}
	for _, a := range allocs {
		pBatches := make([]PreviewBatch, 0, len(a.Batches))
		for _, b := range a.Batches {
			pBatches = append(pBatches, PreviewBatch{
				NomorBatch:  b.NomorBatch,
				ExpiryDate:  b.ExpiryDate,
				JumlahDasar: b.JumlahDasar,
			})
		}
		preview.Items = append(preview.Items, PreviewItem{
			IDObatSatuan:   a.Line.IDObatSatuan,
			NamaObat:       a.Line.NamaObat,
			NamaSatuan:     a.Line.NamaSatuan,
			SatuanDasar:    a.Line.SatuanDasar,
			Jumlah:         a.Line.Jumlah,
			FaktorKonversi: a.Line.FaktorKonversi,
			JumlahDasar:    a.JumlahDasar,
			HargaJual:      a.Line.HargaJual,
			Subtotal:       a.Subtotal,
			Batches:        pBatches,
		})
		preview.TotalPenjualan += a.Subtotal
	}
	return preview, nil
}

func (s *PostgresStore) Checkout(ctx context.Context, user auth.User, req CheckoutRequest) (int, error) {
	method := strings.ToUpper(strings.TrimSpace(req.MetodePembayaran))
	if method != "TUNAI" && method != "QRIS" && method != "DEBIT" {
		return 0, ErrInvalidPayment
	}

	normalized, err := normalizeCartItems(req.Items)
	if err != nil {
		return 0, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin checkout tx: %w", err)
	}
	defer tx.Rollback(ctx)

	cartLines, batches, err := loadCartRequestsAndBatches(ctx, tx, normalized, true)
	if err != nil {
		return 0, err
	}

	allocs, err := AllocateFEFO(cartLines, batches)
	if err != nil {
		return 0, err
	}

	var totalPenjualan, totalHPP int64
	for _, a := range allocs {
		totalPenjualan += int64(a.Subtotal)
		totalHPP += int64(a.TotalHPP)
	}
	if totalPenjualan > maxPostgresInt || totalHPP > maxPostgresInt {
		return 0, ErrInvalidItem
	}

	totalInt := int(totalPenjualan)
	jumlahBayar := totalInt
	kembalian := 0
	if method == "TUNAI" {
		if req.JumlahBayar < totalInt {
			return 0, ErrInsufficientPay
		}
		jumlahBayar = req.JumlahBayar
		kembalian = req.JumlahBayar - totalInt
	}

	var shiftID *int
	var foundShift int
	err = tx.QueryRow(ctx, `
		SELECT id_shift
		FROM shift_kasir
		WHERE id_pengguna = $1 AND status_shift = 'AKTIF'
		ORDER BY waktu_buka DESC
		LIMIT 1
	`, user.ID).Scan(&foundShift)
	if err == nil {
		shiftID = &foundShift
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("lookup active shift: %w", err)
	}

	nowWIB := s.now().In(wibLocation)
	var saleID int
	err = tx.QueryRow(ctx, `SELECT nextval('penjualan_id_penjualan_seq')`).Scan(&saleID)
	if err != nil {
		return 0, fmt.Errorf("next penjualan id: %w", err)
	}
	nomorTransaksi := fmt.Sprintf("TRX-%s-%05d", nowWIB.Format("20060102"), saleID)

	_, err = tx.Exec(ctx, `
		INSERT INTO penjualan (
			id_penjualan, nomor_transaksi, tanggal_penjualan, id_pengguna, id_shift,
			total_penjualan, total_hpp, metode_pembayaran, jumlah_bayar, kembalian, status
		) VALUES ($1, $2, NOW(), $3, $4, $5, $6, $7, $8, $9, 'SELESAI')
	`, saleID, nomorTransaksi, user.ID, shiftID, totalInt, int(totalHPP), method, jumlahBayar, kembalian)
	if err != nil {
		return 0, fmt.Errorf("insert penjualan: %w", err)
	}

	batchDeductions := make(map[int]int)
	for _, a := range allocs {
		var itemID int
		err = tx.QueryRow(ctx, `
			INSERT INTO item_penjualan (
				id_penjualan, id_obat_satuan, jumlah, faktor_konversi, jumlah_dasar, harga_jual_saat_jual, subtotal
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id_item
		`, saleID, a.Line.IDObatSatuan, a.Line.Jumlah, a.Line.FaktorKonversi, a.JumlahDasar, a.Line.HargaJual, a.Subtotal).Scan(&itemID)
		if err != nil {
			return 0, fmt.Errorf("insert item_penjualan: %w", err)
		}

		for _, b := range a.Batches {
			_, err = tx.Exec(ctx, `
				INSERT INTO detail_penjualan (
					id_item, id_batch, urutan_fefo, jumlah_dasar, harga_beli_saat_jual, hpp
				) VALUES ($1, $2, $3, $4, $5, $6)
			`, itemID, b.IDBatch, b.UrutanFEFO, b.JumlahDasar, FormatCostTenThousandths(b.CostTenThousand), b.HPP)
			if err != nil {
				return 0, fmt.Errorf("insert detail_penjualan: %w", err)
			}
			batchDeductions[b.IDBatch] += b.JumlahDasar
		}
	}

	batchIDs := make([]int, 0, len(batchDeductions))
	for id := range batchDeductions {
		batchIDs = append(batchIDs, id)
	}
	sort.Ints(batchIDs)

	for _, batchID := range batchIDs {
		qty := batchDeductions[batchID]
		_, err = tx.Exec(ctx, `
			UPDATE batch
			SET stok_tersisa = stok_tersisa - $1,
			    status_batch = CASE WHEN stok_tersisa - $1 = 0 THEN 'HABIS' ELSE status_batch END
			WHERE id_batch = $2 AND stok_tersisa >= $1
		`, qty, batchID)
		if err != nil {
			return 0, fmt.Errorf("deduct batch stock: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO mutasi_stok (
				id_batch, jenis_mutasi, jumlah_dasar, jenis_referensi, id_referensi, id_pengguna
			) VALUES ($1, 'PENJUALAN', $2, 'penjualan', $3, $4)
		`, batchID, -qty, saleID, user.ID)
		if err != nil {
			return 0, fmt.Errorf("insert mutasi_stok: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit checkout: %w", err)
	}
	return saleID, nil
}

func statusLabel(status string) string {
	switch status {
	case "SELESAI":
		return "Selesai"
	case "DIBATALKAN":
		return "Dibatalkan"
	default:
		return status
	}
}

func (s *PostgresStore) ListReceipts(ctx context.Context, filter ReceiptFilter, includeOwnerFinance bool) ([]ReceiptSummary, map[int]int, error) {
	keyword := strings.TrimSpace(filter.Keyword)
	tanggal := strings.TrimSpace(filter.Tanggal)
	status := strings.TrimSpace(filter.Status)

	query := `
		SELECT
			p.id_penjualan,
			p.nomor_transaksi,
			p.tanggal_penjualan,
			u.nama_pengguna,
			p.metode_pembayaran,
			p.total_penjualan,
			p.status
	`
	if includeOwnerFinance {
		query += `, p.total_hpp`
	}
	query += `
		FROM penjualan p
		JOIN pengguna u ON u.id_pengguna = p.id_pengguna
		WHERE ($1 = '' OR p.nomor_transaksi ILIKE '%' || $1 || '%' OR u.nama_pengguna ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR (p.tanggal_penjualan AT TIME ZONE 'Asia/Jakarta')::date = $2::date)
		  AND ($3 = '' OR p.status = $3)
		ORDER BY p.tanggal_penjualan DESC, p.id_penjualan DESC
		LIMIT 100
	`

	rows, err := s.pool.Query(ctx, query, keyword, tanggal, status)
	if err != nil {
		return nil, nil, fmt.Errorf("list receipts: %w", err)
	}
	defer rows.Close()

	var receipts []ReceiptSummary
	var ownerHPP map[int]int
	if includeOwnerFinance {
		ownerHPP = make(map[int]int)
	}

	for rows.Next() {
		var r ReceiptSummary
		var ts time.Time
		if includeOwnerFinance {
			var hpp int
			if err := rows.Scan(&r.IDPenjualan, &r.NomorTransaksi, &ts, &r.NamaKasir, &r.MetodePembayaran, &r.TotalPenjualan, &r.Status, &hpp); err != nil {
				return nil, nil, fmt.Errorf("scan owner receipt: %w", err)
			}
			ownerHPP[r.IDPenjualan] = hpp
		} else {
			if err := rows.Scan(&r.IDPenjualan, &r.NomorTransaksi, &ts, &r.NamaKasir, &r.MetodePembayaran, &r.TotalPenjualan, &r.Status); err != nil {
				return nil, nil, fmt.Errorf("scan receipt: %w", err)
			}
		}
		r.TanggalLabel = ts.In(wibLocation).Format("02-01-2006 15:04")
		r.StatusLabel = statusLabel(r.Status)
		receipts = append(receipts, r)
	}
	return receipts, ownerHPP, rows.Err()
}

func withinKasirVoidWindow(saleTime, now time.Time) bool {
	saleWIB := saleTime.In(wibLocation)
	nowWIB := now.In(wibLocation)
	if saleWIB.Format("2006-01-02") != nowWIB.Format("2006-01-02") {
		return false
	}
	elapsed := now.Sub(saleTime)
	return elapsed >= 0 && elapsed <= 5*time.Minute
}

func (s *PostgresStore) GetReceipt(ctx context.Context, id int, includeOwnerFinance bool) (ReceiptDetail, *OwnerReceiptFinance, error) {
	var d ReceiptDetail
	var ts time.Time
	var waktuBatal *time.Time
	var alasanBatal, namaPembatal *string
	var totalHPP int

	headerSQL := `
		SELECT
			p.id_penjualan,
			p.nomor_transaksi,
			p.tanggal_penjualan,
			u.nama_pengguna,
			p.metode_pembayaran,
			p.total_penjualan,
			p.jumlah_bayar,
			p.kembalian,
			p.status,
			p.alasan_batal,
			p.waktu_batal,
			ub.nama_pengguna,
			EXISTS (
				SELECT 1 FROM retur_penjualan rp
				WHERE rp.id_penjualan = p.id_penjualan
				  AND rp.status IN ('MENUNGGU_PERSETUJUAN', 'DISETUJUI')
			)
	`
	if includeOwnerFinance {
		headerSQL += `, p.total_hpp`
	}
	headerSQL += `
		FROM penjualan p
		JOIN pengguna u ON u.id_pengguna = p.id_pengguna
		LEFT JOIN pengguna ub ON ub.id_pengguna = p.id_pengguna_batal
		WHERE p.id_penjualan = $1
	`

	var err error
	if includeOwnerFinance {
		err = s.pool.QueryRow(ctx, headerSQL, id).Scan(
			&d.IDPenjualan, &d.NomorTransaksi, &ts, &d.NamaKasir,
			&d.MetodePembayaran, &d.TotalPenjualan, &d.JumlahBayar, &d.Kembalian,
			&d.Status, &alasanBatal, &waktuBatal, &namaPembatal, &d.HasActiveReturn, &totalHPP,
		)
	} else {
		err = s.pool.QueryRow(ctx, headerSQL, id).Scan(
			&d.IDPenjualan, &d.NomorTransaksi, &ts, &d.NamaKasir,
			&d.MetodePembayaran, &d.TotalPenjualan, &d.JumlahBayar, &d.Kembalian,
			&d.Status, &alasanBatal, &waktuBatal, &namaPembatal, &d.HasActiveReturn,
		)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ReceiptDetail{}, nil, ErrReceiptNotFound
	}
	if err != nil {
		return ReceiptDetail{}, nil, fmt.Errorf("get receipt header: %w", err)
	}

	d.TanggalWIB = ts.In(wibLocation)
	d.TanggalLabel = d.TanggalWIB.Format("02-01-2006 15:04")
	d.StatusLabel = statusLabel(d.Status)
	if alasanBatal != nil {
		d.AlasanBatal = *alasanBatal
	}
	if waktuBatal != nil {
		d.WaktuBatalLabel = waktuBatal.In(wibLocation).Format("02-01-2006 15:04")
	}
	if namaPembatal != nil {
		d.NamaPembatal = *namaPembatal
	}
	d.CanOwnerCancel = d.Status == "SELESAI" && !d.HasActiveReturn
	d.CanKasirVoid = d.CanOwnerCancel && withinKasirVoidWindow(ts, s.now())

	lineRows, err := s.pool.Query(ctx, `
		SELECT
			ip.id_item,
			o.kode_obat,
			o.nama_obat,
			st.nama_satuan,
			sd.nama_satuan,
			ip.jumlah,
			ip.faktor_konversi,
			ip.jumlah_dasar,
			ip.harga_jual_saat_jual,
			ip.subtotal
		FROM item_penjualan ip
		JOIN obat_satuan os ON os.id_obat_satuan = ip.id_obat_satuan
		JOIN obat o ON o.id_obat = os.id_obat
		JOIN satuan st ON st.id_satuan = os.id_satuan
		JOIN satuan sd ON sd.id_satuan = o.satuan_dasar_id
		WHERE ip.id_penjualan = $1
		ORDER BY ip.id_item ASC
	`, id)
	if err != nil {
		return ReceiptDetail{}, nil, fmt.Errorf("get receipt lines: %w", err)
	}
	defer lineRows.Close()

	lineIdx := make(map[int]int)
	for lineRows.Next() {
		var l ReceiptLine
		if err := lineRows.Scan(
			&l.IDItem, &l.KodeObat, &l.NamaObat, &l.NamaSatuan, &l.SatuanDasar,
			&l.Jumlah, &l.FaktorKonversi, &l.JumlahDasar, &l.HargaJual, &l.Subtotal,
		); err != nil {
			return ReceiptDetail{}, nil, fmt.Errorf("scan receipt line: %w", err)
		}
		lineIdx[l.IDItem] = len(d.Lines)
		d.Lines = append(d.Lines, l)
	}
	if err := lineRows.Err(); err != nil {
		return ReceiptDetail{}, nil, err
	}

	var finance *OwnerReceiptFinance
	if includeOwnerFinance {
		finance = &OwnerReceiptFinance{
			TotalHPP:  totalHPP,
			LabaKotor: d.TotalPenjualan - totalHPP,
			LineHPP:   make(map[int]int, len(d.Lines)),
		}
	}

	detailSQL := `
		SELECT
			dp.id_detail,
			dp.id_item,
			dp.id_batch,
			b.nomor_batch,
			TO_CHAR(b.tanggal_kedaluwarsa, 'DD-MM-YYYY'),
			dp.urutan_fefo,
			dp.jumlah_dasar
	`
	if includeOwnerFinance {
		detailSQL += `, dp.hpp`
	}
	detailSQL += `
		FROM detail_penjualan dp
		JOIN item_penjualan ip ON ip.id_item = dp.id_item
		JOIN batch b ON b.id_batch = dp.id_batch
		WHERE ip.id_penjualan = $1
		ORDER BY dp.id_item ASC, dp.urutan_fefo ASC
	`

	detailRows, err := s.pool.Query(ctx, detailSQL, id)
	if err != nil {
		return ReceiptDetail{}, nil, fmt.Errorf("get receipt batch details: %w", err)
	}
	defer detailRows.Close()

	for detailRows.Next() {
		var bd ReceiptBatchDetail
		var itemID int
		if includeOwnerFinance {
			var hpp int
			if err := detailRows.Scan(&bd.IDDetail, &itemID, &bd.IDBatch, &bd.NomorBatch, &bd.ExpiryDate, &bd.UrutanFEFO, &bd.JumlahDasar, &hpp); err != nil {
				return ReceiptDetail{}, nil, fmt.Errorf("scan owner receipt batch detail: %w", err)
			}
			finance.LineHPP[itemID] += hpp
		} else {
			if err := detailRows.Scan(&bd.IDDetail, &itemID, &bd.IDBatch, &bd.NomorBatch, &bd.ExpiryDate, &bd.UrutanFEFO, &bd.JumlahDasar); err != nil {
				return ReceiptDetail{}, nil, fmt.Errorf("scan receipt batch detail: %w", err)
			}
		}
		if idx, ok := lineIdx[itemID]; ok {
			d.Lines[idx].Batches = append(d.Lines[idx].Batches, bd)
		}
	}
	if err := detailRows.Err(); err != nil {
		return ReceiptDetail{}, nil, err
	}

	return d, finance, nil
}

func (s *PostgresStore) VoidReceipt(ctx context.Context, id int, user auth.User, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 1000 {
		return ErrReasonRequired
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin void tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var saleTime time.Time
	var status string
	err = tx.QueryRow(ctx, `
		SELECT tanggal_penjualan, status
		FROM penjualan
		WHERE id_penjualan = $1
		FOR UPDATE
	`, id).Scan(&saleTime, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrReceiptNotFound
	}
	if err != nil {
		return fmt.Errorf("lock penjualan: %w", err)
	}

	if status != "SELESAI" {
		return ErrAlreadyCanceled
	}

	var hasReturn bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM retur_penjualan
			WHERE id_penjualan = $1
			  AND status IN ('MENUNGGU_PERSETUJUAN', 'DISETUJUI')
		)
	`, id).Scan(&hasReturn)
	if err != nil {
		return fmt.Errorf("check active return: %w", err)
	}
	if hasReturn {
		return ErrHasActiveReturn
	}

	if user.Role == auth.RoleKasir && !withinKasirVoidWindow(saleTime, s.now()) {
		return ErrVoidWindowExpired
	}

	rows, err := tx.Query(ctx, `
		SELECT dp.id_batch, SUM(dp.jumlah_dasar)::INT
		FROM detail_penjualan dp
		JOIN item_penjualan ip ON ip.id_item = dp.id_item
		WHERE ip.id_penjualan = $1
		GROUP BY dp.id_batch
		ORDER BY dp.id_batch ASC
	`, id)
	if err != nil {
		return fmt.Errorf("load void batch totals: %w", err)
	}
	defer rows.Close()

	type batchRestore struct {
		idBatch int
		qty     int
	}
	var restores []batchRestore
	for rows.Next() {
		var r batchRestore
		if err := rows.Scan(&r.idBatch, &r.qty); err != nil {
			return fmt.Errorf("scan batch restore: %w", err)
		}
		restores = append(restores, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range restores {
		_, err = tx.Exec(ctx, `SELECT id_batch FROM batch WHERE id_batch = $1 FOR UPDATE`, r.idBatch)
		if err != nil {
			return fmt.Errorf("lock batch %d: %w", r.idBatch, err)
		}

		_, err = tx.Exec(ctx, `
			UPDATE batch
			SET stok_tersisa = stok_tersisa + $1,
			    status_batch = CASE
			        WHEN tanggal_kedaluwarsa < CURRENT_DATE THEN 'KEDALUWARSA'
			        WHEN tanggal_kedaluwarsa <= CURRENT_DATE + 30 THEN 'HAMPIR_ED'
			        ELSE 'AKTIF'
			    END
			WHERE id_batch = $2
		`, r.qty, r.idBatch)
		if err != nil {
			return fmt.Errorf("restore batch %d: %w", r.idBatch, err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO mutasi_stok (
				id_batch, jenis_mutasi, jumlah_dasar, jenis_referensi, id_referensi, id_pengguna
			) VALUES ($1, 'PEMBATALAN_PENJUALAN', $2, 'penjualan', $3, $4)
		`, r.idBatch, r.qty, id, user.ID)
		if err != nil {
			return fmt.Errorf("insert void mutasi_stok: %w", err)
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE penjualan
		SET status = 'DIBATALKAN',
		    alasan_batal = $1,
		    waktu_batal = NOW(),
		    id_pengguna_batal = $2
		WHERE id_penjualan = $3
	`, reason, user.ID, id)
	if err != nil {
		return fmt.Errorf("update penjualan status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit void: %w", err)
	}
	return nil
}
