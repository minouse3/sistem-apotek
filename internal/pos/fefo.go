package pos

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInsufficientStock = errors.New("insufficient eligible stock")

type CandidateBatch struct {
	IDBatch         int
	IDObat          int
	NomorBatch      string
	ExpiryDate      string
	StokTersisa     int
	CostTenThousand int64 // NUMERIC(14,4) scaled by 10,000 for exact HPP rounding
}

type CartLineRequest struct {
	IDObatSatuan   int
	IDObat         int
	NamaObat       string
	NamaSatuan     string
	SatuanDasar    string
	Jumlah         int
	FaktorKonversi int
	HargaJual      int
}

type BatchAllocation struct {
	IDBatch         int
	NomorBatch      string
	ExpiryDate      string
	UrutanFEFO      int
	JumlahDasar     int
	CostTenThousand int64
	HPP             int
}

type LineAllocation struct {
	Line        CartLineRequest
	JumlahDasar int
	Subtotal    int
	TotalHPP    int
	Batches     []BatchAllocation
}

// ParseCostTenThousandths converts a NUMERIC(14,4) decimal string into scaled integer ten-thousandths.
func ParseCostTenThousandths(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	wholePart, fracPart, hasDot := strings.Cut(s, ".")
	var whole int64
	for _, ch := range wholePart {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("invalid cost %q", s)
		}
		whole = whole*10 + int64(ch-'0')
	}
	var frac int64
	if hasDot {
		for i := 0; i < 4; i++ {
			frac *= 10
			if i < len(fracPart) {
				ch := fracPart[i]
				if ch < '0' || ch > '9' {
					return 0, fmt.Errorf("invalid cost %q", s)
				}
				frac += int64(ch - '0')
			}
		}
	}
	val := whole*10000 + frac
	if neg {
		val = -val
	}
	return val, nil
}

// FormatCostTenThousandths formats scaled ten-thousandths back to a 4-decimal NUMERIC(14,4) string.
func FormatCostTenThousandths(v int64) string {
	return fmt.Sprintf("%d.%04d", v/10000, v%10000)
}

// ComputeHPP rounds (jumlahDasar * costTenThousandths) / 10000 to the nearest integer rupiah.
func ComputeHPP(jumlahDasar int, costTenThousand int64) int {
	total := int64(jumlahDasar) * costTenThousand
	return int((total + 5000) / 10000)
}

// AllocateFEFO allocates cart lines across candidate batches in FEFO order.
// Candidate batches must already be filtered to eligible AKTIF (>30d expiry, stok > 0)
// and sorted by earliest tanggal_kedaluwarsa ASC, id_batch ASC.
// Whole packages are allocated from single batches first; any remaining base units
// are then filled in FEFO order, splitting across batches if needed.
func AllocateFEFO(lines []CartLineRequest, batches []CandidateBatch) ([]LineAllocation, error) {
	if len(lines) == 0 {
		return nil, errors.New("empty cart")
	}

	// Track mutable remaining stock per batch across all lines in the same checkout.
	remaining := make([]int, len(batches))
	byObat := make(map[int][]int)
	for i, b := range batches {
		remaining[i] = b.StokTersisa
		byObat[b.IDObat] = append(byObat[b.IDObat], i)
	}

	results := make([]LineAllocation, 0, len(lines))
	for _, line := range lines {
		if line.Jumlah <= 0 || line.FaktorKonversi <= 0 {
			return nil, fmt.Errorf("invalid quantity or factor for %s", line.NamaObat)
		}

		idxs := byObat[line.IDObat]
		allocByBatch := make([]int, len(idxs))
		pkgsLeft := line.Jumlah

		// Phase 1: prefer a single batch per whole package in FEFO order.
		for pos, bIdx := range idxs {
			if pkgsLeft == 0 {
				break
			}
			wholeInBatch := remaining[bIdx] / line.FaktorKonversi
			if wholeInBatch <= 0 {
				continue
			}
			takePkgs := pkgsLeft
			if wholeInBatch < takePkgs {
				takePkgs = wholeInBatch
			}
			takeUnits := takePkgs * line.FaktorKonversi
			allocByBatch[pos] += takeUnits
			remaining[bIdx] -= takeUnits
			pkgsLeft -= takePkgs
		}

		// Phase 2: fill any remaining base units across batches in FEFO order.
		unitsLeft := pkgsLeft * line.FaktorKonversi
		if unitsLeft > 0 {
			for pos, bIdx := range idxs {
				if unitsLeft == 0 {
					break
				}
				if remaining[bIdx] <= 0 {
					continue
				}
				takeUnits := unitsLeft
				if remaining[bIdx] < takeUnits {
					takeUnits = remaining[bIdx]
				}
				allocByBatch[pos] += takeUnits
				remaining[bIdx] -= takeUnits
				unitsLeft -= takeUnits
			}
		}

		if unitsLeft > 0 {
			return nil, fmt.Errorf("%w: stok layak jual untuk %s tidak mencukupi", ErrInsufficientStock, line.NamaObat)
		}

		var batchAllocs []BatchAllocation
		var lineHPP int
		urutan := 1
		for pos, units := range allocByBatch {
			if units == 0 {
				continue
			}
			b := batches[idxs[pos]]
			hpp := ComputeHPP(units, b.CostTenThousand)
			lineHPP += hpp
			batchAllocs = append(batchAllocs, BatchAllocation{
				IDBatch:         b.IDBatch,
				NomorBatch:      b.NomorBatch,
				ExpiryDate:      b.ExpiryDate,
				UrutanFEFO:      urutan,
				JumlahDasar:     units,
				CostTenThousand: b.CostTenThousand,
				HPP:             hpp,
			})
			urutan++
		}

		results = append(results, LineAllocation{
			Line:        line,
			JumlahDasar: line.Jumlah * line.FaktorKonversi,
			Subtotal:    line.Jumlah * line.HargaJual,
			TotalHPP:    lineHPP,
			Batches:     batchAllocs,
		})
	}

	return results, nil
}
