package laporan

import (
	"fmt"
	"strings"
)

const (
	ZoneMerah  = "MERAH"
	ZoneKuning = "KUNING"
	ZoneHijau  = "HIJAU"
	ZoneHabis  = "HABIS"
)

type ExpiryClassification struct {
	StatusEfektif string
	ZonaED        string
	Label         string
	Detail        string
	BadgeClass    string
}

// ClassifyExpiry derives effective batch status and 3-zone ED dynamically from remaining stock and days to expiry.
func ClassifyExpiry(daysUntilED int, stok int) ExpiryClassification {
	if stok <= 0 {
		return ExpiryClassification{
			StatusEfektif: "HABIS",
			ZonaED:        ZoneHabis,
			Label:         "Habis",
			Detail:        "Stok 0",
			BadgeClass:    "sw-badge-neutral",
		}
	}

	if daysUntilED < 0 {
		return ExpiryClassification{
			StatusEfektif: "KEDALUWARSA",
			ZonaED:        ZoneMerah,
			Label:         "Kedaluwarsa",
			Detail:        fmt.Sprintf("Lewat %d hari", -daysUntilED),
			BadgeClass:    "sw-badge-rose",
		}
	}

	if daysUntilED == 0 {
		return ExpiryClassification{
			StatusEfektif: "HAMPIR_ED",
			ZonaED:        ZoneMerah,
			Label:         "Tidak boleh dijual",
			Detail:        "ED hari ini",
			BadgeClass:    "sw-badge-rose",
		}
	}

	if daysUntilED <= 30 {
		return ExpiryClassification{
			StatusEfektif: "HAMPIR_ED",
			ZonaED:        ZoneMerah,
			Label:         "Tidak boleh dijual",
			Detail:        fmt.Sprintf("Sisa %d hari", daysUntilED),
			BadgeClass:    "sw-badge-rose",
		}
	}

	if daysUntilED <= 60 {
		return ExpiryClassification{
			StatusEfektif: "AKTIF",
			ZonaED:        ZoneKuning,
			Label:         "Prioritas FEFO",
			Detail:        fmt.Sprintf("Sisa %d hari", daysUntilED),
			BadgeClass:    "sw-badge-amber",
		}
	}

	return ExpiryClassification{
		StatusEfektif: "AKTIF",
		ZonaED:        ZoneHijau,
		Label:         "Aman",
		Detail:        fmt.Sprintf("Sisa %d hari", daysUntilED),
		BadgeClass:    "sw-badge-sage",
	}
}

func MutationLabel(jenis string) string {
	switch jenis {
	case "PENERIMAAN":
		return "Penerimaan PBF"
	case "PENJUALAN":
		return "Penjualan POS"
	case "PEMBATALAN_PENJUALAN":
		return "Pembatalan penjualan"
	case "PEMBATALAN_PENERIMAAN":
		return "Pembatalan penerimaan"
	case "RETUR_MASUK":
		return "Retur masuk stok"
	case "PENYESUAIAN_OPNAME":
		return "Penyesuaian opname"
	default:
		return jenis
	}
}

// Neutralize leading spreadsheet formula characters before writing to CSV.
func safeCSVCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t")
	if len(trimmed) > 0 {
		switch trimmed[0] {
		case '=', '+', '-', '@', '\t', '\r':
			return "'" + value
		}
	}
	return value
}
