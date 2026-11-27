package laporan

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StockFilter struct {
	Keyword string
	Zona    string
	Status  string
	Rak     string
}

type ZoneSummary struct {
	Merah  int64
	Kuning int64
	Hijau  int64
	Habis  int64
}

type BatchStock struct {
	IDBatch            int
	KodeObat           string
	NamaObat           string
	SatuanDasar        string
	LokasiRak          string
	NomorBatch         string
	TanggalKedaluwarsa string
	HariMenujuED       int
	Stok               int
	Status             string
	ZonaED             string
	ZoneLabel          string
	ZoneDetail         string
	BadgeClass         string
}

// OwnerBatchStock adds purchase cost and asset valuation strictly for pemilik views.
type OwnerBatchStock struct {
	BatchStock
	HargaBeliDasar float64
	NilaiAset      int64
}

func GetZoneSummary(ctx context.Context, pool *pgxpool.Pool) (ZoneSummary, error) {
	var z ZoneSummary
	query := `
		SELECT
			COUNT(*) FILTER (WHERE stok_tersisa > 0 AND tanggal_kedaluwarsa <= CURRENT_DATE + 30),
			COUNT(*) FILTER (WHERE stok_tersisa > 0 AND tanggal_kedaluwarsa > CURRENT_DATE + 30 AND tanggal_kedaluwarsa <= CURRENT_DATE + 60),
			COUNT(*) FILTER (WHERE stok_tersisa > 0 AND tanggal_kedaluwarsa > CURRENT_DATE + 60),
			COUNT(*) FILTER (WHERE stok_tersisa = 0)
		FROM batch
	`
	if err := pool.QueryRow(ctx, query).Scan(&z.Merah, &z.Kuning, &z.Hijau, &z.Habis); err != nil {
		return ZoneSummary{}, fmt.Errorf("get zone summary: %w", err)
	}
	return z, nil
}

