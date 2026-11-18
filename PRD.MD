# Sistem Apotek Sumber Waras PRD

Sistem Apotek Sumber Waras is a single-binary web application for pharmacy inventory control, supplier receiving, FEFO point-of-sale checkout, sales returns, blind stock opname, and financial reporting. It follows the Week 12 scope using a Go monolith backend, Bootstrap 5.3 templates, and a 19-table PostgreSQL 17 database.

## Strict project specifications

All implementation, UI templates, interface copy, and database migrations MUST strictly adhere to the following normative documents. Where a detail conflicts with earlier drafts, these documents take precedence:

- **[DESIGN.MD](DESIGN.MD)** — Mandatory visual direction, Bootstrap 5.3 integration, desktop (`216px` white sticky sidebar) and narrow (`<992px` mobile bar and offcanvas) shell, color palette, Roboto (`400..700`) typography scale, Phosphor Bold (`15px–17px`) icons, component finish, and keyboard focus rules.
- **[WEBSITE-GUIDELINES.md](WEBSITE-GUIDELINES.md)** — Mandatory rules for interface copy, inline field error placement, demo-data transparency, working-controls-only UI (omitting unavailable controls), and code/comment hygiene.
- **[DATABASE.MD](DATABASE.MD)** — Authoritative PostgreSQL 17 schema (all 19 tables, column types, constraints, indexes, foreign-key `ON DELETE RESTRICT` rules, ERD, and cardinality).

## Roles and access rules

The application supports three user roles stored in the `role` table: `kasir`, `staf`, and `pemilik`.

Every application route except `/login` requires an authenticated session. The static assets needed to render the sign-in form are public. Role checks run in server-side HTTP middleware and return HTTP 403 Forbidden (`Akses ditolak`) when a user lacks permission. Inactive accounts (`status_aktif = false`) are rejected at login and re-checked on every authenticated request.

Sign-in uses a username and password, not email. Incorrect credentials show `Username atau password yang anda masukkan salah!` on the form. The browser holds an opaque session token in an `HttpOnly` cookie; PostgreSQL stores the token hash and expiration.

Visitors without an active session are redirected from `/` to `/login`; signed-in visitors to `/login` are redirected to `/`. `POST /logout` invalidates the current session, clears its cookie, and redirects to `/login`. A GET request must never sign a user out.

Financial fields (`harga_beli_dasar`, `total_hpp`, `harga_beli_saat_jual`, `hpp`, gross profit, and inventory asset value) belong exclusively to `pemilik`. Backend queries and response structs omit these fields for `kasir` and `staf` on all pages and dashboard widgets.

| Feature | Kasir | Staf | Pemilik |
| :--- | :--- | :--- | :--- |
| Login and logout | Yes | Yes | Yes |
| Operational home dashboard | Yes | Yes | Yes |
| Create, edit, and deactivate employee accounts | No | No | Yes |
| View and search medicines | Yes | Yes | Yes |
| Add and edit medicine data | No | Yes | Yes |
| Set packaging units and conversion factors | No | Conditional | Yes |
| Set and change selling prices | No | No | Yes |
| Deactivate medicine (soft delete) | No | Yes | Yes |
| Hard delete medicine with history | No | No | No |
| Manage PBF distributors | No | Yes | Yes |
| Check stock, nearest batch, and rack location | Yes | Yes | Yes |
| View stock-per-batch and near-expiry reports | Yes | Yes | Yes |
| View PBF receiving invoices | No | Yes | Yes |
| Input and save PBF receiving invoices | No | Yes | Yes |
| POS sales transaction (cart, payment, receipt) | Yes | No | Yes |
| Select batch number manually | No | No | No |
| Search and view sales receipts | Conditional | No | Yes |
| Input sales return and physical inspection | Yes | No | Yes |
| Approve or reject refund payout | No | No | Yes |
| On-the-spot receipt void | Conditional | No | Yes |
| Cancel historical sales receipt | No | No | Yes |
| Cancel PBF receiving invoice | No | No | Yes |
| Open opname session and input blind physical counts | No | Yes | Yes |
| View system stock and variance in submitted opname review | No | No | Yes |
| Approve or reject stock opname adjustments | No | No | Yes |
| Gross profit, HPP, stock asset value, and report export | No | No | Yes |

Conditional rules enforced in the service layer:

