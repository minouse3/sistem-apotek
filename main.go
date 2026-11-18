package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/minouse3/sistem-apotek/internal/auth"
	"github.com/minouse3/sistem-apotek/internal/middleware"
)

//go:embed templates static
var appFiles embed.FS

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

	pages, err := template.ParseFS(appFiles, "templates/login.html", "templates/signed_in.html")
	if err != nil {
		return fmt.Errorf("parse authentication pages: %w", err)
	}
	accounts := auth.New(pool, pages.Lookup("login.html"), os.Getenv("APP_ENV") == "production")
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
		mux.HandleFunc("GET /login/"+asset, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFileFS(w, r, appFiles, asset)
		})
	}
	mux.Handle("GET /{$}", middleware.Require(accounts, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := pages.ExecuteTemplate(w, "signed_in.html", nil); err != nil {
			log.Printf("render signed-in page: %v", err)
		}
	})))
	mux.Handle("POST /logout", middleware.Require(accounts, http.HandlerFunc(accounts.Logout)))
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