func GetRackLocations(ctx context.Context, pool *pgxpool.Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT DISTINCT lokasi_rak FROM obat ORDER BY lokasi_rak ASC`)
	if err != nil {
		return nil, fmt.Errorf("get rack locations: %w", err)
	}
	defer rows.Close()

	var racks []string
	for rows.Next() {
		var rak string
		if err := rows.Scan(&rak); err != nil {
			return nil, fmt.Errorf("scan rack location: %w", err)
		}
		racks = append(racks, rak)
	}
	return racks, rows.Err()
}

const batchFilterWhereSQL = `
	WHERE
		($1 = '' OR
		 o.kode_obat ILIKE '%' || $1 || '%' OR
		 o.nama_obat ILIKE '%' || $1 || '%' OR
		 b.nomor_batch ILIKE '%' || $1 || '%' OR
		 o.lokasi_rak ILIKE '%' || $1 || '%')
		AND ($2 = '' OR o.lokasi_rak = $2)
		AND (
			$3 = '' OR
			($3 = 'MERAH' AND b.stok_tersisa > 0 AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 30) OR
			($3 = 'KUNING' AND b.stok_tersisa > 0 AND b.tanggal_kedaluwarsa > CURRENT_DATE + 30 AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 60) OR
			($3 = 'HIJAU' AND b.stok_tersisa > 0 AND b.tanggal_kedaluwarsa > CURRENT_DATE + 60) OR
			($3 = 'HABIS' AND b.stok_tersisa = 0)
		)
		AND (
			$4 = '' OR
			($4 = 'HABIS' AND b.stok_tersisa = 0) OR
			($4 = 'KEDALUWARSA' AND b.stok_tersisa > 0 AND b.tanggal_kedaluwarsa < CURRENT_DATE) OR
			($4 = 'HAMPIR_ED' AND b.stok_tersisa > 0 AND b.tanggal_kedaluwarsa >= CURRENT_DATE AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 30) OR
			($4 = 'AKTIF' AND b.stok_tersisa > 0 AND b.tanggal_kedaluwarsa > CURRENT_DATE + 30)
		)
	ORDER BY b.tanggal_kedaluwarsa ASC, b.id_batch ASC
`

func GetBatchStock(ctx context.Context, pool *pgxpool.Pool, filter StockFilter) ([]BatchStock, error) {
	query := `
		SELECT
			b.id_batch,
			o.kode_obat,
			o.nama_obat,
			s.nama_satuan,
			o.lokasi_rak,
			b.nomor_batch,
			TO_CHAR(b.tanggal_kedaluwarsa, 'DD-MM-YYYY'),
			(b.tanggal_kedaluwarsa - CURRENT_DATE)::INT,
			b.stok_tersisa
		FROM batch b
		JOIN obat o ON o.id_obat = b.id_obat
		JOIN satuan s ON s.id_satuan = o.satuan_dasar_id
	` + batchFilterWhereSQL

	rows, err := pool.Query(
		ctx,
		query,
		strings.TrimSpace(filter.Keyword),
		strings.TrimSpace(filter.Rak),
		strings.TrimSpace(filter.Zona),
		strings.TrimSpace(filter.Status),
	)
	if err != nil {
		return nil, fmt.Errorf("get batch stock: %w", err)
	}
	defer rows.Close()

	var result []BatchStock
	for rows.Next() {
		var item BatchStock
		if err := rows.Scan(
			&item.IDBatch,
			&item.KodeObat,
			&item.NamaObat,
			&item.SatuanDasar,
			&item.LokasiRak,
			&item.NomorBatch,
			&item.TanggalKedaluwarsa,
			&item.HariMenujuED,
			&item.Stok,
		); err != nil {
			return nil, fmt.Errorf("scan batch stock: %w", err)
		}
		cls := ClassifyExpiry(item.HariMenujuED, item.Stok)
		item.Status = cls.StatusEfektif
		item.ZonaED = cls.ZonaED
		item.ZoneLabel = cls.Label
		item.ZoneDetail = cls.Detail
		item.BadgeClass = cls.BadgeClass
		result = append(result, item)
	}

	return result, rows.Err()
}

func GetOwnerBatchStock(ctx context.Context, pool *pgxpool.Pool, filter StockFilter) ([]OwnerBatchStock, int64, error) {
	query := `
		SELECT
			b.id_batch,
			o.kode_obat,
			o.nama_obat,
			s.nama_satuan,
			o.lokasi_rak,
			b.nomor_batch,
			TO_CHAR(b.tanggal_kedaluwarsa, 'DD-MM-YYYY'),
			(b.tanggal_kedaluwarsa - CURRENT_DATE)::INT,
			b.stok_tersisa,
			b.harga_beli_dasar::DOUBLE PRECISION,
			ROUND(b.stok_tersisa * b.harga_beli_dasar)::BIGINT
		FROM batch b
		JOIN obat o ON o.id_obat = b.id_obat
		JOIN satuan s ON s.id_satuan = o.satuan_dasar_id
	` + batchFilterWhereSQL

	rows, err := pool.Query(
		ctx,
		query,
		strings.TrimSpace(filter.Keyword),
		strings.TrimSpace(filter.Rak),
		strings.TrimSpace(filter.Zona),
		strings.TrimSpace(filter.Status),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("get owner batch stock: %w", err)
	}
	defer rows.Close()

	var result []OwnerBatchStock
	var totalAsset int64
	for rows.Next() {
		var item OwnerBatchStock
		if err := rows.Scan(
			&item.IDBatch,
			&item.KodeObat,
			&item.NamaObat,
			&item.SatuanDasar,
			&item.LokasiRak,
			&item.NomorBatch,
			&item.TanggalKedaluwarsa,
			&item.HariMenujuED,
			&item.Stok,
			&item.HargaBeliDasar,
			&item.NilaiAset,
		); err != nil {
			return nil, 0, fmt.Errorf("scan owner batch stock: %w", err)
		}
		cls := ClassifyExpiry(item.HariMenujuED, item.Stok)
		item.Status = cls.StatusEfektif
		item.ZonaED = cls.ZonaED
		item.ZoneLabel = cls.Label
		item.ZoneDetail = cls.Detail
		item.BadgeClass = cls.BadgeClass
		if item.Stok > 0 && item.HariMenujuED > 30 {
			totalAsset += item.NilaiAset
		}
		result = append(result, item)
	}

	return result, totalAsset, rows.Err()
}