1. Opening and closing a cashier shift by `pemilik` is allowed only when acting as the cashier on duty.
2. Packaging units and conversion factors can be edited by `staf` only for new medicines that have no rows in `batch`. Once a medicine has a batch or transaction, only `pemilik` may change conversion factors.
3. Searching and viewing sales receipts by `kasir` is limited to return or void workflows, with HPP fields omitted.
4. On-the-spot void by `kasir` is permitted within 5 minutes of `tanggal_penjualan`, on the same calendar day, and before the medicine leaves the pharmacy.
5. Sales returns are permitted within 3 calendar days of the original sale, and cumulative returned quantities cannot exceed the original sold quantity.
6. Canceling a PBF invoice by `pemilik` is permitted only when none of the invoice's batches have been used in sales, returns, or stock opname.
7. Canceling a sales receipt is blocked if an active or approved return already references that sale.

## Revised tech stack

The stack prioritizes standard Go tooling and plain browser technologies so five developers can build and maintain their modules within one to two weeks without a Node.js build step.

| Layer | Technology | Purpose |
| :--- | :--- | :--- |
| Language and runtime | Go 1.27 | Single-binary backend application |
| HTTP router and middleware | Go `net/http` (`ServeMux`) | Method and path routing plus custom auth and 403 wrappers |
| HTML rendering | Go `html/template` + `embed.FS` | Server-rendered pages bundled into the binary |
| Password hashing | `golang.org/x/crypto/bcrypt` | One-way password hashing and verification |
| Sessions | PostgreSQL `sessions` + `HttpOnly` cookie | Per-request `status_aktif` enforcement and logout invalidation |
| CSS framework | Bootstrap 5.3 + `custom.css` | Layout grid, tables, forms, modals, and project styles per [DESIGN.MD](DESIGN.MD) |
| Icons | Phosphor Bold (`15px–17px`) | Navigation, control, and action icons per [DESIGN.MD](DESIGN.MD) |
| Client scripting | Plain Vanilla JS (`fetch()`, `localStorage`) | POS keyboard shortcuts (`F2`, `F8`, `Esc`), cart math, dynamic rows |
| Database | PostgreSQL 17 | ACID transactions, `NUMERIC(14,4)`, row-level locks per [DATABASE.MD](DATABASE.MD) |
| Database driver | `github.com/jackc/pgx/v5` (`pgxpool`) | Connection pooling and explicit transaction control |
| Exports and printing | Go `encoding/csv` + HTML `@media print` | CSV downloads, thermal receipts, and browser PDF printing |
| Configuration | Environment variables (`.env`) | `DB_DSN` for PostgreSQL, `PORT` for direct Go runs, `APP_HOST_PORT` for the Compose host port, and `APP_ENV` for runtime mode. The Compose app listens on container port `8080`. |
| Containers | Docker + Docker Compose (`Dockerfile`, `compose.yaml`) | Local PostgreSQL 17 with auto-initialized `migrations/` and single-binary deployment |

## Core business workflows

Every stock quantity in `batch`, `mutasi_stok`, `detail_penerimaan`, `detail_penjualan`, `detail_retur`, and `detail_stock_opname` is stored in the medicine's smallest base unit (`satuan_dasar_id`), such as tablet or capsule.

### Master medicines and packaging conversions

Each medicine has one base unit (`satuan_dasar_id`) and one or more entries in `obat_satuan` (such as box or strip) with a positive integer `faktor_konversi`. Only `pemilik` sets `harga_jual`, and prices across packaging levels are independent. Records with transaction history are deactivated via `status_aktif = false` (`ON DELETE RESTRICT` on all foreign keys prevents accidental hard deletes).

### PBF receiving and batch creation

Each supplier invoice (`penerimaan`) enforces uniqueness on `(id_pbf, no_faktur)`. Saving an invoice converts each line item's quantity into base units (`jumlah_faktur * faktor_konversi`), computes net unit cost to 4 decimal places (`harga_faktur / faktor_konversi`) into `batch.harga_beli_dasar`, creates one `batch` record (`1:1` with `detail_penerimaan`), and appends a positive `PENERIMAAN` row to `mutasi_stok`.

Batches with `tanggal_kedaluwarsa < tanggal_terima` are rejected. Batches expiring within 30 days are saved as `HAMPIR_ED` with a warning and excluded from POS allocation. Batches with more than 30 days remaining are saved as `AKTIF`.

