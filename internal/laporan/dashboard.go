package laporan

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardData struct {
	TotalObat        int64
	TotalBatch       int64
	TotalStok        int64
	HampirED         int64
	Kedaluwarsa      int64
	TransaksiHariIni int64
}

func GetDashboard(ctx context.Context, pool *pgxpool.Pool) (DashboardData, error) {
	var data DashboardData

	query := `
		SELECT
			(SELECT COUNT(*)
			 FROM obat
			 WHERE status_aktif = TRUE),

			(SELECT COUNT(*)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa >= CURRENT_DATE),

			(SELECT COALESCE(SUM(stok_tersisa), 0)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa >= CURRENT_DATE),

			(SELECT COUNT(*)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa BETWEEN CURRENT_DATE
			   AND CURRENT_DATE + 30),

			(SELECT COUNT(*)
			 FROM batch
			 WHERE stok_tersisa > 0
			   AND tanggal_kedaluwarsa < CURRENT_DATE),

			(SELECT COUNT(*)
			 FROM penjualan
			 WHERE status = 'SELESAI'
			   AND tanggal_penjualan >= CURRENT_DATE
			   AND tanggal_penjualan < CURRENT_DATE + INTERVAL '1 day')
	`

	err := pool.QueryRow(ctx, query).Scan(
		&data.TotalObat,
		&data.TotalBatch,
		&data.TotalStok,
		&data.HampirED,
		&data.Kedaluwarsa,
		&data.TransaksiHariIni,
	)
	if err != nil {
		return DashboardData{}, fmt.Errorf("get dashboard summary: %w", err)
	}

	return data, nil
}
