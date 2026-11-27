package laporan

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GrossProfitReport struct {
	TotalPenjualan    int64
	TotalHPP          int64
	TotalPengembalian int64
	HPPReturMasuk     int64
	PenjualanBersih   int64
	HPPBersih         int64
	LabaKotor         int64
	MarginPersen      float64
	NilaiAsetStok     int64
}

type GrossProfitTransaction struct {
	IDPenjualan       int
	NomorTransaksi    string
	TanggalTransaksi  string
	NamaKasir         string
	MetodePembayaran  string
	TotalPenjualan    int64
	TotalHPP          int64
	TotalPengembalian int64
	HPPReturMasuk     int64
	LabaKotor         int64
}

func GetGrossProfit(
	ctx context.Context,
	pool *pgxpool.Pool,
	tanggalMulai string,
	tanggalAkhir string,
	keyword string,
) (GrossProfitReport, error) {
	var report GrossProfitReport
	keyword = strings.TrimSpace(keyword)

	query := `
		WITH penjualan_periode AS (
			SELECT
				id_penjualan,
				total_penjualan,
				total_hpp
			FROM penjualan
			WHERE status = 'SELESAI'
			  AND tanggal_penjualan >= $1::date
			  AND tanggal_penjualan < ($2::date + INTERVAL '1 day')
			  AND ($3 = '' OR nomor_transaksi ILIKE '%' || $3 || '%')
		),
		total_retur AS (
			SELECT
				rp.id_penjualan,
				SUM(rp.total_pengembalian)::BIGINT AS total_pengembalian
			FROM retur_penjualan rp
			WHERE rp.status = 'DISETUJUI'
			GROUP BY rp.id_penjualan
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
			COALESCE(SUM(p.total_penjualan), 0),
			COALESCE(SUM(p.total_hpp), 0),
			COALESCE(SUM(tr.total_pengembalian), 0),
			COALESCE(SUM(hr.hpp_retur_masuk), 0),
			(SELECT COALESCE(SUM(ROUND(stok_tersisa * harga_beli_dasar)), 0)::BIGINT
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa > CURRENT_DATE + 30)
		FROM penjualan_periode p
		LEFT JOIN total_retur tr ON tr.id_penjualan = p.id_penjualan
		LEFT JOIN hpp_retur hr ON hr.id_penjualan = p.id_penjualan
	`

	err := pool.QueryRow(
		ctx,
		query,
		tanggalMulai,
		tanggalAkhir,
		keyword,
	).Scan(
		&report.TotalPenjualan,
		&report.TotalHPP,
		&report.TotalPengembalian,
		&report.HPPReturMasuk,
		&report.NilaiAsetStok,
	)
	if err != nil {
		return GrossProfitReport{}, fmt.Errorf("get gross profit report: %w", err)
	}

	report.PenjualanBersih = report.TotalPenjualan - report.TotalPengembalian
	report.HPPBersih = report.TotalHPP - report.HPPReturMasuk
	report.LabaKotor = report.PenjualanBersih - report.HPPBersih
	if report.PenjualanBersih > 0 {
		report.MarginPersen = (float64(report.LabaKotor) / float64(report.PenjualanBersih)) * 100
	}

	return report, nil
}

func GetGrossProfitTransactions(
	ctx context.Context,
	pool *pgxpool.Pool,
	tanggalMulai string,
	tanggalAkhir string,
	keyword string,
) ([]GrossProfitTransaction, error) {
	keyword = strings.TrimSpace(keyword)

	query := `
		WITH penjualan_periode AS (
			SELECT
				p.id_penjualan,
				p.nomor_transaksi,
				p.tanggal_penjualan,
				u.nama_pengguna,
				p.metode_pembayaran,
				p.total_penjualan,
				p.total_hpp
			FROM penjualan p
			JOIN pengguna u ON u.id_pengguna = p.id_pengguna
			WHERE p.status = 'SELESAI'
			  AND p.tanggal_penjualan >= $1::date
			  AND p.tanggal_penjualan < ($2::date + INTERVAL '1 day')
			  AND ($3 = '' OR p.nomor_transaksi ILIKE '%' || $3 || '%')
		),
		total_retur AS (
			SELECT
				rp.id_penjualan,
				SUM(rp.total_pengembalian)::BIGINT AS total_pengembalian
			FROM retur_penjualan rp
			WHERE rp.status = 'DISETUJUI'
			GROUP BY rp.id_penjualan
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
			p.id_penjualan,
			p.nomor_transaksi,
			TO_CHAR(p.tanggal_penjualan, 'DD-MM-YYYY HH24:MI'),
			p.nama_pengguna,
			p.metode_pembayaran,
			p.total_penjualan,
			p.total_hpp,
			COALESCE(tr.total_pengembalian, 0),
			COALESCE(hr.hpp_retur_masuk, 0),
			(p.total_penjualan - COALESCE(tr.total_pengembalian, 0))
			- (p.total_hpp - COALESCE(hr.hpp_retur_masuk, 0))
		FROM penjualan_periode p
		LEFT JOIN total_retur tr ON tr.id_penjualan = p.id_penjualan
		LEFT JOIN hpp_retur hr ON hr.id_penjualan = p.id_penjualan
		ORDER BY p.tanggal_penjualan DESC, p.id_penjualan DESC
	`

	rows, err := pool.Query(ctx, query, tanggalMulai, tanggalAkhir, keyword)
	if err != nil {
		return nil, fmt.Errorf("get gross profit transactions: %w", err)
	}
	defer rows.Close()

	var transactions []GrossProfitTransaction
	for rows.Next() {
		var tx GrossProfitTransaction
		if err := rows.Scan(
			&tx.IDPenjualan,
			&tx.NomorTransaksi,
			&tx.TanggalTransaksi,
			&tx.NamaKasir,
			&tx.MetodePembayaran,
			&tx.TotalPenjualan,
			&tx.TotalHPP,
			&tx.TotalPengembalian,
			&tx.HPPReturMasuk,
			&tx.LabaKotor,
		); err != nil {
			return nil, fmt.Errorf("scan gross profit transaction: %w", err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, rows.Err()
}