`staf` can enter `harga_faktur` on a new invoice. After saving, staff can read invoice metadata and quantities, but persisted `harga_faktur`, line `subtotal`, `total_faktur`, and `harga_beli_dasar` are owner-only. Omit them from staff read queries and response structs, including the save response. A validation error may redisplay unsaved values the staff member just entered.

### POS sales and automatic FEFO

At preview and checkout, filter to `AKTIF` batches with `stok_tersisa > 0` and `tanggal_kedaluwarsa > CURRENT_DATE + 30`. For the selected packaging, use one batch per whole package whenever possible, visiting eligible batches by earliest expiry date and then lowest batch ID. Only after exhausting whole packages, fill any outstanding base units from the remaining stock in the same expiry order, splitting across batches if needed. A later eligible whole package therefore takes precedence over an earlier remainder. Cashiers cannot pick batches manually.

When the cashier confirms checkout, the backend opens a short PostgreSQL transaction, locks candidate `batch` rows in FEFO order with `SELECT ... FOR UPDATE`, and reruns eligibility and allocation against the locked stock. It deducts `stok_tersisa`, snapshots `harga_jual_saat_jual` and `harga_beli_saat_jual`, and writes `PENJUALAN` ledger entries in the same transaction. If the cart can no longer be filled, roll back the entire checkout.

### Sales returns and owner refund approval

Cashiers look up receipts within 3 calendar days, inspect physical condition (`kondisi_obat`), and submit a return with status `MENUNGGU_PERSETUJUAN`. Pending quantities reserve the returnable balance for each `detail_penjualan` line but do not affect stock or profit. Pending plus approved quantities cannot exceed the original base units sold on that line. Rejection releases the pending reservation.

When `pemilik` approves (`DISETUJUI`), validate the return's existing reservation against `detail_penjualan` without counting that return twice. Items marked `status_stok = 'MASUK_STOK'` increment `batch.stok_tersisa` and write a `RETUR_MASUK` ledger entry. Damaged or expired items marked `status_stok = 'TIDAK_MASUK_STOK'` stay out of `stok_tersisa` so FEFO cannot resell them. Rejecting (`DITOLAK`) leaves stock unchanged.

Return submission, approval, rejection, on-the-spot void, and historical cancellation lock the same `penjualan` row first and recheck sale and return status in that transaction. If a pending or approved return commits first, void or cancellation fails; if void or cancellation commits first, return submission fails. Approval rechecks that the return is still pending and applies the stock and ledger changes at most once.

### Blind stock opname

A `staf` or `pemilik` user opens a `DRAFT` session and records `stok_fisik` per batch. Count pages and their responses omit `stok_sistem` and any numeric variance for both roles. Only `pemilik` sees those values on the submitted review. When a count differs from system stock, require `alasan_selisih` before saving without revealing the variance. Each accepted revision increments `urutan_penghitungan`.

When a batch count starts, the server captures a stock-ledger checkpoint for that batch. When saving the count, lock the batch, reject the count if any stock movement occurred since the checkpoint, and require a fresh count for that batch only. Otherwise, save `stok_sistem` from the locked batch and compute `selisih_stok`. Do not block sales while a count is open.

Submitting the completed session sets `MENUNGGU_PERSETUJUAN`. Only `pemilik` may approve it. In one transaction, lock the affected batches, apply each latest accepted `selisih_stok` to current `stok_tersisa`, write `PENYESUAIAN_OPNAME` ledger entries for adjustments, and mark the session `SELESAI`. If any result would be negative, roll back all adjustments and leave the session pending so the owner can reject it for recount. Rejection (`DITOLAK`) never modifies stock.

### Acceptance scenarios

| Scenario | Expected result |
| --- | --- |
| Sell one 10-unit pack. An earlier eligible batch has 6 units and a later eligible batch has 12. | Allocate all 10 units from the later batch. If the later batch has only 4, allocate 6 from the earlier batch and 4 from the later batch. A batch expiring in exactly 30 days cannot contribute in either case. |
| A staff member records an invoice with purchase prices and then opens the saved invoice. | The invoice saves; the staff response contains metadata and quantities but no persisted prices or totals. The owner can read the saved amounts. |
| Two return requests for 3 units each arrive against the same 5-unit sale line. | Only one request reserves returnable quantity. A concurrent void cannot succeed if that request commits first. Rejecting the pending return allows another eligible request to reserve the released units. |
| A sale changes a batch between starting and saving its count. | Only that batch requires a recount. Neither a staff nor an owner count response reveals system stock or numeric variance. |
| A count of 9 is saved against system stock of 10. Two units are sold before approval. | Only the owner sees the submitted variance of −1. Approval applies it to current stock of 8, leaving 7. |

