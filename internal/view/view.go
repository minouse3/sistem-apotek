package view

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/minouse3/sistem-apotek/internal/auth"
)

type PageMeta struct {
	Title     string
	Active    string
	Role      string
	RoleLabel string
	UserName  string
	Initial   string
}

func NewPageMeta(user auth.User, title, active string) PageMeta {
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
	if parts := strings.Fields(user.Name); len(parts) > 0 {
		if runes := []rune(parts[0]); len(runes) > 0 {
			initial = strings.ToUpper(string(runes[0]))
		}
	}

	return PageMeta{
		Title:     title,
		Active:    active,
		Role:      string(user.Role),
		RoleLabel: roleLabel,
		UserName:  user.Name,
		Initial:   initial,
	}
}

func FormatInt(n int64) string {
	neg := n < 0
	var digits string
	if n == math.MinInt64 {
		digits = "9223372036854775808"
	} else {
		if neg {
			n = -n
		}
		digits = strconv.FormatInt(n, 10)
	}

	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	rem := len(digits) % 3
	for i := 0; i < len(digits); i++ {
		if i > 0 && (i-rem)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteByte(digits[i])
	}
	return b.String()
}

func FormatRupiah(n int64) string {
	if n < 0 {
		if n == math.MinInt64 {
			return "-Rp9.223.372.036.854.775.808"
		}
		return "-Rp" + FormatInt(-n)
	}
	return "Rp" + FormatInt(n)
}

func FormatRupiahDecimal(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	formatted := strconv.FormatFloat(v, 'f', 4, 64)
	wholeStr, fracStr, _ := strings.Cut(formatted, ".")
	whole, err := strconv.ParseInt(wholeStr, 10, 64)
	if err != nil {
		return "Rp" + formatted
	}
	prefix := "Rp"
	if neg {
		prefix = "-Rp"
	}
	return prefix + FormatInt(whole) + "," + fracStr
}

func FormatSigned(n int) string {
	if n > 0 {
		return "+" + FormatInt(int64(n))
	}
	return FormatInt(int64(n))
}

func FuncMap() template.FuncMap {
	return template.FuncMap{
		"rupiah":        FormatRupiah,
		"rupiahInt":     func(n int) string { return FormatRupiah(int64(n)) },
		"rupiahDecimal": FormatRupiahDecimal,
		"angka":         FormatInt,
		"angkaInt":      func(n int) string { return FormatInt(int64(n)) },
		"signed":        FormatSigned,
		"persen":        func(v float64) string { return strings.Replace(strconv.FormatFloat(v, 'f', 1, 64), ".", ",", 1) + "%" },
	}
}

type Renderer struct {
	pages map[string]*template.Template
}

var sharedShellFiles = []string{
	"templates/layout.html",
	"templates/sidebar.html",
	"templates/mobile_topbar.html",
}

func NewRenderer(files fs.FS, pageFiles ...string) (*Renderer, error) {
	pages := make(map[string]*template.Template, len(pageFiles))
	funcs := FuncMap()

	for _, page := range pageFiles {
		patterns := append(append([]string{}, sharedShellFiles...), page)
		tmpl, err := template.New("layout").Funcs(funcs).ParseFS(files, patterns...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", page, err)
		}
		pages[page] = tmpl
	}

	return &Renderer{pages: pages}, nil
}

func (r *Renderer) Render(w http.ResponseWriter, status int, page string, data any) {
	tmpl, ok := r.pages[page]
	if !ok {
		log.Printf("template not registered: %s", page)
		http.Error(w, "Halaman tidak ditemukan.", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("render %s: %v", page, err)
		http.Error(w, "Gagal menampilkan halaman. Coba lagi.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
