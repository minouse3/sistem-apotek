package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/laporan"
	"github.com/minouse3/sistem-apotek/internal/middleware"
	"github.com/minouse3/sistem-apotek/internal/obat"
	"github.com/minouse3/sistem-apotek/internal/penerimaan"
	"github.com/minouse3/sistem-apotek/internal/view"
)

//go:embed templates static
var appFiles embed.FS

var appTemplates = []string{
	"templates/home.html",
	"templates/forbidden.html",
	"templates/laporan/stok.html",
	"templates/laporan/hampir_ed.html",
	"templates/laporan/batch_audit.html",
	"templates/laporan/laba_kotor.html",
	"templates/obat/daftar.html",
	"templates/obat/detail.html",
	"templates/obat/form.html",
	"templates/obat/kemasan.html",
	"templates/obat/satuan.html",
	"templates/obat/satuan_ubah.html",
	"templates/karyawan/daftar.html",
	"templates/karyawan/form.html",
	"templates/karyawan/akun.html",
	"templates/penerimaan/pbf.html",
	"templates/penerimaan/receiving_list.html",
	"templates/penerimaan/receiving_detail.html",
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("cannot load .env: check file permissions and syntax")
	}

	pool, err := openDatabase()
	if err != nil {
		return err
	}
	defer pool.Close()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	loginTmpl, err := template.ParseFS(appFiles, "templates/login.html")
	if err != nil {
		return fmt.Errorf("parse login template: %w", err)
	}

	renderer, err := view.NewRenderer(appFiles, appTemplates...)
	if err != nil {
		return fmt.Errorf("parse application pages: %w", err)
	}
	middleware.SetForbiddenHandler(func(w http.ResponseWriter, r *http.Request, user auth.User) {
		renderer.Render(w, http.StatusForbidden, "templates/forbidden.html", view.NewPageMeta(user, "Akses Ditolak", ""))
	})

	staticFiles, err := fs.Sub(appFiles, "static")
	if err != nil {
		return fmt.Errorf("load static files: %w", err)
	}

	accounts := auth.New(pool, loginTmpl.Lookup("login.html"), os.Getenv("APP_ENV") == "production")
	laporanHandler := laporan.NewHandler(pool, renderer)
	obatHandler := obat.NewHandler(pool, renderer)
	penerimaanHandler := penerimaan.NewHandler(pool, renderer)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", accounts.LoginPage)
	mux.HandleFunc("POST /login", accounts.SubmitLogin)
	for _, asset := range []string{
		"static/css/custom.css",
		"static/js/login.js",
		"static/vendor/bootstrap/bootstrap.min.css",
		"static/vendor/phosphor/eye-bold.svg",
		"static/vendor/phosphor/eye-slash-bold.svg",
		"static/vendor/roboto/roboto-latin-wght-normal.woff2",
	} {
		assetPath := asset
		mux.HandleFunc("GET /login/"+assetPath, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, appFiles, assetPath)
		})
	}

	// App static assets require an active session and exclude directories or metadata files.
	mux.Handle("GET /static/", middleware.Require(accounts, staticAssetHandler(staticFiles)))
	mux.Handle("POST /logout", middleware.Require(accounts, http.HandlerFunc(accounts.Logout)))
	laporanHandler.Register(mux, accounts)
	obatHandler.Register(mux, accounts)
	penerimaanHandler.Register(mux, accounts)

	origin := http.NewCrossOriginProtection()
	origin.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
	}))

	log.Printf("server berjalan di :%s", port)
	return http.ListenAndServe(":"+port, origin.Handler(mux))
}

func staticAssetHandler(files fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rel := strings.TrimPrefix(path.Clean("/"+strings.TrimPrefix(r.URL.Path, "/static/")), "/")
		if rel == "" || strings.HasPrefix(rel, ".") {
			http.NotFound(w, r)
			return
		}
		base := path.Base(rel)
		if strings.HasPrefix(base, ".") || base == "LICENSE" {
			http.NotFound(w, r)
			return
		}
		info, err := fs.Stat(files, rel)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, files, rel)
	})
}

func openDatabase() (*pgxpool.Pool, error) {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		return nil, errors.New("DB_DSN is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, errors.New("invalid DB_DSN: expected a PostgreSQL connection string")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("cannot connect to PostgreSQL using DB_DSN: %w", err)
	}
	return pool, nil
}
