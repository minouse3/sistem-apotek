package laporan

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestClassifyExpiryBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		days       int
		stok       int
		wantStatus string
		wantZona   string
	}{
		{"zero stock overrides expiry", 90, 0, "HABIS", ZoneHabis},
		{"expired yesterday", -1, 10, "KEDALUWARSA", ZoneMerah},
		{"expires today", 0, 5, "HAMPIR_ED", ZoneMerah},
		{"exactly 30 days blocked", 30, 12, "HAMPIR_ED", ZoneMerah},
		{"31 days enters kuning", 31, 12, "AKTIF", ZoneKuning},
		{"60 days stays kuning", 60, 12, "AKTIF", ZoneKuning},
		{"61 days enters hijau", 61, 12, "AKTIF", ZoneHijau},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyExpiry(tc.days, tc.stok)
			if got.StatusEfektif != tc.wantStatus || got.ZonaED != tc.wantZona {
				t.Fatalf("ClassifyExpiry(%d, %d) = (%s, %s), want (%s, %s)",
					tc.days, tc.stok, got.StatusEfektif, got.ZonaED, tc.wantStatus, tc.wantZona)
			}
		})
	}
}

func TestSafeCSVCellNeutralizesFormulas(t *testing.T) {
	if got := safeCSVCell("=CMD()"); got != "'=CMD()" {
		t.Errorf("got %q, want '=CMD()", got)
	}
	if got := safeCSVCell(" +1+1"); got != "' +1+1" {
		t.Errorf("got %q, want ' +1+1", got)
	}
	if got := safeCSVCell("Paracetamol 500 mg"); got != "Paracetamol 500 mg" {
		t.Errorf("got %q, want unchanged", got)
	}
}

func TestParseDateRange(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)

	reqDefault := httptest.NewRequest("GET", "/laporan/laba-kotor", nil)
	mulai, akhir, q, errMsg := parseDateRange(reqDefault, now)
	if errMsg != "" || mulai != "2026-11-01" || akhir != "2026-11-27" || q != "" {
		t.Fatalf("default range = (%s, %s, %s, %q)", mulai, akhir, q, errMsg)
	}

	reqInverted := httptest.NewRequest("GET", "/laporan/laba-kotor?mulai=2026-11-20&akhir=2026-11-10", nil)
	_, _, _, errMsg = parseDateRange(reqInverted, now)
	if errMsg == "" {
		t.Fatal("expected validation error when mulai > akhir")
	}

	reqBad := httptest.NewRequest("GET", "/laporan/laba-kotor?mulai=2026-13-45", nil)
	_, _, _, errMsg = parseDateRange(reqBad, now)
	if errMsg == "" {
		t.Fatal("expected validation error for invalid date format")
	}
}

func TestParseStockFilter(t *testing.T) {
	reqValid := httptest.NewRequest("GET", "/laporan/stok?q=PCM&zona=MERAH&status=HAMPIR_ED&rak=RAK-A1", nil)
	filter, errMsg := parseStockFilter(reqValid)
	if errMsg != "" || filter.Keyword != "PCM" || filter.Zona != "MERAH" || filter.Status != "HAMPIR_ED" || filter.Rak != "RAK-A1" {
		t.Fatalf("unexpected valid filter: %+v, err=%q", filter, errMsg)
	}

	reqInvalidZone := httptest.NewRequest("GET", "/laporan/stok?zona=UNGU", nil)
	_, errMsg = parseStockFilter(reqInvalidZone)
	if errMsg == "" {
		t.Fatal("expected error for unsupported zone")
	}
}