## Database schema

The complete 19-table PostgreSQL 17 schema, table definitions, constraints, indexes, ERD, and cardinality rules are maintained in **[DATABASE.MD](DATABASE.MD)** and MUST be followed strictly without altering foreign-key `ON DELETE RESTRICT` protections, base-unit quantity columns, or `NUMERIC(14,4)` cost precision.

## Project directory structure

The codebase uses a flat root entry point (`main.go`, `templates/`, `static/`) paired with module-scoped packages under `internal/` so all five developers work in isolated directories while embedding templates and static assets into a single Go binary.

```text
sistem-apotek/
├── main.go                 # //go:embed templates static, .env loading, pgxpool setup, ServeMux routes
├── go.mod / go.sum
├── Dockerfile              # Multi-stage Go build -> minimal Alpine runtime
├── .dockerignore           # Keeps local configuration and non-build files out of the context
├── compose.yaml            # PostgreSQL 17 (mounts migrations/) + app service
├── .env.example
├── .gitignore
├── internal/
│   ├── db/                 # pgxpool & transaction helpers (Steven)
│   ├── middleware/         # Session auth & RBAC 403 wrappers (Steven)
│   ├── view/               # Shared template parser & FuncMap (Steven)
│   ├── auth/               # Login, logout, shift_kasir (Steven & Zerafica)
│   ├── obat/               # Master obat, satuan, obat_satuan, employee accounts (Zerafica)
│   ├── penerimaan/         # PBF distributors, receiving invoices, 1:1 batch creation (Fachreza)
│   ├── pos/                # POS checkout, automatic FEFO, voids, receipts (Nadya)
│   ├── retur/              # Sales returns & owner refund approval (Steven)
│   ├── opname/             # Blind stock opname & owner approval (Steven)
│   └── laporan/            # Beranda, batch stock, 3-zone ED, batch audit, gross profit (Tyas)
├── templates/              # Shared templates and module pages
│   ├── layout.html         # Shared app shell for signed-in pages
│   ├── sidebar.html        # Role-aware navigation with the current page marked
│   ├── login.html          # Standalone sign-in page without app navigation
│   ├── mobile_topbar.html  # Top bar and Bootstrap offcanvas menu below 992px
│   ├── home.html           # Beranda dashboard with role-filtered operational summaries
│   ├── obat/
│   ├── karyawan/
│   ├── penerimaan/
│   ├── pos/
│   ├── retur/
│   ├── opname/
│   └── laporan/
├── static/                 # Local Bootstrap 5.3, Phosphor Bold, Roboto, custom.css, vanilla JS
│   ├── vendor/
│   │   ├── bootstrap/
│   │   ├── phosphor/
│   │   └── roboto/
│   ├── css/
│   │   └── custom.css      # Shared project styles loaded after Bootstrap
│   └── js/
├── migrations/
│   └── 001_init_schema.sql # 19-table schema, three roles, and one local demo owner
├── docs/                   # Project specification PDFs
├── preview/                # Static HTML/CSS/JS UI preview
└── references/
    └── go-class/           # Coding style and Go/template structure reference
```

## Team module ownership

| Team member | Assigned module | Owned tables and packages |
| :--- | :--- | :--- |
| Steven Bong (Chair) | Project scaffold, DB pool, RBAC middleware, returns, opname | `sessions`, `shift_kasir`, `retur_penjualan`, `detail_retur`, `stock_opname`, `detail_stock_opname` |
| Zerafica Adhyasti Dinar Patriawan | Master medicines, units, conversions, user accounts | `obat`, `satuan`, `obat_satuan`, `pengguna`, `role` |
| Tyas Sekar Arum | Home dashboard, batch stock reports, expiry alerts, gross profit | Read views across `batch`, `mutasi_stok`, `penjualan`, `detail_penjualan`, `retur_penjualan` |
| Muh Fachreza Nur Rachman | PBF distributors and batch receiving | `pbf`, `penerimaan`, `detail_penerimaan`, `batch`, `mutasi_stok` |
| Nadya Kamiila Rahmadiansyah | POS checkout, automatic FEFO, voids, receipts | `penjualan`, `item_penjualan`, `detail_penjualan`, `batch`, `mutasi_stok` |
