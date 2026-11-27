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
	SatuanDasar        string
	NomorBatch         string
	LokasiRak          string
	TanggalKedaluwarsa string
	Stok               int
	HariMenujuED       int
	Status             string
	ZonaED             string
	ZoneLabel          string
	ZoneDetail         string
	BadgeClass         string
}

// GetExpiringBatches returns active-stock batches filtered by 3-zone ED.
// An empty zona defaults to attention zones (MERAH and KUNING, <= 60 days).
func GetExpiringBatches(ctx context.Context, pool *pgxpool.Pool, zona string) ([]ExpiringBatch, error) {
	query := `
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
		  AND (
			($1 = '' AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 60) OR
			($1 = 'SEMUA') OR
			($1 = 'MERAH' AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 30) OR
			($1 = 'KUNING' AND b.tanggal_kedaluwarsa > CURRENT_DATE + 30 AND b.tanggal_kedaluwarsa <= CURRENT_DATE + 60) OR
			($1 = 'HIJAU' AND b.tanggal_kedaluwarsa > CURRENT_DATE + 60)
		  )
		ORDER BY b.tanggal_kedaluwarsa ASC, b.id_batch ASC
	`

	rows, err := pool.Query(ctx, query, zona)
	if err != nil {
		return nil, fmt.Errorf("get expiring batches: %w", err)
	}
	defer rows.Close()

	var batches []ExpiringBatch
	for rows.Next() {
		var batch ExpiringBatch
		if err := rows.Scan(
			&batch.IDBatch,
			&batch.KodeObat,
			&batch.NamaObat,
			&batch.SatuanDasar,
			&batch.NomorBatch,
			&batch.LokasiRak,
			&batch.TanggalKedaluwarsa,
			&batch.Stok,
			&batch.HariMenujuED,
		); err != nil {
			return nil, fmt.Errorf("scan expiring batch: %w", err)
		}
		cls := ClassifyExpiry(batch.HariMenujuED, batch.Stok)
		batch.Status = cls.StatusEfektif
		batch.ZonaED = cls.ZonaED
		batch.ZoneLabel = cls.Label
		batch.ZoneDetail = cls.Detail
		batch.BadgeClass = cls.BadgeClass
		batches = append(batches, batch)
	}

	return batches, rows.Err()
}
