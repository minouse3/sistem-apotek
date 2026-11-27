package laporan

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBatchNotFound = errors.New("batch not found")

type BatchMutation struct {
	IDMutasi       int
	Waktu          string
	JenisMutasi    string
	JenisLabel     string
	JumlahDasar    int
	JenisReferensi string
	IDReferensi    int
	NamaPengguna   string
}

type BatchAudit struct {
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
	NamaPBF            string
	NoFaktur           string
	TanggalTerima      string
	Mutasi             []BatchMutation
}

type OwnerBatchFinance struct {
	HargaBeliDasar float64
	NilaiAset      int64
}

func GetBatchAudit(ctx context.Context, pool *pgxpool.Pool, idBatch int) (BatchAudit, error) {
	var audit BatchAudit

	headerQuery := `
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
			COALESCE(pbf.nama_pbf, ''),
			COALESCE(pn.no_faktur, ''),
			COALESCE(TO_CHAR(pn.tanggal_terima, 'DD-MM-YYYY'), '')
		FROM batch b
		JOIN obat o ON o.id_obat = b.id_obat
		JOIN satuan s ON s.id_satuan = o.satuan_dasar_id
		LEFT JOIN detail_penerimaan dp ON dp.id_batch = b.id_batch
		LEFT JOIN penerimaan pn ON pn.id_penerimaan = dp.id_penerimaan
		LEFT JOIN pbf ON pbf.id_pbf = pn.id_pbf
		WHERE b.id_batch = $1
	`

	err := pool.QueryRow(ctx, headerQuery, idBatch).Scan(
		&audit.IDBatch,
		&audit.KodeObat,
		&audit.NamaObat,
		&audit.SatuanDasar,
		&audit.LokasiRak,
		&audit.NomorBatch,
		&audit.TanggalKedaluwarsa,
		&audit.HariMenujuED,
		&audit.Stok,
		&audit.NamaPBF,
		&audit.NoFaktur,
		&audit.TanggalTerima,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return BatchAudit{}, ErrBatchNotFound
	}
	if err != nil {
		return BatchAudit{}, fmt.Errorf("get batch audit header: %w", err)
	}

	cls := ClassifyExpiry(audit.HariMenujuED, audit.Stok)
	audit.Status = cls.StatusEfektif
	audit.ZonaED = cls.ZonaED
	audit.ZoneLabel = cls.Label
	audit.ZoneDetail = cls.Detail
	audit.BadgeClass = cls.BadgeClass

	ledgerQuery := `
		SELECT
			m.id_mutasi,
			TO_CHAR(m.waktu_mutasi, 'DD-MM-YYYY HH24:MI'),
			m.jenis_mutasi,
			m.jumlah_dasar,
			m.jenis_referensi,
			m.id_referensi,
			u.nama_pengguna
		FROM mutasi_stok m
		JOIN pengguna u ON u.id_pengguna = m.id_pengguna
		WHERE m.id_batch = $1
		ORDER BY m.waktu_mutasi DESC, m.id_mutasi DESC
	`

	rows, err := pool.Query(ctx, ledgerQuery, idBatch)
	if err != nil {
		return BatchAudit{}, fmt.Errorf("get batch mutations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m BatchMutation
		if err := rows.Scan(
			&m.IDMutasi,
			&m.Waktu,
			&m.JenisMutasi,
			&m.JumlahDasar,
			&m.JenisReferensi,
			&m.IDReferensi,
			&m.NamaPengguna,
		); err != nil {
			return BatchAudit{}, fmt.Errorf("scan batch mutation: %w", err)
		}
		m.JenisLabel = MutationLabel(m.JenisMutasi)
		audit.Mutasi = append(audit.Mutasi, m)
	}

	return audit, rows.Err()
}

func GetOwnerBatchFinance(ctx context.Context, pool *pgxpool.Pool, idBatch int) (OwnerBatchFinance, error) {
	var fin OwnerBatchFinance
	err := pool.QueryRow(ctx, `
		SELECT
			harga_beli_dasar::DOUBLE PRECISION,
			ROUND(stok_tersisa * harga_beli_dasar)::BIGINT
		FROM batch
		WHERE id_batch = $1
	`, idBatch).Scan(&fin.HargaBeliDasar, &fin.NilaiAset)
	if errors.Is(err, pgx.ErrNoRows) {
		return OwnerBatchFinance{}, ErrBatchNotFound
	}
	if err != nil {
		return OwnerBatchFinance{}, fmt.Errorf("get owner batch finance: %w", err)
	}
	return fin, nil
}
