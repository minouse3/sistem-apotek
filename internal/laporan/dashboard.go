package laporan

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OperationalDashboard struct {
	TotalObat        int64
	TotalBatchLayak  int64
	TotalStokLayak   int64
	ZonaMerah        int64
	ZonaKuning       int64
	ZonaHijau        int64
	TransaksiHariIni int64
	BatchPerluTindak []ExpiringBatch
}

type OwnerFinanceSummary struct {
	PenjualanBersihBulanIni int64
	HPPBersihBulanIni       int64
	LabaKotorBulanIni       int64
	MarginPersen            float64
	NilaiAsetStok           int64
}

func GetOperationalDashboard(ctx context.Context, pool *pgxpool.Pool) (OperationalDashboard, error) {
	var data OperationalDashboard

	summaryQuery := `
		SELECT
			(SELECT COUNT(*)
			 FROM obat
			 WHERE status_aktif = TRUE),

			(SELECT COUNT(*)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa > CURRENT_DATE + 30),

			(SELECT COALESCE(SUM(stok_tersisa), 0)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa > CURRENT_DATE + 30),

			(SELECT COUNT(*)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa <= CURRENT_DATE + 30),

			(SELECT COUNT(*)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa > CURRENT_DATE + 30
			   AND tanggal_kedaluwarsa <= CURRENT_DATE + 60),

			(SELECT COUNT(*)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa > CURRENT_DATE + 60),

			(SELECT COUNT(*)
			 FROM penjualan
			 WHERE status = 'SELESAI'
			   AND tanggal_penjualan >= CURRENT_DATE
			   AND tanggal_penjualan < CURRENT_DATE + INTERVAL '1 day')
	`

	err := pool.QueryRow(ctx, summaryQuery).Scan(
		&data.TotalObat,
		&data.TotalBatchLayak,
		&data.TotalStokLayak,
		&data.ZonaMerah,
		&data.ZonaKuning,
		&data.ZonaHijau,
		&data.TransaksiHariIni,
	)
	if err != nil {
		return OperationalDashboard{}, fmt.Errorf("get operational dashboard summary: %w", err)
	}

	attentionQuery := `
		SELECT
			b.id_batch,
			o.kode_obat,
			o.nama_obat,
			s.nama_satuan,
			b.nomor_batch,
			o.lokasi_rak,
			TO_CHAR(b.tanggal_kedaluwarsa, 'DD-MM-YYYY'),
			b.stok_tersisa,
			(b.tanggal_kedaluwarsa - CURRENT_DATE)::INT
		FROM batch b
		JOIN obat o ON o.id_obat = b.id_obat
		JOIN satuan s ON s.id_satuan = o.satuan_dasar_id
		WHERE b.stok_tersisa > 0
		  AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 60
		ORDER BY b.tanggal_kedaluwarsa ASC, b.id_batch ASC
		LIMIT 6
	`

	rows, err := pool.Query(ctx, attentionQuery)
	if err != nil {
		return OperationalDashboard{}, fmt.Errorf("get attention batches: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item ExpiringBatch
		if err := rows.Scan(
			&item.IDBatch,
			&item.KodeObat,
			&item.NamaObat,
			&item.SatuanDasar,
			&item.NomorBatch,
			&item.LokasiRak,
			&item.TanggalKedaluwarsa,
			&item.Stok,
			&item.HariMenujuED,
		); err != nil {
			return OperationalDashboard{}, fmt.Errorf("scan attention batch: %w", err)
		}
		cls := ClassifyExpiry(item.HariMenujuED, item.Stok)
		item.Status = cls.StatusEfektif
		item.ZonaED = cls.ZonaED
		item.ZoneLabel = cls.Label
		item.ZoneDetail = cls.Detail
		item.BadgeClass = cls.BadgeClass
		data.BatchPerluTindak = append(data.BatchPerluTindak, item)
	}
	if err := rows.Err(); err != nil {
		return OperationalDashboard{}, fmt.Errorf("iterate attention batches: %w", err)
	}

	return data, nil
}

func GetOwnerFinanceSummary(ctx context.Context, pool *pgxpool.Pool) (OwnerFinanceSummary, error) {
	var summary OwnerFinanceSummary

	query := `
		WITH penjualan_bulan AS (
			SELECT id_penjualan, total_penjualan, total_hpp
			FROM penjualan
			WHERE status = 'SELESAI'
			  AND tanggal_penjualan >= DATE_TRUNC('month', CURRENT_DATE)
			  AND tanggal_penjualan < CURRENT_DATE + INTERVAL '1 day'
		),
		total_retur AS (
			SELECT id_penjualan, SUM(total_pengembalian)::BIGINT AS total_pengembalian
			FROM retur_penjualan
			WHERE status = 'DISETUJUI'
			GROUP BY id_penjualan
		),
		hpp_retur AS (
			SELECT
				rp.id_penjualan,
				SUM(ROUND(dr.jumlah_dasar * dp.harga_beli_saat_jual))::BIGINT AS hpp_retur_masuk
			FROM retur_penjualan rp
			JOIN detail_retur dr ON dr.id_retur = rp.id_retur
			JOIN detail_penjualan dp ON dp.id_detail = dr.id_detail
			WHERE rp.status = 'DISETUJUI'
			  AND dr.status_stok = 'MASUK_STOK'
			GROUP BY rp.id_penjualan
		)
		SELECT
			COALESCE(SUM(p.total_penjualan) - SUM(COALESCE(tr.total_pengembalian, 0)), 0),
			COALESCE(SUM(p.total_hpp) - SUM(COALESCE(hr.hpp_retur_masuk, 0)), 0),
			(SELECT COALESCE(SUM(ROUND(stok_tersisa * harga_beli_dasar)), 0)::BIGINT
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa > CURRENT_DATE + 30)
		FROM penjualan_bulan p
		LEFT JOIN total_retur tr ON tr.id_penjualan = p.id_penjualan
		LEFT JOIN hpp_retur hr ON hr.id_penjualan = p.id_penjualan
	`

	err := pool.QueryRow(ctx, query).Scan(
		&summary.PenjualanBersihBulanIni,
		&summary.HPPBersihBulanIni,
		&summary.NilaiAsetStok,
	)
	if err != nil {
		return OwnerFinanceSummary{}, fmt.Errorf("get owner finance summary: %w", err)
	}

	summary.LabaKotorBulanIni = summary.PenjualanBersihBulanIni - summary.HPPBersihBulanIni
	if summary.PenjualanBersihBulanIni > 0 {
		summary.MarginPersen = (float64(summary.LabaKotorBulanIni) / float64(summary.PenjualanBersihBulanIni)) * 100
	}

	return summary, nil
}
