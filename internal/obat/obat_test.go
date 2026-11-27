package obat

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/minouse3/sistem-apotek/internal/auth"
)

func TestBilanganBulat(t *testing.T) {
	tests := []struct {
		in   string
		min  int
		max  int
		want int
		ok   bool
	}{
		{"15.000", 0, maksHarga, 15000, true},
		{"0", 0, maksHarga, 0, true},
		{"1", 2, maksFaktor, 1, false},
		{"100.001", 2, maksFaktor, 100001, false},
		{"abc", 0, maksHarga, 0, false},
	}

	for _, tc := range tests {
		got, ok := bilanganBulat(tc.in, tc.min, tc.max)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("bilanganBulat(%q, %d, %d) = (%d, %v), want (%d, %v)", tc.in, tc.min, tc.max, got, ok, tc.want, tc.ok)
		}
	}
}

func TestEscapeLike(t *testing.T) {
	if got, want := escapeLike(`50%_a\b`), `50\%\_a\\b`; got != want {
		t.Fatalf("escapeLike = %q, want %q", got, want)
	}
}

func TestUsernameValid(t *testing.T) {
	if !usernameValid("kasir.rina_01") {
		t.Error("expected valid username")
	}
	for _, bad := range []string{"ab", "Admin", "user name", "user@mail"} {
		if usernameValid(bad) {
			t.Errorf("expected %q to be invalid", bad)
		}
	}
}

func TestBacaIsianObatAndPasswordValidation(t *testing.T) {
	form := url.Values{
		"kode_obat":       {"pct-500"},
		"nama_obat":       {"Paracetamol 500 mg"},
		"satuan_dasar_id": {"1"},
	}
	req := httptest.NewRequest(http.MethodPost, "/obat", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = req.ParseForm()

	isian, errs := bacaIsianObat(req)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if isian.Kode != "PCT-500" || isian.Rak != "RAK-UMUM" {
		t.Fatalf("unexpected normalized isian: %+v", isian)
	}

	pwForm := url.Values{"password": {"rahasia123"}, "konfirmasi_password": {"beda1234"}}
	pwReq := httptest.NewRequest(http.MethodPost, "/karyawan", strings.NewReader(pwForm.Encode()))
	pwReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = pwReq.ParseForm()
	pwErrs := map[string]string{}
	bacaPassword(pwReq, true, pwErrs)
	if pwErrs["konfirmasi_password"] == "" {
		t.Fatal("expected confirmation mismatch error")
	}
}

func TestHalamanRoleHelpers(t *testing.T) {
	if !(Halaman{User: auth.User{Role: auth.RolePemilik}}).Pemilik() {
		t.Error("pemilik should be Pemilik")
	}
	if !(Halaman{User: auth.User{Role: auth.RoleStaf}}).BolehKelolaObat() {
		t.Error("staf should be allowed to manage obat")
	}
	if (Halaman{User: auth.User{Role: auth.RoleKasir}}).BolehKelolaObat() {
		t.Error("kasir must not manage obat")
	}
}
