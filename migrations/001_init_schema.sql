-- Sistem Apotek Sumber Waras - Initial PostgreSQL 17 Schema (19 Tables)
-- Authoritative reference: DATABASE.MD

BEGIN;

-- 1. Authentication, roles, and cashier shifts

CREATE TABLE role (
    id_role SERIAL PRIMARY KEY,
    nama_role VARCHAR(30) UNIQUE NOT NULL,
    deskripsi TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE pengguna (
    id_pengguna SERIAL PRIMARY KEY,
    id_role INT NOT NULL REFERENCES role(id_role) ON DELETE RESTRICT,
    nama_pengguna VARCHAR(100) NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(150) UNIQUE NULL,
    password_hash VARCHAR(255) NOT NULL,
    status_aktif BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_login TIMESTAMPTZ NULL
);

CREATE TABLE sessions (
    id_session SERIAL PRIMARY KEY,
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE shift_kasir (
    id_shift SERIAL PRIMARY KEY,
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    saldo_awal INT NOT NULL DEFAULT 0 CHECK (saldo_awal >= 0),
    kas_fisik_akhir INT NULL CHECK (kas_fisik_akhir >= 0),
    status_shift VARCHAR(20) NOT NULL DEFAULT 'AKTIF' CHECK (status_shift IN ('AKTIF', 'TERKUNCI', 'TUTUP')),
    waktu_buka TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    waktu_tutup TIMESTAMPTZ NULL
);

-- 2. Medicine catalog, units, and distributors

CREATE TABLE satuan (
    id_satuan SERIAL PRIMARY KEY,
    nama_satuan VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE obat (
    id_obat SERIAL PRIMARY KEY,
    kode_obat VARCHAR(30) UNIQUE NOT NULL,
    nama_obat VARCHAR(150) NOT NULL,
    kandungan TEXT NULL,
    satuan_dasar_id INT NOT NULL REFERENCES satuan(id_satuan) ON DELETE RESTRICT,
    lokasi_rak VARCHAR(50) NOT NULL DEFAULT 'RAK-UMUM',
    status_aktif BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE obat_satuan (
    id_obat_satuan SERIAL PRIMARY KEY,
    id_obat INT NOT NULL REFERENCES obat(id_obat) ON DELETE RESTRICT,
    id_satuan INT NOT NULL REFERENCES satuan(id_satuan) ON DELETE RESTRICT,
    faktor_konversi INT NOT NULL CHECK (faktor_konversi > 0),
    harga_jual INT NOT NULL DEFAULT 0 CHECK (harga_jual >= 0),
    boleh_dijual BOOLEAN NOT NULL DEFAULT TRUE,
    aktif BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (id_obat, id_satuan)
);

CREATE TABLE pbf (
    id_pbf SERIAL PRIMARY KEY,
    nama_pbf VARCHAR(150) UNIQUE NOT NULL,
    aktif BOOLEAN NOT NULL DEFAULT TRUE
);

-- 3. PBF receiving, batches, and stock ledger

CREATE TABLE penerimaan (
    id_penerimaan SERIAL PRIMARY KEY,
    id_pbf INT NOT NULL REFERENCES pbf(id_pbf) ON DELETE RESTRICT,
    no_faktur VARCHAR(100) NOT NULL,
    tanggal_faktur DATE NOT NULL,
    tanggal_terima DATE NOT NULL DEFAULT CURRENT_DATE,
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    total_faktur INT NOT NULL CHECK (total_faktur >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'DITERIMA' CHECK (status IN ('DITERIMA', 'DIBATALKAN')),
    alasan_batal TEXT NULL,
    waktu_batal TIMESTAMPTZ NULL,
    id_pengguna_batal INT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    UNIQUE (id_pbf, no_faktur)
);

CREATE TABLE batch (
    id_batch SERIAL PRIMARY KEY,
    id_obat INT NOT NULL REFERENCES obat(id_obat) ON DELETE RESTRICT,
    nomor_batch VARCHAR(100) NOT NULL,
    tanggal_kedaluwarsa DATE NOT NULL,
    harga_beli_dasar NUMERIC(14,4) NOT NULL CHECK (harga_beli_dasar >= 0),
    stok_tersisa INT NOT NULL DEFAULT 0 CHECK (stok_tersisa >= 0),
    status_batch VARCHAR(20) NOT NULL DEFAULT 'AKTIF' CHECK (status_batch IN ('AKTIF', 'HAMPIR_ED', 'KEDALUWARSA', 'HABIS')),
    waktu_dibuat TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_batch_fefo ON batch (id_obat, status_batch, tanggal_kedaluwarsa);

CREATE TABLE detail_penerimaan (
    id_detail_penerimaan SERIAL PRIMARY KEY,
    id_penerimaan INT NOT NULL REFERENCES penerimaan(id_penerimaan) ON DELETE RESTRICT,
    id_batch INT UNIQUE NOT NULL REFERENCES batch(id_batch) ON DELETE RESTRICT,
    id_obat_satuan INT NOT NULL REFERENCES obat_satuan(id_obat_satuan) ON DELETE RESTRICT,
    jumlah_faktur INT NOT NULL CHECK (jumlah_faktur > 0),
    harga_faktur INT NOT NULL CHECK (harga_faktur >= 0),
    faktor_konversi INT NOT NULL CHECK (faktor_konversi > 0),
    jumlah_dasar INT NOT NULL CHECK (jumlah_dasar > 0),
    subtotal INT NOT NULL CHECK (subtotal >= 0)
);

CREATE TABLE mutasi_stok (
    id_mutasi SERIAL PRIMARY KEY,
    id_batch INT NOT NULL REFERENCES batch(id_batch) ON DELETE RESTRICT,
    jenis_mutasi VARCHAR(30) NOT NULL,
    jumlah_dasar INT NOT NULL,
    jenis_referensi VARCHAR(50) NOT NULL,
    id_referensi INT NOT NULL,
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    waktu_mutasi TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mutasi_stok_batch_waktu ON mutasi_stok (id_batch, waktu_mutasi);
CREATE INDEX idx_mutasi_stok_referensi ON mutasi_stok (jenis_referensi, id_referensi);

-- 4. POS sales

CREATE TABLE penjualan (
    id_penjualan SERIAL PRIMARY KEY,
    nomor_transaksi VARCHAR(50) UNIQUE NOT NULL,
    tanggal_penjualan TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    id_shift INT NULL REFERENCES shift_kasir(id_shift) ON DELETE RESTRICT,
    total_penjualan INT NOT NULL CHECK (total_penjualan >= 0),
    total_hpp INT NOT NULL CHECK (total_hpp >= 0),
    metode_pembayaran VARCHAR(20) NOT NULL DEFAULT 'TUNAI' CHECK (metode_pembayaran IN ('TUNAI', 'QRIS', 'DEBIT')),
    jumlah_bayar INT NOT NULL DEFAULT 0 CHECK (jumlah_bayar >= 0),
    kembalian INT NOT NULL DEFAULT 0 CHECK (kembalian >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'SELESAI' CHECK (status IN ('SELESAI', 'DIBATALKAN')),
    alasan_batal TEXT NULL,
    waktu_batal TIMESTAMPTZ NULL,
    id_pengguna_batal INT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT
);

CREATE TABLE item_penjualan (
    id_item SERIAL PRIMARY KEY,
    id_penjualan INT NOT NULL REFERENCES penjualan(id_penjualan) ON DELETE RESTRICT,
    id_obat_satuan INT NOT NULL REFERENCES obat_satuan(id_obat_satuan) ON DELETE RESTRICT,
    jumlah INT NOT NULL CHECK (jumlah > 0),
    faktor_konversi INT NOT NULL CHECK (faktor_konversi > 0),
    jumlah_dasar INT NOT NULL CHECK (jumlah_dasar > 0),
    harga_jual_saat_jual INT NOT NULL CHECK (harga_jual_saat_jual >= 0),
    subtotal INT NOT NULL CHECK (subtotal >= 0)
);

CREATE TABLE detail_penjualan (
    id_detail SERIAL PRIMARY KEY,
    id_item INT NOT NULL REFERENCES item_penjualan(id_item) ON DELETE RESTRICT,
    id_batch INT NOT NULL REFERENCES batch(id_batch) ON DELETE RESTRICT,
    urutan_fefo INT NOT NULL CHECK (urutan_fefo > 0),
    jumlah_dasar INT NOT NULL CHECK (jumlah_dasar > 0),
    harga_beli_saat_jual NUMERIC(14,4) NOT NULL CHECK (harga_beli_saat_jual >= 0),
    hpp INT NOT NULL CHECK (hpp >= 0)
);

-- 5. Sales returns

CREATE TABLE retur_penjualan (
    id_retur SERIAL PRIMARY KEY,
    nomor_retur VARCHAR(50) UNIQUE NOT NULL,
    id_penjualan INT NOT NULL REFERENCES penjualan(id_penjualan) ON DELETE RESTRICT,
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    id_pengguna_penyetuju INT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    tanggal_retur TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_pengembalian INT NOT NULL CHECK (total_pengembalian >= 0),
    status VARCHAR(30) NOT NULL DEFAULT 'MENUNGGU_PERSETUJUAN' CHECK (status IN ('MENUNGGU_PERSETUJUAN', 'DISETUJUI', 'DITOLAK')),
    catatan_penyetuju TEXT NULL,
    waktu_selesai TIMESTAMPTZ NULL
);

CREATE TABLE detail_retur (
    id_detail_retur SERIAL PRIMARY KEY,
    id_retur INT NOT NULL REFERENCES retur_penjualan(id_retur) ON DELETE RESTRICT,
    id_detail INT NOT NULL REFERENCES detail_penjualan(id_detail) ON DELETE RESTRICT,
    jumlah_dasar INT NOT NULL CHECK (jumlah_dasar > 0),
    alasan_retur TEXT NOT NULL,
    kondisi_obat VARCHAR(50) NOT NULL,
    status_stok VARCHAR(30) NOT NULL CHECK (status_stok IN ('MASUK_STOK', 'TIDAK_MASUK_STOK')),
    subtotal_pengembalian INT NOT NULL CHECK (subtotal_pengembalian >= 0)
);

-- 6. Stock opname

CREATE TABLE stock_opname (
    id_stock_opname SERIAL PRIMARY KEY,
    nomor_sesi VARCHAR(50) UNIQUE NOT NULL,
    tanggal_opname DATE NOT NULL DEFAULT CURRENT_DATE,
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    id_pengguna_penyetuju INT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    status VARCHAR(30) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'MENUNGGU_PERSETUJUAN', 'SELESAI', 'DITOLAK')),
    catatan_penyetuju TEXT NULL,
    waktu_persetujuan TIMESTAMPTZ NULL
);

CREATE TABLE detail_stock_opname (
    id_detail_opname SERIAL PRIMARY KEY,
    id_stock_opname INT NOT NULL REFERENCES stock_opname(id_stock_opname) ON DELETE RESTRICT,
    id_batch INT NOT NULL REFERENCES batch(id_batch) ON DELETE RESTRICT,
    stok_sistem INT NOT NULL,
    stok_fisik INT NOT NULL CHECK (stok_fisik >= 0),
    selisih_stok INT NOT NULL,
    alasan_selisih TEXT NULL,
    urutan_penghitungan INT NOT NULL DEFAULT 1 CHECK (urutan_penghitungan > 0),
    id_pengguna INT NOT NULL REFERENCES pengguna(id_pengguna) ON DELETE RESTRICT,
    waktu_hitung TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status_hasil VARCHAR(20) NOT NULL DEFAULT 'PENDING' CHECK (status_hasil IN ('PENDING', 'SELESAI')),
    UNIQUE (id_stock_opname, id_batch, urutan_penghitungan)
);

COMMIT;
