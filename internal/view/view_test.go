package view

import (
	"testing"

	"github.com/minouse3/sistem-apotek/internal/auth"
)

func TestFormatRupiah(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "Rp0"},
		{750, "Rp750"},
		{1500000, "Rp1.500.000"},
		{-45000, "-Rp45.000"},
	}

	for _, tc := range tests {
		if got := FormatRupiah(tc.input); got != tc.want {
			t.Errorf("FormatRupiah(%d) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatRupiahDecimal(t *testing.T) {
	if got, want := FormatRupiahDecimal(180.5), "Rp180,5000"; got != want {
		t.Fatalf("FormatRupiahDecimal(180.5) = %q, want %q", got, want)
	}
	if got, want := FormatRupiahDecimal(12500.1234), "Rp12.500,1234"; got != want {
		t.Fatalf("FormatRupiahDecimal(12500.1234) = %q, want %q", got, want)
	}
}

func TestFormatSigned(t *testing.T) {
	if got := FormatSigned(1200); got != "+1.200" {
		t.Errorf("FormatSigned(1200) = %q, want +1.200", got)
	}
	if got := FormatSigned(-150); got != "-150" {
		t.Errorf("FormatSigned(-150) = %q, want -150", got)
	}
	if got := FormatSigned(0); got != "0" {
		t.Errorf("FormatSigned(0) = %q, want 0", got)
	}
}

func TestNewPageMeta(t *testing.T) {
	meta := NewPageMeta(auth.User{ID: 1, Name: "tyas sekar", Role: auth.RolePemilik}, "Beranda", "dashboard")
	if meta.RoleLabel != "Pemilik" || meta.Initial != "T" || meta.Role != "pemilik" {
		t.Fatalf("unexpected PageMeta: %+v", meta)
	}

	staffMeta := NewPageMeta(auth.User{ID: 2, Name: "   ", Role: auth.RoleStaf}, "Stok", "stok")
	if staffMeta.RoleLabel != "Staf" || staffMeta.Initial != "U" {
		t.Fatalf("unexpected staff PageMeta: %+v", staffMeta)
	}
}
