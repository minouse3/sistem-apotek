# Sistem Apotek Sumber Waras

Web application for pharmacy inventory control, PBF supplier receiving, automatic FEFO point-of-sale checkout, sales returns, blind stock opname, and gross profit reporting.

## Tech stack

- Backend: Go 1.27 (`net/http` `ServeMux`, `html/template`, `embed.FS`)
- Database: PostgreSQL 17 with `github.com/jackc/pgx/v5` (`pgxpool`)
- Frontend: Bootstrap 5.3, custom CSS (`DESIGN.MD`), Phosphor Bold icons, plain Vanilla JS
- Containers: Docker and Docker Compose (`Dockerfile`, `compose.yaml`)

## Current repository state

- `main.go` verifies the PostgreSQL connection before serving Hello World at `GET /`.
- `migrations/001_init_schema.sql` defines all 19 PostgreSQL tables and seeds the three roles and one demo owner.
- `preview/index.html` provides a static local HTML preview with sample data (Beranda, Kasir / POS, Stok & Kedaluwarsa, and Laporan Laba Kotor).
- `internal/`, `templates/`, and `static/` contain module placeholders for team development.

## Getting started

### Prerequisites

- Go 1.27 or newer
- Docker and Docker Compose (or a local PostgreSQL 17 server)

### 1. Create `.env`

From the project root, copy `.env.example` to `.env` (`Copy-Item .env.example .env` in PowerShell):

```sh
cp .env.example .env
```

### 2. Run the application

For local Go development, start PostgreSQL and wait for it to accept TCP connections:

```sh
docker compose up -d --wait db
go run .
```

Go loads `.env` from the working directory. Variables already exported in the shell take precedence. The app requires a reachable PostgreSQL database and listens on `http://localhost:8080` by default; change `PORT` for a direct run. If you change the published `DB_PORT` or database credentials, update `DB_DSN` to match. URL-encode any reserved characters in the DSN's username or password.

To run both services in Docker instead:

```sh
docker compose up --build
```

Set `APP_HOST_PORT` to change the URL on your machine. The app still listens on `8080` inside the container and reaches PostgreSQL at `db:5432`, independent of `DB_PORT` and the local `DB_DSN`.

Compose applies `migrations/001_init_schema.sql` on the first database start with an empty volume. For a manually provisioned PostgreSQL 17 database, apply the SQL yourself and set `DB_DSN` to that database before running Go.

### Local demo account

- Username: `pemilik`
- Password: `SumberWaras-Demo-2026!`

These are public demo credentials. Use this seed only on a local demo database. The account is in PostgreSQL, but the app does not have a login page yet. The `kasir` and `staf` roles have no default accounts.

If your local database volume already exists, Compose will not rerun initialization. To keep your data, apply only the two seed `INSERT` statements at the end of [`001_init_schema.sql`](migrations/001_init_schema.sql) to that database. They leave existing roles and accounts with the same names unchanged.

### Open the static UI preview

Open `preview/index.html` directly in a browser. No backend or database is required for the preview.

## Project structure

- `main.go`, `go.mod`, `go.sum`: Go module entry point and HTTP server
- `Dockerfile`, `compose.yaml`, `.dockerignore`: Container build, local services, and build-context exclusions
- `internal/`: Module packages (`db`, `middleware`, `view`, `auth`, `obat`, `penerimaan`, `pos`, `retur`, `opname`, `laporan`)
- `templates/`: Server-rendered HTML templates
- `static/`: Vendored Bootstrap 5.3, Phosphor Bold, Roboto, custom CSS, and JS
- `migrations/`: PostgreSQL 17 schema and initial demo data
- `preview/`: Standalone HTML/CSS/JS UI preview
- `references/go-class/`: Go and `html/template` coding style reference
- `docs/`: Course specification PDFs

## Documentation

| File | Purpose |
| :--- | :--- |
| [PRD.MD](PRD.MD) | Product requirements, RBAC rules, workflows, and directory structure |
| [DATABASE.MD](DATABASE.MD) | 19-table PostgreSQL 17 schema, constraints, indexes, and ERD |
| [DESIGN.MD](DESIGN.MD) | Project design guide and Bootstrap 5.3 rules |
| [WEBSITE-GUIDELINES.md](WEBSITE-GUIDELINES.md) | Interface copy, validation, and code hygiene rules |
| [AGENTS.MD](AGENTS.MD) | Operating rules and skill requirements for AI coding agents |

## Team ownership

| Member | Module |
| :--- | :--- |
| Steven Bong (Chair) | Project scaffold, DB pool, RBAC middleware, returns, opname |
| Zerafica Adhyasti Dinar Patriawan | Master medicines, units, conversions, employee accounts |
| Tyas Sekar Arum | Home dashboard, batch stock reports, expiry alerts, gross profit |
| Muh Fachreza Nur Rachman | PBF distributors and batch receiving |
| Nadya Kamiila Rahmadiansyah | POS checkout, automatic FEFO, voids, receipts |
