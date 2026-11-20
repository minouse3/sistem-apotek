package laporan

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type BatchStock struct {
	IDBatch            int
	KodeObat           string
	NamaObat           string
	LokasiRak          string
	NomorBatch         string
	TanggalKedaluwarsa string
	Stok               int
	Status             string
}

func GetBatchStock(ctx context.Context, pool *pgxpool.Pool, keyword, status string) ([]BatchStock, error) {
	query := `
		SELECT
			b.id_batch,
			o.kode_obat,
			o.nama_obat,
			o.lokasi_rak,
			b.nomor_batch,
			TO_CHAR(b.tanggal_kedaluwarsa, 'DD-MM-YYYY'),
			b.stok_tersisa,
			b.status_batch
		FROM batch b
		JOIN obat o ON o.id_obat = b.id_obat
		WHERE
			($1 = '' OR
			 o.kode_obat ILIKE '%' || $1 || '%' OR
			 o.nama_obat ILIKE '%' || $1 || '%' OR
			 b.nomor_batch ILIKE '%' || $1 || '%')
			AND ($2 = '' OR b.status_batch = $2)
		ORDER BY b.tanggal_kedaluwarsa ASC, o.nama_obat ASC
	`

	rows, err := pool.Query(ctx, query, strings.TrimSpace(keyword), strings.TrimSpace(status))
	if err != nil {
		return nil, fmt.Errorf("get batch stock: %w", err)
	}
	defer rows.Close()

	var result []BatchStock

	for rows.Next() {
		var item BatchStock

		err := rows.Scan(
			&item.IDBatch,
			&item.KodeObat,
			&item.NamaObat,
			&item.LokasiRak,
			&item.NomorBatch,
			&item.TanggalKedaluwarsa,
			&item.Stok,
			&item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("scan batch stock: %w", err)
		}

		result = append(result, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate batch stock: %w", err)
	}

	return result, nil
}
