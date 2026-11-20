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
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/laporan"
	"github.com/minouse3/sistem-apotek/internal/middleware"
)

//go:embed templates static
var appFiles embed.FS

type signedInPageData struct {
	Title     string
	Active    string
	Role      string
	RoleLabel string
	UserName  string
	Initial   string
	Dashboard laporan.DashboardData
}

type batchStockPageData struct {
	Title     string
	Active    string
	Role      string
	RoleLabel string
	UserName  string
	Initial   string
	Keyword   string
	Status    string
	Batches   []laporan.BatchStock
}

type expiryPageData struct {
	Title     string
	Active    string
	Role      string
	RoleLabel string
	UserName  string
	Initial   string
	Batches   []laporan.ExpiringBatch
}

type grossProfitPageData struct {
	Title        string
	Active       string
	Role         string
	RoleLabel    string
	UserName     string
	Initial      string
	TanggalMulai string
	TanggalAkhir string
	Keyword      string
	Summary      laporan.GrossProfitReport
	Transactions []laporan.GrossProfitTransaction
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

	pages, err := template.ParseFS(
		appFiles,
		"templates/login.html",
		"templates/signed_in.html",
		"templates/layout.html",
	)
	if err != nil {
		return fmt.Errorf("parse application pages: %w", err)
	}

	stockPages, err := template.ParseFS(
		appFiles,
		"templates/layout.html",
		"templates/laporan/stok.html",
	)
	if err != nil {
		return fmt.Errorf("parse stock pages: %w", err)
	}

	expiryPages, err := template.ParseFS(
		appFiles,
		"templates/layout.html",
		"templates/laporan/hampir_ed.html",
	)
	if err != nil {
		return fmt.Errorf("parse expiry pages: %w", err)
	}

	grossProfitPages, err := template.ParseFS(
		appFiles,
		"templates/layout.html",
		"templates/laporan/laba_kotor.html",
	)
	if err != nil {
		return fmt.Errorf("parse gross profit pages: %w", err)
	}

	accounts := auth.New(
		pool,
		pages.Lookup("login.html"),
		os.Getenv("APP_ENV") == "production",
	)

	staticFiles, err := fs.Sub(appFiles, "static")
	if err != nil {
		return fmt.Errorf("load static files: %w", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /login", accounts.LoginPage)
	mux.HandleFunc("POST /login", accounts.SubmitLogin)

	// File statis aplikasi.
	mux.Handle(
		"GET /static/",
		http.StripPrefix("/static/", http.FileServerFS(staticFiles)),
	)

	// Mempertahankan alamat aset halaman login yang sudah ada.
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

	// Dashboard hanya dapat diakses setelah login.
	mux.Handle("GET /{$}", middleware.Require(
		accounts,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := middleware.CurrentUser(r.Context())
			if !ok {
				http.Error(w, "Sesi pengguna tidak ditemukan", http.StatusInternalServerError)
				return
			}

			roleLabel := "Pengguna"
			switch user.Role {
			case auth.RolePemilik:
				roleLabel = "Pemilik"
			case auth.RoleStaf:
				roleLabel = "Staf"
			case auth.RoleKasir:
				roleLabel = "Kasir"
			}

			initial := "U"
			nameParts := strings.Fields(user.Name)
			if len(nameParts) > 0 {
				runes := []rune(nameParts[0])
				if len(runes) > 0 {
					initial = strings.ToUpper(string(runes[0]))
				}
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)

			defer cancel()

			dashboard, err := laporan.GetDashboard(ctx, pool)
			if err != nil {
				log.Printf("get dashboard: %v", err)
				http.Error(w, "Gagal memuat data dashboard. Silakan coba lagi.", http.StatusInternalServerError)
				return
			}

			data := signedInPageData{
				Title:     "Dashboard",
				Active:    "dashboard",
				Role:      string(user.Role),
				RoleLabel: roleLabel,
				UserName:  user.Name,
				Initial:   initial,
				Dashboard: dashboard,
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")

			if err := pages.ExecuteTemplate(w, "layout", data); err != nil {
				log.Printf("render dashboard: %v", err)
			}
		}),
	))

	mux.Handle("GET /laporan/stok", middleware.Require(
		accounts,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := middleware.CurrentUser(r.Context())
			if !ok {
				http.Error(w, "Sesi pengguna tidak ditemukan", http.StatusInternalServerError)
				return
			}

			roleLabel := "Pengguna"
			switch user.Role {
			case auth.RolePemilik:
				roleLabel = "Pemilik"
			case auth.RoleStaf:
				roleLabel = "Staf"
			case auth.RoleKasir:
				roleLabel = "Kasir"
			}

			initial := "U"
			nameParts := strings.Fields(user.Name)
			if len(nameParts) > 0 {
				runes := []rune(nameParts[0])
				if len(runes) > 0 {
					initial = strings.ToUpper(string(runes[0]))
				}
			}

			keyword := strings.TrimSpace(r.URL.Query().Get("q"))
			status := strings.TrimSpace(r.URL.Query().Get("status"))

			switch status {
			case "", "AKTIF", "HAMPIR_ED", "KEDALUWARSA", "HABIS":
			default:
				http.Error(w, "Filter status tidak valid", http.StatusBadRequest)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			batches, err := laporan.GetBatchStock(ctx, pool, keyword, status)
			if err != nil {
				log.Printf("get batch stock: %v", err)
				http.Error(w, "Gagal memuat laporan stok.", http.StatusInternalServerError)
				return
			}

			data := batchStockPageData{
				Title:     "Laporan Stok Batch",
				Active:    "stok",
				Role:      string(user.Role),
				RoleLabel: roleLabel,
				UserName:  user.Name,
				Initial:   initial,
				Keyword:   keyword,
				Status:    status,
				Batches:   batches,
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")

			if err := stockPages.ExecuteTemplate(w, "layout", data); err != nil {
				log.Printf("render batch stock: %v", err)
			}
		}),
	))

	mux.Handle("GET /laporan/hampir-ed", middleware.Require(
		accounts,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := middleware.CurrentUser(r.Context())
			if !ok {
				http.Error(w, "Sesi pengguna tidak ditemukan", http.StatusInternalServerError)
				return
			}

			roleLabel := "Pengguna"
			switch user.Role {
			case auth.RolePemilik:
				roleLabel = "Pemilik"
			case auth.RoleStaf:
				roleLabel = "Staf"
			case auth.RoleKasir:
				roleLabel = "Kasir"
			}

			initial := "U"
			nameParts := strings.Fields(user.Name)
			if len(nameParts) > 0 {
				runes := []rune(nameParts[0])
				if len(runes) > 0 {
					initial = strings.ToUpper(string(runes[0]))
				}
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			batches, err := laporan.GetExpiringBatches(ctx, pool)
			if err != nil {
				log.Printf("get expiring batches: %v", err)
				http.Error(w, "Gagal memuat peringatan kedaluwarsa.", http.StatusInternalServerError)
				return
			}

			data := expiryPageData{
				Title:     "Batch Hampir ED",
				Active:    "hampir-ed",
				Role:      string(user.Role),
				RoleLabel: roleLabel,
				UserName:  user.Name,
				Initial:   initial,
				Batches:   batches,
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")

			if err := expiryPages.ExecuteTemplate(w, "layout", data); err != nil {
				log.Printf("render expiry page: %v", err)
			}
		}),
	))

	mux.Handle("GET /laporan/laba-kotor", middleware.Require(
		accounts,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := middleware.CurrentUser(r.Context())
			if !ok {
				http.Error(w, "Sesi pengguna tidak ditemukan", http.StatusInternalServerError)
				return
			}

			// Data keuangan hanya boleh diakses Pemilik.
			if user.Role != auth.RolePemilik {
				http.Error(w, "Akses ditolak", http.StatusForbidden)
				return
			}

			now := time.Now()
			firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)

			mulai := r.URL.Query().Get("mulai")
			akhir := r.URL.Query().Get("akhir")
			keyword := strings.TrimSpace(r.URL.Query().Get("q"))

			if mulai == "" {
				mulai = firstDay.Format("2006-01-02")
			}
			if akhir == "" {
				akhir = now.Format("2006-01-02")
			}

			tanggalMulai, err := time.Parse("2006-01-02", mulai)
			if err != nil {
				http.Error(w, "Tanggal mulai tidak valid.", http.StatusBadRequest)
				return
			}

			tanggalAkhir, err := time.Parse("2006-01-02", akhir)
			if err != nil {
				http.Error(w, "Tanggal akhir tidak valid.", http.StatusBadRequest)
				return
			}

			if tanggalMulai.After(tanggalAkhir) {
				http.Error(w, "Tanggal mulai tidak boleh melebihi tanggal akhir.", http.StatusBadRequest)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			summary, err := laporan.GetGrossProfit(
				ctx,
				pool,
				tanggalMulai,
				tanggalAkhir,
			)
			if err != nil {
				log.Printf("get gross profit: %v", err)
				http.Error(w, "Gagal memuat ringkasan laba kotor.", http.StatusInternalServerError)
				return
			}

			transactions, err := laporan.GetGrossProfitTransactions(
				ctx,
				pool,
				tanggalMulai,
				tanggalAkhir,
				keyword,
			)
			if err != nil {
				log.Printf("get gross profit transactions: %v", err)
				http.Error(w, "Gagal memuat rincian transaksi.", http.StatusInternalServerError)
				return
			}

			nameParts := strings.Fields(user.Name)
			initial := "U"
			if len(nameParts) > 0 {
				runes := []rune(nameParts[0])
				if len(runes) > 0 {
					initial = strings.ToUpper(string(runes[0]))
				}
			}

			data := grossProfitPageData{
				Title:        "Laporan Laba Kotor",
				Active:       "laba-kotor",
				Role:         string(user.Role),
				RoleLabel:    "Pemilik",
				UserName:     user.Name,
				Initial:      initial,
				TanggalMulai: mulai,
				TanggalAkhir: akhir,
				Keyword:      keyword,
				Summary:      summary,
				Transactions: transactions,
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")

			if err := grossProfitPages.ExecuteTemplate(w, "layout", data); err != nil {
				log.Printf("render gross profit: %v", err)
			}
		}),
	))

	mux.Handle("POST /logout", middleware.Require(
		accounts,
		http.HandlerFunc(accounts.Logout),
	))

	origin := http.NewCrossOriginProtection()
	origin.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Akses ditolak", http.StatusForbidden)
	}))

	log.Printf("server berjalan di :%s", port)
	return http.ListenAndServe(":"+port, origin.Handler(mux))
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
