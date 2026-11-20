package laporan

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ExpiringBatch struct {
	IDBatch            int
	KodeObat           string
	NamaObat           string
	NomorBatch         string
	LokasiRak          string
	TanggalKedaluwarsa string
	Stok               int
	HariMenujuED       int
	Status             string
}

func GetExpiringBatches(ctx context.Context, pool *pgxpool.Pool) ([]ExpiringBatch, error) {
	query := `
		SELECT
			b.id_batch,
			o.kode_obat,
			o.nama_obat,
			b.nomor_batch,
			o.lokasi_rak,
			TO_CHAR(b.tanggal_kedaluwarsa, 'DD-MM-YYYY'),
			b.stok_tersisa,
			(b.tanggal_kedaluwarsa - CURRENT_DATE)::INT,
			CASE
				WHEN b.tanggal_kedaluwarsa < CURRENT_DATE THEN 'KEDALUWARSA'
				ELSE 'HAMPIR_ED'
			END
		FROM batch b
		JOIN obat o ON o.id_obat = b.id_obat
		WHERE b.stok_tersisa > 0
		  AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 30
		ORDER BY b.tanggal_kedaluwarsa ASC, o.nama_obat ASC
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get expiring batches: %w", err)
	}
	defer rows.Close()

	var batches []ExpiringBatch

	for rows.Next() {
		var batch ExpiringBatch

		err := rows.Scan(
			&batch.IDBatch,
			&batch.KodeObat,
			&batch.NamaObat,
			&batch.NomorBatch,
			&batch.LokasiRak,
			&batch.TanggalKedaluwarsa,
			&batch.Stok,
			&batch.HariMenujuED,
			&batch.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("scan expiring batch: %w", err)
		}

		batches = append(batches, batch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expiring batches: %w", err)
	}

	return batches, nil
}
