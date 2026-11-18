(() => {
  "use strict";

  // Fixed reference date so sample expiry zones remain deterministic across local previews.
  const REF_DATE_STR = "2026-11-18";
  const REF_DATE = new Date(`${REF_DATE_STR}T00:00:00`);

  const ROLES = {
    pemilik: {
      id: "pemilik",
      label: "Pemilik Apotek",
      userName: "dr. Ratna Sari",
      canSeeFinance: true,
      allowedRoutes: ["beranda", "pos", "stok", "laba"],
    },
    kasir: {
      id: "kasir",
      label: "Kasir",
      userName: "Nadya (Kasir 01)",
      canSeeFinance: false,
      allowedRoutes: ["beranda", "pos", "stok"],
    },
    staf: {
      id: "staf",
      label: "Staf Gudang",
      userName: "Fachreza (Staf)",
      canSeeFinance: false,
      allowedRoutes: ["beranda", "stok"],
    },
  };

  const NAV_GROUPS = [
    {
      label: "Operasional",
      items: [
        { id: "beranda", label: "Beranda", icon: "home" },
        { id: "pos", label: "Kasir / POS", icon: "cart" },
      ],
    },
    {
      label: "Persediaan",
      items: [
        { id: "stok", label: "Stok & Kedaluwarsa", icon: "layers" },
      ],
    },
    {
      label: "Keuangan",
      items: [
        { id: "laba", label: "Laporan Laba Kotor", icon: "chart" },
      ],
    },
  ];

  const ICONS = {
    home: '<i class="ph-bold ph-house" aria-hidden="true"></i>',
    cart: '<i class="ph-bold ph-shopping-cart" aria-hidden="true"></i>',
    layers: '<i class="ph-bold ph-stack" aria-hidden="true"></i>',
    chart: '<i class="ph-bold ph-chart-bar" aria-hidden="true"></i>',
    search: '<i class="ph-bold ph-magnifying-glass" aria-hidden="true"></i>',
    plus: '<i class="ph-bold ph-plus" aria-hidden="true"></i>',
    trash: '<i class="ph-bold ph-trash" aria-hidden="true"></i>',
  };

  const MEDICINES = [
    {
      id_obat: 1,
      kode_obat: "OB001",
      nama_obat: "Paracetamol 500 mg",
      kandungan: "Paracetamol 500 mg",
      golongan_obat: "Obat Bebas",
      satuan_dasar: "tablet",
      lokasi_rak: "RAK-A1",
      stok_minimum: 200,
      satuans: [
        { id_obat_satuan: 101, nama_satuan: "strip", faktor_konversi: 10, harga_jual: 3000, barcode: "8991001001012", boleh_dijual: true },
        { id_obat_satuan: 102, nama_satuan: "tablet", faktor_konversi: 1, harga_jual: 350, barcode: "8991001001019", boleh_dijual: true },
        { id_obat_satuan: 103, nama_satuan: "box", faktor_konversi: 100, harga_jual: 25000, barcode: "8991001001005", boleh_dijual: true },
      ],
    },
    {
      id_obat: 2,
      kode_obat: "OB002",
      nama_obat: "Amoxicillin 500 mg",
      kandungan: "Amoxicillin trihydrate 500 mg",
      golongan_obat: "Obat Keras",
      satuan_dasar: "kapsul",
      lokasi_rak: "RAK-B2",
      stok_minimum: 150,
      satuans: [
        { id_obat_satuan: 201, nama_satuan: "strip", faktor_konversi: 10, harga_jual: 8500, barcode: "8991001002019", boleh_dijual: true },
        { id_obat_satuan: 202, nama_satuan: "kapsul", faktor_konversi: 1, harga_jual: 950, barcode: "8991001002026", boleh_dijual: true },
        { id_obat_satuan: 203, nama_satuan: "box", faktor_konversi: 100, harga_jual: 78000, barcode: "8991001002002", boleh_dijual: true },
      ],
    },
    {
      id_obat: 3,
      kode_obat: "OB003",
      nama_obat: "CTM 4 mg",
      kandungan: "Chlorpheniramine maleate 4 mg",
      golongan_obat: "Obat Bebas Terbatas",
      satuan_dasar: "tablet",
      lokasi_rak: "RAK-A2",
      stok_minimum: 100,
      satuans: [
        { id_obat_satuan: 301, nama_satuan: "strip", faktor_konversi: 12, harga_jual: 2500, barcode: "8991001003016", boleh_dijual: true },
        { id_obat_satuan: 302, nama_satuan: "tablet", faktor_konversi: 1, harga_jual: 250, barcode: "8991001003023", boleh_dijual: true },
      ],
    },
    {
      id_obat: 4,
      kode_obat: "OB004",
      nama_obat: "Antasida Doen Suspensi 60 ml",
      kandungan: "Aluminium hidroksida, magnesium hidroksida",
      golongan_obat: "Obat Bebas",
      satuan_dasar: "botol",
      lokasi_rak: "RAK-A3",
      stok_minimum: 15,
      satuans: [
        { id_obat_satuan: 401, nama_satuan: "botol", faktor_konversi: 1, harga_jual: 9500, barcode: "8991001004013", boleh_dijual: true },
      ],
    },
    {
      id_obat: 5,
      kode_obat: "OB005",
      nama_obat: "Oralit 200 ml",
      kandungan: "Natrium klorida, kalium klorida, glukosa anhidrat",
      golongan_obat: "Obat Bebas",
      satuan_dasar: "sachet",
      lokasi_rak: "RAK-B1",
      stok_minimum: 50,
      satuans: [
        { id_obat_satuan: 501, nama_satuan: "sachet", faktor_konversi: 1, harga_jual: 1500, barcode: "8991001005010", boleh_dijual: true },
      ],
    },
    {
      id_obat: 6,
      kode_obat: "OB006",
      nama_obat: "Omeprazole 20 mg",
      kandungan: "Omeprazole 20 mg",
      golongan_obat: "Obat Keras",
      satuan_dasar: "kapsul",
      lokasi_rak: "RAK-B3",
      stok_minimum: 60,
      satuans: [
        { id_obat_satuan: 601, nama_satuan: "strip", faktor_konversi: 10, harga_jual: 12000, barcode: "8991001006017", boleh_dijual: true },
        { id_obat_satuan: 602, nama_satuan: "kapsul", faktor_konversi: 1, harga_jual: 1350, barcode: "8991001006024", boleh_dijual: true },
      ],
    },
  ];

  const BATCHES = [
    { id_batch: 11, id_obat: 1, nomor_batch: "PCM-2608", tanggal_kedaluwarsa: "2026-12-28", harga_beli_dasar: 180.0, stok_tersisa: 8, stok_karantina: 4, pbf: "PBF Sehat Sentosa", no_faktur: "FP/1026/009" },
    { id_batch: 12, id_obat: 1, nomor_batch: "PCM-2610", tanggal_kedaluwarsa: "2027-06-15", harga_beli_dasar: 195.0, stok_tersisa: 300, stok_karantina: 0, pbf: "PBF Sehat Sentosa", no_faktur: "FP/1026/018" },
    { id_batch: 21, id_obat: 2, nomor_batch: "AMX-2611", tanggal_kedaluwarsa: "2027-08-20", harga_beli_dasar: 520.0, stok_tersisa: 240, stok_karantina: 0, pbf: "PBF Medika Nusantara", no_faktur: "FP/1026/024" },
    { id_batch: 31, id_obat: 3, nomor_batch: "CTM-2605", tanggal_kedaluwarsa: "2026-12-08", harga_beli_dasar: 85.0, stok_tersisa: 80, stok_karantina: 12, pbf: "PBF Sehat Sentosa", no_faktur: "FP/0926/114" },
    { id_batch: 32, id_obat: 3, nomor_batch: "CTM-2612", tanggal_kedaluwarsa: "2027-09-10", harga_beli_dasar: 90.0, stok_tersisa: 120, stok_karantina: 0, pbf: "PBF Medika Nusantara", no_faktur: "FP/1126/003" },
    { id_batch: 41, id_obat: 4, nomor_batch: "ANT-2609", tanggal_kedaluwarsa: "2027-03-14", harga_beli_dasar: 6100.0, stok_tersisa: 11, stok_karantina: 0, pbf: "PBF Sehat Sentosa", no_faktur: "FP/1026/018" },
    { id_batch: 51, id_obat: 5, nomor_batch: "ORS-2602", tanggal_kedaluwarsa: "2026-11-05", harga_beli_dasar: 700.0, stok_tersisa: 24, stok_karantina: 4, pbf: "PBF Kimia Farma Trading", no_faktur: "FP/0726/088" },
    { id_batch: 61, id_obat: 6, nomor_batch: "OMP-2607", tanggal_kedaluwarsa: "2026-12-10", harga_beli_dasar: 750.0, stok_tersisa: 30, stok_karantina: 0, pbf: "PBF Medika Nusantara", no_faktur: "FP/0926/099" },
  ];

  const MUTATIONS = [
    { id_mutasi: 101, id_batch: 11, jenis_mutasi: "PENERIMAAN", jumlah_dasar: 200, ref_nomor: "FP/1026/009", waktu: "2026-10-12 09:10", note: "Penerimaan 2 box × 100 tablet" },
    { id_mutasi: 102, id_batch: 11, jenis_mutasi: "PENJUALAN", jumlah_dasar: -190, ref_nomor: "PJ-20261115-014", waktu: "2026-11-15 14:20", note: "Penjualan 19 strip × 10 tablet" },
    { id_mutasi: 103, id_batch: 11, jenis_mutasi: "PENYESUAIAN_OPNAME", jumlah_dasar: -2, ref_nomor: "SO-202611-01", waktu: "2026-11-16 17:45", note: "Selisih opname -2 tablet (BARANG_RUSAK)" },
    { id_mutasi: 104, id_batch: 12, jenis_mutasi: "PENERIMAAN", jumlah_dasar: 300, ref_nomor: "FP/1026/018", waktu: "2026-11-02 10:05", note: "Penerimaan 3 box × 100 tablet" },
    { id_mutasi: 105, id_batch: 21, jenis_mutasi: "PENJUALAN", jumlah_dasar: -20, ref_nomor: "PJ-20261118-001", waktu: "2026-11-18 09:00", note: "Penjualan 2 strip Amoxicillin 500 mg" },
    { id_mutasi: 106, id_batch: 41, jenis_mutasi: "RETUR_MASUK", jumlah_dasar: 1, ref_nomor: "RTR-20261117-002", waktu: "2026-11-17 11:30", note: "Retur 1 botol Antasida segel utuh" },
    { id_mutasi: 107, id_batch: 31, jenis_mutasi: "PENERIMAAN", jumlah_dasar: 120, ref_nomor: "FP/0926/114", waktu: "2026-09-18 13:15", note: "Penerimaan 10 strip × 12 tablet" },
    { id_mutasi: 108, id_batch: 31, jenis_mutasi: "PENJUALAN", jumlah_dasar: -40, ref_nomor: "PJ-20261110-008", waktu: "2026-11-10 16:05", note: "Penjualan 40 tablet sebelum batas 30 hari ED" },
  ];

  // Sample historical sales records with frozen per-batch HPP snapshots.
  const SALES_LEDGER = [
    {
      nomor_transaksi: "PJ-20261118-001",
      waktu: "2026-11-18 09:00",
      kasir: "Nadya (Kasir 01)",
      metode: "TUNAI",
      total_penjualan: 17000,
      total_hpp: 10400,
      batches: [
        { nama_obat: "Amoxicillin 500 mg", nomor_batch: "AMX-2611", jumlah_dasar: 20, satuan_dasar: "kapsul", harga_beli_saat_jual: 520.0, hpp: 10400 },
      ],
    },
    {
      nomor_transaksi: "PJ-20261115-014",
      waktu: "2026-11-15 14:20",
      kasir: "Nadya (Kasir 01)",
      metode: "QRIS",
      total_penjualan: 57000,
      total_hpp: 34200,
      batches: [
        { nama_obat: "Paracetamol 500 mg", nomor_batch: "PCM-2608", jumlah_dasar: 190, satuan_dasar: "tablet", harga_beli_saat_jual: 180.0, hpp: 34200 },
      ],
    },
    {
      nomor_transaksi: "PJ-20261110-008",
      waktu: "2026-11-10 16:05",
      kasir: "Nadya (Kasir 01)",
      metode: "DEBIT",
      total_penjualan: 9000,
      total_hpp: 3400,
      batches: [
        { nama_obat: "CTM 4 mg", nomor_batch: "CTM-2605", jumlah_dasar: 40, satuan_dasar: "tablet", harga_beli_saat_jual: 85.0, hpp: 3400 },
      ],
    },
  ];

  const PBF_BILLS = [
    { no_faktur: "FP/1026/018", pbf: "PBF Sehat Sentosa", jatuh_tempo: "2026-11-15", total: 555000, status: "TERLAMBAT" },
    { no_faktur: "FP/1026/024", pbf: "PBF Medika Nusantara", jatuh_tempo: "2026-11-20", total: 333000, status: "H_MIN_3" },
  ];

  const state = {
    role: "pemilik",
    route: "beranda",
    zoneFilter: "ALL",
    stockQuery: "",
    stockRackFilter: "ALL",
    pos: {
      selectedMedId: 1,
      selectedSatuanId: 101,
      qtyRaw: "1",
      paymentMethod: "TUNAI",
      cashRaw: "10000",
      cart: [{ id_obat: 1, id_obat_satuan: 101, qty: 1 }],
      notice: null,
      lastSale: null,
    },
  };

  function formatRp(amount, decimals = 0) {
    const val = Number(amount || 0);
    return "Rp " + val.toLocaleString("id-ID", {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    });
  }

  function formatDateID(dateStr) {
    const d = new Date(`${dateStr}T00:00:00`);
    return d.toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" });
  }

  function daysUntilED(dateStr) {
    const ed = new Date(`${dateStr}T00:00:00`);
    return Math.round((ed - REF_DATE) / (1000 * 60 * 60 * 24));
  }

  function getBatchZone(batch) {
    const days = daysUntilED(batch.tanggal_kedaluwarsa);
    if (days < 0) {
      return { zone: "MERAH", days, label: "Kedaluwarsa", badgeClass: "sw-badge-rose", detail: `Lewat ${Math.abs(days)} hari` };
    }
    if (days <= 30) {
      return { zone: "MERAH", days, label: "Tidak boleh dijual", badgeClass: "sw-badge-rose", detail: `${days} hari` };
    }
    if (days <= 60) {
      return { zone: "KUNING", days, label: "Prioritas jual", badgeClass: "sw-badge-amber", detail: `${days} hari` };
    }
    return { zone: "HIJAU", days, label: "Aman", badgeClass: "sw-badge-sage", detail: `${days} hari` };
  }

  function getMedicineById(id) {
    return MEDICINES.find((m) => m.id_obat === Number(id));
  }

  function getSatuanById(med, satuanId) {
    return med.satuans.find((s) => s.id_obat_satuan === Number(satuanId));
  }

  // Only batches with remaining sellable stock and more than 30 days shelf life may enter POS FEFO.
  function getEligibleBatches(idObat) {
    return BATCHES
      .filter((b) => b.id_obat === idObat && b.stok_tersisa > 0 && daysUntilED(b.tanggal_kedaluwarsa) > 30)
      .sort((a, b) => {
        if (a.tanggal_kedaluwarsa !== b.tanggal_kedaluwarsa) {
          return a.tanggal_kedaluwarsa.localeCompare(b.tanggal_kedaluwarsa);
        }
        return a.id_batch - b.id_batch;
      });
  }

  function getSellableStockBase(idObat) {
    return getEligibleBatches(idObat).reduce((sum, b) => sum + b.stok_tersisa, 0);
  }

  // Whole-package batches are prioritized before partial remainder batches when faktor_konversi > 1.
  function previewFefoAllocation(cartItems) {
    const virtualStock = new Map();
    BATCHES.forEach((b) => virtualStock.set(b.id_batch, b.stok_tersisa));

    const lineResults = [];
    let hasInsufficient = false;

    for (const item of cartItems) {
      const med = getMedicineById(item.id_obat);
      const sat = getSatuanById(med, item.id_obat_satuan);
      const neededBase = item.qty * sat.faktor_konversi;

      const eligibleAll = BATCHES
        .filter((b) => b.id_obat === med.id_obat && (virtualStock.get(b.id_batch) || 0) > 0 && daysUntilED(b.tanggal_kedaluwarsa) > 30)
        .sort((a, b) => {
          if (a.tanggal_kedaluwarsa !== b.tanggal_kedaluwarsa) {
            return a.tanggal_kedaluwarsa.localeCompare(b.tanggal_kedaluwarsa);
          }
          return a.id_batch - b.id_batch;
        });

      const totalAvail = eligibleAll.reduce((acc, b) => acc + (virtualStock.get(b.id_batch) || 0), 0);

      let orderedBatches = eligibleAll;
      let skippedLooseBatch = null;
      if (sat.faktor_konversi > 1) {
        const wholeBatches = eligibleAll.filter((b) => (virtualStock.get(b.id_batch) || 0) >= sat.faktor_konversi);
        const looseBatches = eligibleAll.filter((b) => (virtualStock.get(b.id_batch) || 0) < sat.faktor_konversi);
        if (looseBatches.length > 0 && wholeBatches.length > 0 && eligibleAll[0].id_batch === looseBatches[0].id_batch) {
          skippedLooseBatch = {
            nomor_batch: looseBatches[0].nomor_batch,
            sisa: virtualStock.get(looseBatches[0].id_batch),
          };
        }
        orderedBatches = [...wholeBatches, ...looseBatches];
      }

      let remaining = neededBase;
      const allocations = [];
      let order = 1;

      for (const batch of orderedBatches) {
        if (remaining <= 0) break;
        const avail = virtualStock.get(batch.id_batch) || 0;
        if (avail <= 0) continue;
        const take = Math.min(avail, remaining);
        allocations.push({
          id_batch: batch.id_batch,
          nomor_batch: batch.nomor_batch,
          tanggal_kedaluwarsa: batch.tanggal_kedaluwarsa,
          days: daysUntilED(batch.tanggal_kedaluwarsa),
          urutan_fefo: order++,
          jumlah_dasar: take,
          harga_beli_saat_jual: batch.harga_beli_dasar,
          hpp: Math.round(take * batch.harga_beli_dasar),
        });
        virtualStock.set(batch.id_batch, avail - take);
        remaining -= take;
      }

      const sufficient = remaining === 0;
      if (!sufficient) hasInsufficient = true;

      lineResults.push({
        med,
        sat,
        qty: item.qty,
        neededBase,
        totalAvail,
        sufficient,
        subtotal: item.qty * sat.harga_jual,
        allocations,
        skippedLooseBatch,
      });
    }

    const subtotal = lineResults.reduce((sum, l) => sum + l.subtotal, 0);
    return { lines: lineResults, subtotal, hasInsufficient };
  }

  // Only Tunai rounds to nearest Rp100; QRIS and Debit charge exact subtotal.
  function computePayment(subtotal, method, cashRaw) {
    if (method !== "TUNAI") {
      return { totalBayar: subtotal, pembulatan: 0, kembalian: 0, cashValid: true, cashError: "" };
    }
    const totalBayar = Math.round(subtotal / 100) * 100;
    const pembulatan = totalBayar - subtotal;
    const trimmed = String(cashRaw).trim();
    const parsed = Number(trimmed);

    if (trimmed === "" || !Number.isInteger(parsed) || parsed < 0) {
      return {
        totalBayar,
        pembulatan,
        kembalian: 0,
        cashValid: false,
        cashError: "Masukkan bilangan bulat Rupiah mulai dari 0.",
      };
    }
    if (parsed < totalBayar) {
      return {
        totalBayar,
        pembulatan,
        kembalian: 0,
        cashValid: false,
        cashError: `Nominal kurang ${formatRp(totalBayar - parsed)} dari total bayar ${formatRp(totalBayar)}.`,
      };
    }
    return {
      totalBayar,
      pembulatan,
      kembalian: parsed - totalBayar,
      cashValid: true,
      cashError: "",
    };
  }

  function validatePosQty(qtyRaw, med, sat) {
    const trimmed = String(qtyRaw).trim();
    const parsed = Number(trimmed);
    if (trimmed === "" || !Number.isInteger(parsed) || parsed < 1 || parsed > 9999) {
      return { valid: false, qty: 0, neededBase: 0, error: "Masukkan bilangan bulat antara 1 dan 9.999." };
    }
    const avail = getSellableStockBase(med.id_obat);
    const neededBase = parsed * sat.faktor_konversi;
    if (avail === 0) {
      return { valid: false, qty: parsed, neededBase, error: "Stok layak jual kosong atau semua batch masuk Zona Merah (ED ≤ 30 hari)." };
    }
    if (neededBase > avail) {
      return {
        valid: false,
        qty: parsed,
        neededBase,
        error: `Stok layak jual tersisa ${avail} ${med.satuan_dasar}. Kurangi jumlah atau pilih satuan yang lebih kecil.`,
      };
    }
    return { valid: true, qty: parsed, neededBase, error: "" };
  }

  function renderPageHeader({ title, subtitle = "", actionsHtml = "" }) {
    const roleObj = ROLES[state.role];
    return `
      <div class="sw-page-header">
        <div>
          <h1 class="sw-page-title">${title}</h1>
          ${subtitle ? `<p class="sw-page-subtitle">${subtitle}</p>` : ""}
        </div>
        <div class="sw-header-actions">
          <div class="sw-utility-group">
            <label for="role-switcher" class="sw-meta">Peran:</label>
            <select id="role-switcher" class="form-select sw-role-select" aria-label="Pilih peran pengguna">
              <option value="pemilik" ${state.role === "pemilik" ? "selected" : ""}>Pemilik Apotek</option>
              <option value="kasir" ${state.role === "kasir" ? "selected" : ""}>Kasir</option>
              <option value="staf" ${state.role === "staf" ? "selected" : ""}>Staf Gudang</option>
            </select>
            <span class="sw-meta d-none d-sm-inline">${roleObj.userName}</span>
          </div>
          ${actionsHtml}
        </div>
      </div>
    `;
  }

  function renderSidebarInto(containerId) {
    const container = document.getElementById(containerId);
    if (!container) return;
    const roleObj = ROLES[state.role];

    let html = "";
    for (const group of NAV_GROUPS) {
      const visibleItems = group.items.filter((item) => roleObj.allowedRoutes.includes(item.id));
      if (visibleItems.length === 0) continue;

      html += `<div class="sw-nav-group">`;
      html += `<div class="sw-group-label sw-nav-group-label">${group.label}</div>`;
      html += `<ul class="sw-nav-list">`;
      for (const item of visibleItems) {
        const isActive = state.route === item.id;
        html += `
          <li>
            <button type="button" class="sw-nav-link ${isActive ? "is-active" : ""}" data-route="${item.id}">
              ${ICONS[item.icon] || ""}
              <span>${item.label}</span>
            </button>
          </li>`;
      }
      html += `</ul></div>`;
    }
    container.innerHTML = html;

    container.querySelectorAll("[data-route]").forEach((btn) => {
      btn.addEventListener("click", () => {
        navigateTo(btn.dataset.route);
        const offcanvasEl = document.getElementById("mobileSidebarDrawer");
        if (offcanvasEl && window.bootstrap?.Offcanvas) {
          window.bootstrap.Offcanvas.getInstance(offcanvasEl)?.hide();
        }
      });
    });
  }

  function navigateTo(route) {
    state.route = route;
    renderAll();
  }

  function renderForbiddenView(routeLabel) {
    const roleObj = ROLES[state.role];
    return `
      ${renderPageHeader({
        title: "Akses ditolak",
        subtitle: `HTTP 403 Forbidden · Peran aktif: ${roleObj.label}`,
      })}
      <div class="sw-panel" style="max-width: 640px;">
        <div class="sw-badge sw-badge-rose">403 Forbidden</div>
        <h2 class="sw-section-title mt-3">${routeLabel} tidak tersedia untuk peran ${roleObj.label}</h2>
        <p class="mt-2 mb-3" style="font-size: 14px; color: var(--sw-muted);">
          Pada sistem produksi, permintaan ke rute ini ditolak oleh middleware server dan seluruh kolom finansial
          (<code>harga_beli_dasar</code>, <code>total_hpp</code>, laba kotor, dan nilai aset) dihapus dari respons.
        </p>
        <div class="d-flex flex-wrap gap-2">
          <button type="button" class="sw-btn sw-btn-primary" data-goto="beranda">Kembali ke Beranda</button>
          <button type="button" class="sw-btn sw-btn-secondary" data-switch-role="pemilik">Beralih ke peran Pemilik</button>
        </div>
      </div>
    `;
  }

  function renderBeranda() {
    const canFinance = ROLES[state.role].canSeeFinance;
    const totalOmzet = SALES_LEDGER.reduce((s, x) => s + x.total_penjualan, 0);
    const totalHpp = SALES_LEDGER.reduce((s, x) => s + x.total_hpp, 0);
    const labaKotor = totalOmzet - totalHpp;
    const marginPct = totalOmzet > 0 ? ((labaKotor / totalOmzet) * 100).toFixed(1) : "0.0";

    const attentionBatches = BATCHES
      .map((b) => ({ batch: b, med: getMedicineById(b.id_obat), z: getBatchZone(b) }))
      .filter((x) => x.z.zone === "MERAH" || x.z.zone === "KUNING")
      .sort((a, b) => a.z.days - b.z.days);

    const lowStockMeds = MEDICINES.filter((m) => getSellableStockBase(m.id_obat) < m.stok_minimum);
    const recentMutations = [...MUTATIONS].slice(-4).reverse();
    const unpaidBillsTotal = PBF_BILLS.reduce((s, b) => s + b.total, 0);

    return `
      ${renderPageHeader({
        title: "Beranda",
        subtitle: "Data contoh · 18 November 2026",
        actionsHtml: `
          ${ROLES[state.role].allowedRoutes.includes("pos")
            ? `<button type="button" class="sw-btn sw-btn-primary" data-goto="pos">${ICONS.cart} Buka kasir POS</button>`
            : ""}
          <button type="button" class="sw-btn sw-btn-secondary" data-goto="stok">${ICONS.layers} Periksa stok batch</button>
        `,
      })}

      <div class="sw-split-grid">
        <div class="sw-stack">
          ${canFinance ? `
            <section class="sw-panel" aria-label="Ringkasan penjualan contoh">
              <div class="sw-panel-header">
                <div>
                  <h2 class="sw-section-title">Penjualan tercatat (contoh)</h2>
                  <div class="sw-meta mt-1">Akumulasi dari ${SALES_LEDGER.length} nota pada data contoh</div>
                </div>
                <button type="button" class="sw-btn sw-btn-secondary" data-goto="laba">Buka laporan laba</button>
              </div>
              <div class="row g-3">
                <div class="col-6 col-sm-3">
                  <div class="sw-meta">Penjualan bersih</div>
                  <div class="sw-figure mt-1">${formatRp(totalOmzet)}</div>
                </div>
                <div class="col-6 col-sm-3">
                  <div class="sw-meta">HPP batch terjual</div>
                  <div class="sw-figure mt-1">${formatRp(totalHpp)}</div>
                </div>
                <div class="col-6 col-sm-3">
                  <div class="sw-meta">Laba kotor</div>
                  <div class="sw-figure mt-1" style="color: var(--sw-forest);">${formatRp(labaKotor)}</div>
                </div>
                <div class="col-6 col-sm-3">
                  <div class="sw-meta">Margin</div>
                  <div class="sw-figure mt-1">${marginPct}%</div>
                </div>
              </div>
            </section>
          ` : `
            <section class="sw-panel" aria-label="Ringkasan operasional">
              <div class="sw-panel-header">
                <div>
                  <h2 class="sw-section-title">Ringkasan persediaan (contoh)</h2>
                  <div class="sw-meta mt-1">Tampilan nonfinansial untuk peran ${ROLES[state.role].label}</div>
                </div>
              </div>
              <div class="row g-3">
                <div class="col-6 col-sm-4">
                  <div class="sw-meta">Nota tercatat</div>
                  <div class="sw-figure mt-1">${SALES_LEDGER.length}</div>
                </div>
                <div class="col-6 col-sm-4">
                  <div class="sw-meta">Batch perlu perhatian</div>
                  <div class="sw-figure mt-1">${attentionBatches.length}</div>
                </div>
                <div class="col-12 col-sm-4">
                  <div class="sw-meta">Obat di bawah minimum</div>
                  <div class="sw-figure mt-1">${lowStockMeds.length}</div>
                </div>
              </div>
            </section>
          `}

          <section class="sw-panel" aria-label="Persediaan perlu perhatian">
            <div class="sw-panel-header">
              <div>
                <h2 class="sw-section-title">Persediaan perlu perhatian</h2>
                <div class="sw-meta mt-1">Pisahkan batch Zona Merah (&le; 30 hari) dari rak penjualan</div>
              </div>
              <button type="button" class="sw-btn sw-btn-secondary" data-goto="stok">Lihat semua batch</button>
            </div>
            <div class="sw-table-wrap">
              <table class="sw-table">
                <thead>
                  <tr>
                    <th>Obat / batch</th>
                    <th>Lokasi rak</th>
                    <th>Tanggal ED</th>
                    <th class="text-end">Stok layak jual</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  ${attentionBatches.map(({ batch, med, z }) => `
                    <tr>
                      <td>
                        <div class="fw-semibold">${med.nama_obat}</div>
                        <div class="sw-meta">${batch.nomor_batch} · ${med.kode_obat}</div>
                      </td>
                      <td>${med.lokasi_rak}</td>
                      <td class="sw-num">${formatDateID(batch.tanggal_kedaluwarsa)}</td>
                      <td class="text-end sw-num">${batch.stok_tersisa} ${med.satuan_dasar}</td>
                      <td><span class="sw-badge ${z.badgeClass}">${z.label} · ${z.detail}</span></td>
                    </tr>
                  `).join("")}
                </tbody>
              </table>
            </div>
          </section>

          <section class="sw-panel" aria-label="Mutasi stok terbaru">
            <div class="sw-panel-header">
              <h2 class="sw-section-title">Mutasi stok terbaru (contoh)</h2>
            </div>
            <div class="sw-table-wrap">
              <table class="sw-table">
                <thead>
                  <tr>
                    <th>Waktu</th>
                    <th>Keterangan</th>
                    <th class="text-end">Perubahan</th>
                    <th>Referensi</th>
                  </tr>
                </thead>
                <tbody>
                  ${recentMutations.map((m) => {
                    const batch = BATCHES.find((b) => b.id_batch === m.id_batch);
                    const med = batch ? getMedicineById(batch.id_obat) : null;
                    const sign = m.jumlah_dasar > 0 ? `+${m.jumlah_dasar}` : `${m.jumlah_dasar}`;
                    return `
                      <tr>
                        <td class="sw-num sw-meta">${m.waktu.slice(5)}</td>
                        <td>
                          <div class="fw-medium">${m.note}</div>
                          <div class="sw-meta">${med ? med.nama_obat : ""} · ${batch ? batch.nomor_batch : ""}</div>
                        </td>
                        <td class="text-end sw-num fw-semibold">${sign} ${med ? med.satuan_dasar : ""}</td>
                        <td class="sw-meta">${m.ref_nomor}</td>
                      </tr>
                    `;
                  }).join("")}
                </tbody>
              </table>
            </div>
          </section>
        </div>

        <div class="sw-stack">
          <section class="sw-panel">
            <div class="sw-panel-header">
              <h2 class="sw-section-title">Peringatan stok minimum</h2>
              <span class="sw-meta sw-num">${lowStockMeds.length} obat</span>
            </div>
            ${lowStockMeds.length === 0 ? `<p class="sw-meta mb-0">Semua obat berada di atas batas stok minimum.</p>` : `
              <ul class="sw-rule-list">
                ${lowStockMeds.map((m) => `
                  <li class="sw-rule-item">
                    <div>
                      <div style="font-size: 13px; font-weight: 600;">${m.nama_obat}</div>
                      <div class="sw-meta">${m.lokasi_rak} · Batas ${m.stok_minimum} ${m.satuan_dasar}</div>
                    </div>
                    <div class="sw-num fw-semibold" style="font-size: 13px; color: var(--sw-rose-text);">
                      ${getSellableStockBase(m.id_obat)} ${m.satuan_dasar}
                    </div>
                  </li>
                `).join("")}
              </ul>
            `}
          </section>

          ${canFinance ? `
            <section class="sw-panel">
              <div class="sw-panel-header">
                <h2 class="sw-section-title">Tagihan tempo PBF (contoh)</h2>
                <span class="sw-figure-sm">${formatRp(unpaidBillsTotal)}</span>
              </div>
              <ul class="sw-rule-list">
                ${PBF_BILLS.map((bill) => `
                  <li class="sw-rule-item" style="font-size: 13px;">
                    <div>
                      <div class="fw-semibold">${bill.no_faktur}</div>
                      <div class="sw-meta">${bill.pbf}</div>
                    </div>
                    <span class="sw-badge ${bill.status === "TERLAMBAT" ? "sw-badge-rose" : "sw-badge-amber"}">
                      ${bill.status === "TERLAMBAT" ? "Terlambat" : "Jatuh tempo H-2"}
                    </span>
                  </li>
                `).join("")}
              </ul>
            </section>
          ` : ""}
        </div>
      </div>
    `;
  }

  function renderPOS() {
    if (!ROLES[state.role].allowedRoutes.includes("pos")) {
      return renderForbiddenView("Kasir / Point of Sale");
    }

    const selectedMed = getMedicineById(state.pos.selectedMedId) || MEDICINES[0];
    const selectedSatuan = getSatuanById(selectedMed, state.pos.selectedSatuanId) || selectedMed.satuans[0];
    const sellableBase = getSellableStockBase(selectedMed.id_obat);
    const qtyValidation = validatePosQty(state.pos.qtyRaw, selectedMed, selectedSatuan);
    const blockedBatches = BATCHES.filter((b) => b.id_obat === selectedMed.id_obat && b.stok_tersisa > 0 && daysUntilED(b.tanggal_kedaluwarsa) <= 30);

    const preview = previewFefoAllocation(state.pos.cart);
    const pay = computePayment(preview.subtotal, state.pos.paymentMethod, state.pos.cashRaw);
    const canCheckout = preview.lines.length > 0 && !preview.hasInsufficient && pay.cashValid;

    return `
      ${renderPageHeader({
        title: "Kasir / Point of Sale",
        subtitle: "Data contoh · 18 November 2026",
        actionsHtml: `
          <div class="d-flex flex-wrap align-items-center gap-2">
            <span class="sw-meta"><kbd class="sw-kbd">F2</kbd> Cari obat</span>
            <span class="sw-meta"><kbd class="sw-kbd">F8</kbd> Periksa &amp; bayar</span>
            <span class="sw-meta"><kbd class="sw-kbd">Esc</kbd> Tutup pencarian</span>
          </div>
        `,
      })}

      ${state.pos.notice ? `
        <div class="sw-notice sw-notice-${state.pos.notice.type} mb-3 d-flex justify-content-between align-items-center gap-2">
          <span>${state.pos.notice.text}</span>
          <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" id="btn-dismiss-notice">Tutup</button>
        </div>
      ` : ""}

      <div class="sw-split-grid">
        <div class="sw-stack">
          <section class="sw-panel">
            <div class="sw-panel-header mb-3">
              <h2 class="sw-section-title">Pilih obat</h2>
              <span class="sw-meta">Alokasi FEFO otomatis</span>
            </div>

            <div class="sw-search-dropdown mb-2">
              <label for="pos-search-input" class="sw-label">Cari nama, kode, kandungan, lokasi rak, atau pindai barcode</label>
              <div class="input-group">
                <span class="input-group-text">${ICONS.search}</span>
                <input
                  type="text"
                  id="pos-search-input"
                  class="form-control"
                  placeholder="Contoh: Paracetamol, OB001, RAK-A1, atau 8991001001012"
                  autocomplete="off"
                >
              </div>
              <div id="pos-search-results" class="sw-search-results d-none"></div>
            </div>

            <div class="d-flex flex-wrap gap-2 mb-3">
              ${MEDICINES.map((m) => `
                <button
                  type="button"
                  class="sw-btn sw-btn-sm ${m.id_obat === selectedMed.id_obat ? "sw-btn-primary" : "sw-btn-secondary"}"
                  data-quick-med="${m.id_obat}"
                >
                  ${m.nama_obat}
                </button>
              `).join("")}
            </div>

            <div class="sw-panel-inset">
              <div class="d-flex flex-wrap justify-content-between align-items-start gap-2 mb-3">
                <div>
                  <div class="fw-bold" style="font-size: 16px;">${selectedMed.nama_obat}</div>
                  <div class="sw-meta">${selectedMed.kode_obat} · ${selectedMed.lokasi_rak} · ${selectedMed.golongan_obat}</div>
                </div>
                <span class="sw-badge ${sellableBase > 0 ? "sw-badge-sage" : "sw-badge-rose"}">
                  Stok layak jual: ${sellableBase} ${selectedMed.satuan_dasar}
                </span>
              </div>

              ${blockedBatches.length > 0 ? `
                <div class="sw-notice sw-notice-amber mb-3">
                  <strong>ED &le; 30 hari:</strong> Batch ${blockedBatches.map((b) => b.nomor_batch).join(", ")} tidak dialokasikan ke kasir.
                </div>
              ` : ""}

              <div class="row g-2 align-items-start">
                <div class="col-12 col-sm-5">
                  <label for="pos-satuan-select" class="sw-label">Satuan jual</label>
                  <select id="pos-satuan-select" class="form-select sw-control">
                    ${selectedMed.satuans.filter((s) => s.boleh_dijual).map((s) => `
                      <option value="${s.id_obat_satuan}" ${s.id_obat_satuan === selectedSatuan.id_obat_satuan ? "selected" : ""}>
                        ${s.nama_satuan} (isi ${s.faktor_konversi} ${selectedMed.satuan_dasar}) · ${formatRp(s.harga_jual)}
                      </option>
                    `).join("")}
                  </select>
                </div>
                <div class="col-6 col-sm-3">
                  <label for="pos-qty-input" class="sw-label">Jumlah kemasan</label>
                  <input
                    id="pos-qty-input"
                    type="number"
                    min="1"
                    step="1"
                    class="form-control sw-control sw-num ${qtyValidation.valid ? "" : "is-invalid"}"
                    value="${state.pos.qtyRaw}"
                    aria-describedby="pos-qty-error"
                  >
                </div>
                <div class="col-6 col-sm-4">
                  <div class="sw-label">Konversi satuan dasar</div>
                  <div id="pos-needed-base" class="fw-semibold sw-num" style="font-size: 14px; padding: 7px 0;">
                    ${qtyValidation.neededBase} ${selectedMed.satuan_dasar}
                  </div>
                </div>
              </div>

              <div id="pos-qty-error" class="sw-field-error ${qtyValidation.valid ? "d-none" : ""}">${qtyValidation.error}</div>

              <div class="d-flex flex-wrap align-items-center justify-content-between gap-2 mt-3">
                <button
                  type="button"
                  id="btn-add-to-cart"
                  class="sw-btn sw-btn-primary"
                  ${qtyValidation.valid ? "" : "disabled"}
                >
                  ${ICONS.plus} Tambahkan ke keranjang
                </button>
                <span class="sw-meta">Nomor batch ditentukan sistem saat alokasi FEFO.</span>
              </div>
            </div>
          </section>

          <section class="sw-panel">
            <div class="sw-panel-header">
              <div>
                <h2 class="sw-section-title">Keranjang &amp; pengambilan rak</h2>
                <div class="sw-meta mt-1">Urutan batch dihitung otomatis; periksa kecocokan fisik sebelum serah obat</div>
              </div>
              <span class="sw-meta sw-num">${preview.lines.length} baris</span>
            </div>

            ${preview.lines.length === 0 ? `
              <div class="sw-panel-inset text-center py-4">
                <div class="fw-semibold" style="font-size: 14px;">Keranjang masih kosong</div>
                <div class="sw-meta mt-1">Pilih obat di atas lalu tekan Tambahkan ke keranjang.</div>
              </div>
            ` : `
              <div class="sw-table-wrap mb-3">
                <table class="sw-table">
                  <thead>
                    <tr>
                      <th>Obat &amp; batch FEFO</th>
                      <th>Jumlah</th>
                      <th class="text-end">Subtotal</th>
                      <th class="text-end">Hapus</th>
                    </tr>
                  </thead>
                  <tbody>
                    ${preview.lines.map((line, idx) => `
                      <tr>
                        <td>
                          <div class="fw-semibold">${line.med.nama_obat}</div>
                          <div class="sw-meta">${line.med.lokasi_rak} · ${line.neededBase} ${line.med.satuan_dasar}</div>
                          <div class="d-flex flex-wrap gap-1 mt-2">
                            ${line.allocations.map((a) => `
                              <span class="sw-badge ${a.days <= 60 ? "sw-badge-amber" : "sw-badge-sage"}">
                                #${a.urutan_fefo} ${a.nomor_batch} (${a.jumlah_dasar} ${line.med.satuan_dasar} · ED ${formatDateID(a.tanggal_kedaluwarsa)})
                              </span>
                            `).join("")}
                          </div>
                          ${line.skippedLooseBatch ? `
                            <div class="sw-meta mt-1" style="color: var(--sw-amber-text);">
                              Prioritas kemasan utuh: melewati sisa ${line.skippedLooseBatch.sisa} ${line.med.satuan_dasar} di batch ${line.skippedLooseBatch.nomor_batch}.
                            </div>
                          ` : ""}
                          ${!line.sufficient ? `
                            <div class="sw-field-error">
                              Stok layak jual tersisa ${line.totalAvail} ${line.med.satuan_dasar}. Kurangi jumlah pada baris ini.
                            </div>
                          ` : ""}
                        </td>
                        <td class="sw-num">
                          <div class="d-inline-flex align-items-center gap-1">
                            <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-cart-dec="${idx}" aria-label="Kurangi jumlah ${line.med.nama_obat}">−</button>
                            <span class="px-1 fw-semibold">${line.qty} ${line.sat.nama_satuan}</span>
                            <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-cart-inc="${idx}" aria-label="Tambah jumlah ${line.med.nama_obat}">+</button>
                          </div>
                        </td>
                        <td class="text-end sw-num fw-semibold">${formatRp(line.subtotal)}</td>
                        <td class="text-end">
                          <button type="button" class="sw-btn sw-btn-danger-outline sw-btn-sm" data-cart-del="${idx}">Hapus</button>
                        </td>
                      </tr>
                    `).join("")}
                  </tbody>
                </table>
              </div>

              <div class="d-flex justify-content-end">
                <button type="button" id="btn-clear-cart" class="sw-btn sw-btn-danger-outline sw-btn-sm">
                  ${ICONS.trash} Kosongkan keranjang
                </button>
              </div>
            `}
          </section>
        </div>

        <div class="sw-stack">
          <section class="sw-panel">
            <div class="sw-panel-header mb-3">
              <h2 class="sw-section-title">Pembayaran</h2>
            </div>

            <div class="d-flex justify-content-between mb-2" style="font-size: 13px;">
              <span class="sw-meta">Subtotal</span>
              <span class="sw-num fw-semibold">${formatRp(preview.subtotal)}</span>
            </div>
            <div class="d-flex justify-content-between mb-2" style="font-size: 13px;">
              <span class="sw-meta">Pembulatan tunai</span>
              <span class="sw-num fw-semibold">${pay.pembulatan >= 0 ? formatRp(pay.pembulatan) : `- ${formatRp(Math.abs(pay.pembulatan))}`}</span>
            </div>

            <div class="pt-2 mb-3" style="border-top: 1px solid var(--sw-divider);">
              <div class="sw-meta">Total dibayar</div>
              <div class="sw-figure mt-1">${formatRp(pay.totalBayar)}</div>
            </div>

            <div class="mb-3">
              <label for="pos-payment-method" class="sw-label">Metode pembayaran</label>
              <select id="pos-payment-method" class="form-select sw-control">
                <option value="TUNAI" ${state.pos.paymentMethod === "TUNAI" ? "selected" : ""}>Tunai</option>
                <option value="QRIS" ${state.pos.paymentMethod === "QRIS" ? "selected" : ""}>QRIS</option>
                <option value="DEBIT" ${state.pos.paymentMethod === "DEBIT" ? "selected" : ""}>Kartu Debit</option>
              </select>
            </div>

            ${state.pos.paymentMethod === "TUNAI" ? `
              <div class="mb-3">
                <label for="pos-cash-input" class="sw-label">Uang diterima (Rp)</label>
                <input
                  id="pos-cash-input"
                  type="number"
                  min="0"
                  step="500"
                  class="form-control sw-control sw-num ${pay.cashValid ? "" : "is-invalid"}"
                  value="${state.pos.cashRaw}"
                  aria-describedby="pos-cash-error"
                >
                <div id="pos-cash-error" class="sw-field-error ${pay.cashValid ? "d-none" : ""}">${pay.cashError}</div>
                <div class="d-flex flex-wrap gap-1 mt-2">
                  <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-cash-preset="${pay.totalBayar}">Uang pas</button>
                  <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-cash-preset="10000">10 rb</button>
                  <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-cash-preset="20000">20 rb</button>
                  <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-cash-preset="50000">50 rb</button>
                  <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-cash-preset="100000">100 rb</button>
                </div>
              </div>
              <div class="d-flex justify-content-between align-items-center py-2 mb-3" style="border-top: 1px solid var(--sw-divider); border-bottom: 1px solid var(--sw-divider); font-size: 13px;">
                <span class="sw-meta">Kembalian</span>
                <span id="pos-change-display" class="sw-num fw-bold" style="font-size: 16px; color: ${pay.cashValid ? "var(--sw-forest)" : "var(--sw-rose-text)"};">
                  ${pay.cashValid ? formatRp(pay.kembalian) : "Belum mencukupi"}
                </span>
              </div>
            ` : `
              <div class="sw-notice mb-3">
                Nominal tagihan ${state.pos.paymentMethod}: <strong>${formatRp(pay.totalBayar)}</strong> (tanpa pembulatan tunai).
              </div>
            `}

            <button
              type="button"
              id="btn-open-checkout"
              class="sw-btn sw-btn-primary w-100"
              ${canCheckout ? "" : "disabled"}
            >
              Periksa &amp; bayar <kbd class="sw-kbd ms-1" style="background: rgba(255,255,255,0.16); border-color: rgba(255,255,255,0.28); color: #fff;">F8</kbd>
            </button>

            <p class="sw-meta mt-2 mb-0">
              Tunai dibulatkan ke kelipatan Rp100 terdekat.
            </p>
          </section>
        </div>
      </div>
    `;
  }

  function renderStokPage() {
    const canFinance = ROLES[state.role].canSeeFinance;

    const allRows = BATCHES.map((b) => ({
      batch: b,
      med: getMedicineById(b.id_obat),
      z: getBatchZone(b),
    }));

    const countMerah = allRows.filter((r) => r.z.zone === "MERAH").length;
    const countKuning = allRows.filter((r) => r.z.zone === "KUNING").length;
    const countHijau = allRows.filter((r) => r.z.zone === "HIJAU").length;
    const racks = [...new Set(MEDICINES.map((m) => m.lokasi_rak))].sort();

    const q = state.stockQuery.trim().toLowerCase();
    const filtered = allRows
      .filter(({ batch, med, z }) => {
        if (state.zoneFilter !== "ALL" && z.zone !== state.zoneFilter) return false;
        if (state.stockRackFilter !== "ALL" && med.lokasi_rak !== state.stockRackFilter) return false;
        if (q) {
          const hay = `${med.nama_obat} ${med.kode_obat} ${batch.nomor_batch} ${med.lokasi_rak}`.toLowerCase();
          if (!hay.includes(q)) return false;
        }
        return true;
      })
      .sort((a, b) => a.z.days - b.z.days);

    const totalAssetSellable = filtered.reduce((sum, { batch }) => {
      return batch.stok_tersisa > 0 ? sum + Math.round(batch.stok_tersisa * batch.harga_beli_dasar) : sum;
    }, 0);

    return `
      ${renderPageHeader({
        title: "Stok Batch &amp; Zonasi ED",
        subtitle: "Data contoh · Kuantitas dalam satuan dasar; karantina dipisahkan dari alokasi kasir.",
        actionsHtml: `<button type="button" class="sw-btn sw-btn-secondary" id="btn-export-csv">Unduh CSV</button>`,
      })}

      <div class="row g-3 mb-3">
        <div class="col-12 col-md-4">
          <button type="button" class="sw-zone-card sw-zone-card-rose ${state.zoneFilter === "MERAH" ? "is-selected" : ""}" data-zone-filter="MERAH">
            <div class="d-flex justify-content-between align-items-center mb-1">
              <span class="fw-bold" style="font-size: 13px; color: var(--sw-rose-text);">Zona Merah · ED &le; 30 hari</span>
              <span class="sw-badge sw-badge-rose">${countMerah} batch</span>
            </div>
            <div class="sw-meta">Diblokir dari alokasi kasir, termasuk batch yang sudah kedaluwarsa.</div>
          </button>
        </div>
        <div class="col-12 col-md-4">
          <button type="button" class="sw-zone-card sw-zone-card-amber ${state.zoneFilter === "KUNING" ? "is-selected" : ""}" data-zone-filter="KUNING">
            <div class="d-flex justify-content-between align-items-center mb-1">
              <span class="fw-bold" style="font-size: 13px; color: var(--sw-amber-text);">Zona Kuning · ED 31–60 hari</span>
              <span class="sw-badge sw-badge-amber">${countKuning} batch</span>
            </div>
            <div class="sw-meta">Prioritas keluar otomatis melalui urutan FEFO.</div>
          </button>
        </div>
        <div class="col-12 col-md-4">
          <button type="button" class="sw-zone-card sw-zone-card-sage ${state.zoneFilter === "HIJAU" ? "is-selected" : ""}" data-zone-filter="HIJAU">
            <div class="d-flex justify-content-between align-items-center mb-1">
              <span class="fw-bold" style="font-size: 13px; color: var(--sw-sage-text);">Zona Hijau · ED &gt; 60 hari</span>
              <span class="sw-badge sw-badge-sage">${countHijau} batch</span>
            </div>
            <div class="sw-meta">Di luar ambang peringatan ED.</div>
          </button>
        </div>
      </div>

      <section class="sw-panel">
        <div class="sw-panel-header">
          <h2 class="sw-section-title">Daftar batch</h2>
          <span class="sw-meta sw-num">${filtered.length} batch ditampilkan</span>
        </div>

        <div class="row g-2 mb-3 align-items-end">
          <div class="col-12 col-md-5">
            <label for="stock-search-input" class="sw-label">Cari obat, nomor batch, atau kode</label>
            <input id="stock-search-input" type="text" class="form-control sw-control" placeholder="Contoh: Paracetamol atau PCM-2608" value="${state.stockQuery}">
          </div>
          <div class="col-6 col-md-3">
            <label for="stock-rack-select" class="sw-label">Lokasi rak</label>
            <select id="stock-rack-select" class="form-select sw-control">
              <option value="ALL" ${state.stockRackFilter === "ALL" ? "selected" : ""}>Semua rak</option>
              ${racks.map((r) => `<option value="${r}" ${state.stockRackFilter === r ? "selected" : ""}>${r}</option>`).join("")}
            </select>
          </div>
          <div class="col-6 col-md-2">
            <label for="stock-zone-select" class="sw-label">Zona ED</label>
            <select id="stock-zone-select" class="form-select sw-control">
              <option value="ALL" ${state.zoneFilter === "ALL" ? "selected" : ""}>Semua zona</option>
              <option value="MERAH" ${state.zoneFilter === "MERAH" ? "selected" : ""}>Merah (&le; 30 hari)</option>
              <option value="KUNING" ${state.zoneFilter === "KUNING" ? "selected" : ""}>Kuning (31–60 hari)</option>
              <option value="HIJAU" ${state.zoneFilter === "HIJAU" ? "selected" : ""}>Hijau (&gt; 60 hari)</option>
            </select>
          </div>
          <div class="col-12 col-md-2">
            <button type="button" id="btn-reset-stock-filter" class="sw-btn sw-btn-secondary w-100">Reset filter</button>
          </div>
        </div>

        <div class="sw-table-wrap">
          <table class="sw-table">
            <thead>
              <tr>
                <th>Obat / batch</th>
                <th>Lokasi rak</th>
                <th>Tanggal ED</th>
                <th>Status</th>
                <th class="text-end">Stok layak jual</th>
                <th class="text-end">Stok karantina</th>
                ${canFinance ? `<th class="text-end">Harga beli dasar</th><th class="text-end">Nilai aset</th>` : ""}
                <th class="text-end">Riwayat</th>
              </tr>
            </thead>
            <tbody>
              ${filtered.length === 0 ? `
                <tr>
                  <td colspan="${canFinance ? 9 : 7}" class="text-center py-4">
                    <div class="fw-semibold" style="font-size: 14px;">Tidak ada batch yang cocok dengan filter</div>
                    <div class="sw-meta mt-1 mb-2">Ubah kata kunci pencarian atau setel ulang filter zona dan rak.</div>
                    <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" id="btn-empty-reset-filter">Reset filter</button>
                  </td>
                </tr>
              ` : filtered.map(({ batch, med, z }) => {
                const assetVal = Math.round(batch.stok_tersisa * batch.harga_beli_dasar);
                return `
                  <tr>
                    <td>
                      <div class="fw-semibold">${med.nama_obat}</div>
                      <div class="sw-meta">${med.kode_obat} · ${batch.nomor_batch}</div>
                    </td>
                    <td>${med.lokasi_rak}</td>
                    <td>
                      <div class="sw-num">${formatDateID(batch.tanggal_kedaluwarsa)}</div>
                      <div class="sw-meta sw-num">${z.detail}</div>
                    </td>
                    <td><span class="sw-badge ${z.badgeClass}">${z.label}</span></td>
                    <td class="text-end sw-num fw-semibold">${batch.stok_tersisa} ${med.satuan_dasar}</td>
                    <td class="text-end sw-num">${batch.stok_karantina} ${med.satuan_dasar}</td>
                    ${canFinance ? `
                      <td class="text-end sw-num">${formatRp(batch.harga_beli_dasar, 4)}</td>
                      <td class="text-end sw-num fw-semibold">${formatRp(assetVal)}</td>
                    ` : ""}
                    <td class="text-end">
                      <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-audit-batch="${batch.id_batch}">Tampilkan jejak</button>
                    </td>
                  </tr>
                `;
              }).join("")}
            </tbody>
          </table>
        </div>

        ${canFinance ? `
          <div class="d-flex flex-wrap justify-content-between align-items-center gap-2 mt-3 pt-3" style="border-top: 1px solid var(--sw-divider);">
            <div class="sw-meta">Valuasi stok layak jual hasil filter (karantina tidak dihitung)</div>
            <div class="sw-figure-sm">${formatRp(totalAssetSellable)}</div>
          </div>
        ` : ""}
      </section>
    `;
  }

  function renderLaporanLaba() {
    if (!ROLES[state.role].allowedRoutes.includes("laba")) {
      return renderForbiddenView("Laporan Laba Kotor");
    }

    const totalOmzet = SALES_LEDGER.reduce((s, x) => s + x.total_penjualan, 0);
    const totalHpp = SALES_LEDGER.reduce((s, x) => s + x.total_hpp, 0);
    const totalLaba = totalOmzet - totalHpp;
    const avgMargin = totalOmzet > 0 ? ((totalLaba / totalOmzet) * 100).toFixed(1) : null;

    return `
      ${renderPageHeader({
        title: "Laporan Laba Kotor",
        subtitle: "Data contoh · Laba dihitung dari snapshot harga jual dan harga beli batch saat transaksi.",
        actionsHtml: `<button type="button" class="sw-btn sw-btn-secondary" id="btn-export-laba-csv">Unduh CSV laba</button>`,
      })}

      <section class="sw-panel mb-3">
        <div class="row g-3">
          <div class="col-12 col-sm-6 col-lg-3">
            <div class="sw-meta">Omzet netto</div>
            <div class="sw-figure mt-1">${formatRp(totalOmzet)}</div>
          </div>
          <div class="col-12 col-sm-6 col-lg-3">
            <div class="sw-meta">Total HPP riil</div>
            <div class="sw-figure mt-1">${formatRp(totalHpp)}</div>
          </div>
          <div class="col-12 col-sm-6 col-lg-3">
            <div class="sw-meta">Laba kotor</div>
            <div class="sw-figure mt-1" style="color: var(--sw-forest);">${formatRp(totalLaba)}</div>
          </div>
          <div class="col-12 col-sm-6 col-lg-3">
            <div class="sw-meta">Margin rata-rata</div>
            <div class="sw-figure mt-1">${avgMargin !== null ? `${avgMargin}%` : "Tidak tersedia"}</div>
          </div>
        </div>
      </section>

      <section class="sw-panel">
        <div class="sw-panel-header">
          <h2 class="sw-section-title">Laba per transaksi</h2>
          <span class="sw-meta sw-num">${SALES_LEDGER.length} nota</span>
        </div>
        <div class="sw-table-wrap">
          <table class="sw-table">
            <thead>
              <tr>
                <th>Nota / waktu</th>
                <th>Kasir / metode</th>
                <th class="text-end">Omzet</th>
                <th class="text-end">HPP batch</th>
                <th class="text-end">Laba kotor</th>
                <th class="text-end">Margin</th>
                <th class="text-end">Rincian</th>
              </tr>
            </thead>
            <tbody>
              ${SALES_LEDGER.map((s, idx) => {
                const laba = s.total_penjualan - s.total_hpp;
                const margin = s.total_penjualan > 0 ? `${((laba / s.total_penjualan) * 100).toFixed(1)}%` : "Tidak tersedia";
                return `
                  <tr>
                    <td>
                      <div class="fw-semibold sw-num">${s.nomor_transaksi}</div>
                      <div class="sw-meta sw-num">${s.waktu}</div>
                    </td>
                    <td>
                      <div>${s.kasir}</div>
                      <div class="sw-meta">${s.metode}</div>
                    </td>
                    <td class="text-end sw-num">${formatRp(s.total_penjualan)}</td>
                    <td class="text-end sw-num">${formatRp(s.total_hpp)}</td>
                    <td class="text-end sw-num fw-semibold" style="color: var(--sw-forest);">${formatRp(laba)}</td>
                    <td class="text-end sw-num">${margin}</td>
                    <td class="text-end">
                      <button type="button" class="sw-btn sw-btn-secondary sw-btn-sm" data-sale-detail="${idx}">Lihat rincian batch</button>
                    </td>
                  </tr>
                `;
              }).join("")}
            </tbody>
          </table>
        </div>
      </section>
    `;
  }

  // Fallback keeps modals working when opened offline via file:// without Bootstrap JS loaded.
  function showModal(id) {
    const el = document.getElementById(id);
    if (!el) return;
    if (window.bootstrap?.Modal) {
      window.bootstrap.Modal.getOrCreateInstance(el).show();
      return;
    }
    el.classList.add("show");
    el.style.display = "block";
    el.style.backgroundColor = "rgba(32, 45, 37, 0.35)";
    el.removeAttribute("aria-hidden");
    el.querySelectorAll('[data-bs-dismiss="modal"]').forEach((btn) => {
      btn.onclick = () => hideModal(id);
    });
  }

  function hideModal(id) {
    const el = document.getElementById(id);
    if (!el) return;
    if (window.bootstrap?.Modal) {
      window.bootstrap.Modal.getInstance(el)?.hide();
      return;
    }
    el.classList.remove("show");
    el.style.display = "none";
    el.setAttribute("aria-hidden", "true");
  }

  function openCheckoutConfirmModal() {
    const preview = previewFefoAllocation(state.pos.cart);
    const pay = computePayment(preview.subtotal, state.pos.paymentMethod, state.pos.cashRaw);
    const bodyEl = document.getElementById("checkout-modal-body");
    if (!bodyEl) return;

    bodyEl.innerHTML = `
      <div class="sw-panel-inset mb-3">
        <div class="sw-meta">Total transaksi (${state.pos.paymentMethod})</div>
        <div class="sw-figure mt-1">${formatRp(pay.totalBayar)}</div>
        ${state.pos.paymentMethod === "TUNAI" ? `
          <div class="sw-meta mt-1 sw-num">
            Uang diterima: ${formatRp(Number(state.pos.cashRaw))} · Kembalian: <strong>${formatRp(pay.kembalian)}</strong>
          </div>
        ` : ""}
      </div>
      <p style="font-size: 13px; color: var(--sw-muted);" class="mb-2">
        Pastikan fisik obat yang diserahkan sesuai dengan alokasi batch FEFO berikut:
      </p>
      <ul class="list-unstyled mb-0 d-flex flex-column gap-2" style="font-size: 13px;">
        ${preview.lines.map((l) => `
          <li class="p-2 rounded-2" style="border: 1px solid var(--sw-divider); background: var(--sw-surface);">
            <div class="fw-semibold">${l.med.nama_obat} (${l.qty} ${l.sat.nama_satuan})</div>
            <div class="sw-meta">
              ${l.allocations.map((a) => `${a.nomor_batch} (${a.jumlah_dasar} ${l.med.satuan_dasar}, ${l.med.lokasi_rak})`).join(" + ")}
            </div>
          </li>
        `).join("")}
      </ul>
    `;

    showModal("checkoutConfirmModal");
  }

  function executeCheckout() {
    const preview = previewFefoAllocation(state.pos.cart);
    const pay = computePayment(preview.subtotal, state.pos.paymentMethod, state.pos.cashRaw);
    if (preview.lines.length === 0 || preview.hasInsufficient || !pay.cashValid) return;

    let totalHpp = 0;
    const trxNo = `PJ-20261118-${String(SALES_LEDGER.length + 1).padStart(3, "0")}`;
    const nowStr = "2026-11-18 19:05";
    const batchSnapshots = [];

    for (const line of preview.lines) {
      for (const alloc of line.allocations) {
        const batch = BATCHES.find((b) => b.id_batch === alloc.id_batch);
        if (batch) {
          batch.stok_tersisa -= alloc.jumlah_dasar;
          totalHpp += alloc.hpp;
          batchSnapshots.push({
            nama_obat: line.med.nama_obat,
            nomor_batch: batch.nomor_batch,
            jumlah_dasar: alloc.jumlah_dasar,
            satuan_dasar: line.med.satuan_dasar,
            harga_beli_saat_jual: alloc.harga_beli_saat_jual,
            hpp: alloc.hpp,
          });
          MUTATIONS.push({
            id_mutasi: MUTATIONS.length + 101,
            id_batch: batch.id_batch,
            jenis_mutasi: "PENJUALAN",
            jumlah_dasar: -alloc.jumlah_dasar,
            ref_nomor: trxNo,
            waktu: nowStr,
            note: `Penjualan ${line.qty} ${line.sat.nama_satuan} ${line.med.nama_obat}`,
          });
        }
      }
    }

    SALES_LEDGER.unshift({
      nomor_transaksi: trxNo,
      waktu: nowStr,
      kasir: ROLES[state.role].userName,
      metode: state.pos.paymentMethod,
      total_penjualan: pay.totalBayar,
      total_hpp: totalHpp,
      batches: batchSnapshots,
    });

    state.pos.lastSale = {
      nomor_transaksi: trxNo,
      waktu: nowStr,
      kasir: ROLES[state.role].userName,
      lines: preview.lines,
      subtotal: preview.subtotal,
      pembulatan: pay.pembulatan,
      total: pay.totalBayar,
      metode: state.pos.paymentMethod,
      bayar: state.pos.paymentMethod === "TUNAI" ? Number(state.pos.cashRaw) : pay.totalBayar,
      kembalian: pay.kembalian,
    };

    state.pos.cart = [];
    state.pos.notice = {
      type: "sage",
      text: `Transaksi ${trxNo} selesai disimpan. Stok batch terpotong sesuai urutan FEFO.`,
    };

    hideModal("checkoutConfirmModal");
    renderAll();
    openReceiptModal(state.pos.lastSale);
  }

  function openReceiptModal(sale) {
    const el = document.getElementById("thermal-receipt-printable");
    if (!el || !sale) return;

    el.innerHTML = `
      <div class="text-center">
        <div class="fw-bold" style="font-size: 14px;">Apotek Sumber Waras</div>
        <div>STRUK PENJUALAN (CONTOH)</div>
        <div>${sale.nomor_transaksi}</div>
        <div style="font-size: 11px;">${sale.waktu} · ${sale.kasir}</div>
      </div>
      <hr class="sw-receipt-hr">
      ${sale.lines.map((l) => `
        <div class="mb-2">
          <div class="fw-bold">${l.med.nama_obat}</div>
          <div class="d-flex justify-content-between">
            <span>${l.qty} ${l.sat.nama_satuan} x ${formatRp(l.sat.harga_jual)}</span>
            <span>${formatRp(l.subtotal)}</span>
          </div>
          <div style="font-size: 10.5px; color: #555;">
            Batch: ${l.allocations.map((a) => a.nomor_batch).join(", ")}
          </div>
        </div>
      `).join("")}
      <hr class="sw-receipt-hr">
      <div class="d-flex justify-content-between"><span>Subtotal</span><span>${formatRp(sale.subtotal)}</span></div>
      <div class="d-flex justify-content-between"><span>Pembulatan</span><span>${formatRp(sale.pembulatan)}</span></div>
      <div class="d-flex justify-content-between fw-bold mt-1"><span>Total</span><span>${formatRp(sale.total)}</span></div>
      <div class="d-flex justify-content-between mt-1"><span>${sale.metode}</span><span>${formatRp(sale.bayar)}</span></div>
      <div class="d-flex justify-content-between"><span>Kembali</span><span>${formatRp(sale.kembalian)}</span></div>
      <hr class="sw-receipt-hr">
      <div class="text-center">Terima kasih. Semoga lekas sehat.</div>
    `;

    showModal("receiptModal");
  }

  function openBatchAuditModal(idBatch) {
    const batch = BATCHES.find((b) => b.id_batch === Number(idBatch));
    if (!batch) return;
    const med = getMedicineById(batch.id_obat);
    const z = getBatchZone(batch);
    const canFinance = ROLES[state.role].canSeeFinance;
    const logs = MUTATIONS.filter((m) => m.id_batch === batch.id_batch).slice().reverse();

    const bodyEl = document.getElementById("batch-audit-body");
    if (!bodyEl) return;

    bodyEl.innerHTML = `
      <div class="sw-panel-inset mb-3">
        <div class="d-flex flex-wrap justify-content-between align-items-start gap-2">
          <div>
            <div class="fw-bold" style="font-size: 15px;">${med.nama_obat} · ${batch.nomor_batch}</div>
            <div class="sw-meta mt-1">
              Sumber: ${batch.pbf} (${batch.no_faktur}) · ${med.lokasi_rak} · Tanggal ED ${formatDateID(batch.tanggal_kedaluwarsa)}
            </div>
          </div>
          <span class="sw-badge ${z.badgeClass}">${z.label}</span>
        </div>
        <div class="row g-2 mt-2 pt-2" style="border-top: 1px solid var(--sw-divider); font-size: 13px;">
          <div class="col-6 col-sm-4">
            <span class="sw-meta d-block">Stok layak jual</span>
            <strong class="sw-num">${batch.stok_tersisa} ${med.satuan_dasar}</strong>
          </div>
          <div class="col-6 col-sm-4">
            <span class="sw-meta d-block">Stok karantina</span>
            <strong class="sw-num">${batch.stok_karantina} ${med.satuan_dasar}</strong>
          </div>
          ${canFinance ? `
            <div class="col-12 col-sm-4">
              <span class="sw-meta d-block">Harga beli dasar</span>
              <strong class="sw-num">${formatRp(batch.harga_beli_dasar, 4)} / ${med.satuan_dasar}</strong>
            </div>
          ` : ""}
        </div>
      </div>

      <h3 class="fw-bold mb-2" style="font-size: 14px;">Riwayat mutasi stok</h3>
      <div class="sw-table-wrap">
        <table class="sw-table">
          <thead>
            <tr>
              <th>Waktu</th>
              <th>Jenis mutasi</th>
              <th>Referensi</th>
              <th class="text-end">Perubahan</th>
              <th>Keterangan</th>
            </tr>
          </thead>
          <tbody>
            ${logs.length === 0 ? `
              <tr><td colspan="5" class="text-center sw-meta py-3">Belum ada catatan mutasi pada batch ini.</td></tr>
            ` : logs.map((m) => `
              <tr>
                <td class="sw-num sw-meta">${m.waktu}</td>
                <td><code>${m.jenis_mutasi}</code></td>
                <td class="sw-meta">${m.ref_nomor}</td>
                <td class="text-end sw-num fw-semibold">${m.jumlah_dasar > 0 ? `+${m.jumlah_dasar}` : m.jumlah_dasar}</td>
                <td>${m.note}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>
    `;

    showModal("batchAuditModal");
  }

  function openSaleDetailModal(index) {
    if (!ROLES[state.role].canSeeFinance) return;
    const sale = SALES_LEDGER[Number(index)];
    if (!sale) return;
    const laba = sale.total_penjualan - sale.total_hpp;

    const bodyEl = document.getElementById("sale-detail-body");
    if (!bodyEl) return;

    bodyEl.innerHTML = `
      <div class="sw-panel-inset mb-3">
        <div class="d-flex flex-wrap justify-content-between align-items-start gap-2">
          <div>
            <div class="fw-bold sw-num" style="font-size: 15px;">${sale.nomor_transaksi}</div>
            <div class="sw-meta mt-1">${sale.waktu} · ${sale.kasir} · ${sale.metode}</div>
          </div>
          <div class="text-end">
            <div class="sw-meta">Laba kotor</div>
            <div class="sw-figure-sm" style="color: var(--sw-forest);">${formatRp(laba)}</div>
          </div>
        </div>
      </div>

      <div class="sw-table-wrap">
        <table class="sw-table">
          <thead>
            <tr>
              <th>Obat</th>
              <th>Nomor batch</th>
              <th class="text-end">Jumlah</th>
              <th class="text-end">Harga beli saat jual</th>
              <th class="text-end">HPP batch</th>
            </tr>
          </thead>
          <tbody>
            ${sale.batches.map((b) => `
              <tr>
                <td class="fw-semibold">${b.nama_obat}</td>
                <td class="sw-num">${b.nomor_batch}</td>
                <td class="text-end sw-num">${b.jumlah_dasar} ${b.satuan_dasar}</td>
                <td class="text-end sw-num">${formatRp(b.harga_beli_saat_jual, 4)}</td>
                <td class="text-end sw-num fw-semibold">${formatRp(b.hpp)}</td>
              </tr>
            `).join("")}
          </tbody>
        </table>
      </div>
    `;

    showModal("saleDetailModal");
  }

  function bindMainEvents() {
    const main = document.getElementById("main-content");
    if (!main) return;

    document.getElementById("role-switcher")?.addEventListener("change", (e) => {
      state.role = e.target.value;
      renderAll();
    });

    main.querySelectorAll("[data-goto]").forEach((btn) => {
      btn.addEventListener("click", () => navigateTo(btn.dataset.goto));
    });

    main.querySelectorAll("[data-switch-role]").forEach((btn) => {
      btn.addEventListener("click", () => {
        state.role = btn.dataset.switchRole;
        renderAll();
      });
    });

    if (state.route === "pos") {
      bindPOSEvents();
    } else if (state.route === "stok") {
      bindStokEvents();
    } else if (state.route === "laba") {
      document.querySelectorAll("[data-sale-detail]").forEach((btn) => {
        btn.addEventListener("click", () => openSaleDetailModal(btn.dataset.saleDetail));
      });
      document.getElementById("btn-export-laba-csv")?.addEventListener("click", exportLabaCSV);
    }
  }

  function bindPOSEvents() {
    document.getElementById("btn-dismiss-notice")?.addEventListener("click", () => {
      state.pos.notice = null;
      renderAll();
    });

    document.querySelectorAll("[data-quick-med]").forEach((btn) => {
      btn.addEventListener("click", () => {
        const med = getMedicineById(btn.dataset.quickMed);
        if (!med) return;
        state.pos.selectedMedId = med.id_obat;
        state.pos.selectedSatuanId = med.satuans[0].id_obat_satuan;
        state.pos.qtyRaw = "1";
        renderAll();
      });
    });

    const searchInput = document.getElementById("pos-search-input");
    const resultsBox = document.getElementById("pos-search-results");
    if (searchInput && resultsBox) {
      const updateSearch = () => {
        const term = searchInput.value.trim().toLowerCase();
        if (!term) {
          resultsBox.classList.add("d-none");
          resultsBox.innerHTML = "";
          return;
        }
        const matches = MEDICINES.filter((m) => {
          const barcodeMatch = m.satuans.some((s) => s.barcode.includes(term));
          return (
            m.nama_obat.toLowerCase().includes(term) ||
            m.kode_obat.toLowerCase().includes(term) ||
            m.kandungan.toLowerCase().includes(term) ||
            m.lokasi_rak.toLowerCase().includes(term) ||
            barcodeMatch
          );
        });
        resultsBox.classList.remove("d-none");
        if (matches.length === 0) {
          resultsBox.innerHTML = `<div class="p-3 sw-meta">Tidak ada obat yang cocok. Periksa ejaan nama, kode obat, atau barcode.</div>`;
          return;
        }
        resultsBox.innerHTML = matches.map((m) => {
          const avail = getSellableStockBase(m.id_obat);
          return `
            <button type="button" class="sw-search-item" data-pick-med="${m.id_obat}" ${avail <= 0 ? "disabled" : ""}>
              <div>
                <div class="fw-semibold">${m.nama_obat}</div>
                <div class="sw-meta">${m.kode_obat} · ${m.lokasi_rak} · ${m.kandungan}</div>
              </div>
              <span class="sw-badge ${avail > 0 ? "sw-badge-sage" : "sw-badge-rose"}">
                ${avail > 0 ? `${avail} ${m.satuan_dasar}` : "Kosong / ED ≤ 30h"}
              </span>
            </button>
          `;
        }).join("");

        resultsBox.querySelectorAll("[data-pick-med]").forEach((itemBtn) => {
          itemBtn.addEventListener("click", () => {
            const med = getMedicineById(itemBtn.dataset.pickMed);
            state.pos.selectedMedId = med.id_obat;
            state.pos.selectedSatuanId = med.satuans[0].id_obat_satuan;
            state.pos.qtyRaw = "1";
            renderAll();
          });
        });
      };

      searchInput.addEventListener("input", updateSearch);
      searchInput.addEventListener("keydown", (e) => {
        if (e.key === "Escape") {
          searchInput.value = "";
          resultsBox.classList.add("d-none");
          searchInput.blur();
        }
      });
    }

    document.getElementById("pos-satuan-select")?.addEventListener("change", (e) => {
      state.pos.selectedSatuanId = Number(e.target.value);
      renderAll();
    });

    document.getElementById("pos-qty-input")?.addEventListener("input", (e) => {
      state.pos.qtyRaw = e.target.value;
      const med = getMedicineById(state.pos.selectedMedId);
      const sat = getSatuanById(med, state.pos.selectedSatuanId);
      const res = validatePosQty(state.pos.qtyRaw, med, sat);

      const neededEl = document.getElementById("pos-needed-base");
      if (neededEl) neededEl.textContent = `${res.neededBase} ${med.satuan_dasar}`;

      const errEl = document.getElementById("pos-qty-error");
      if (errEl) {
        errEl.textContent = res.error;
        errEl.classList.toggle("d-none", res.valid);
      }
      e.target.classList.toggle("is-invalid", !res.valid);

      const btnAdd = document.getElementById("btn-add-to-cart");
      if (btnAdd) btnAdd.disabled = !res.valid;
    });

    document.getElementById("btn-add-to-cart")?.addEventListener("click", () => {
      const med = getMedicineById(state.pos.selectedMedId);
      const sat = getSatuanById(med, state.pos.selectedSatuanId);
      const res = validatePosQty(state.pos.qtyRaw, med, sat);
      if (!res.valid) return;

      const existing = state.pos.cart.find(
        (c) => c.id_obat === state.pos.selectedMedId && c.id_obat_satuan === state.pos.selectedSatuanId
      );
      if (existing) {
        existing.qty += res.qty;
      } else {
        state.pos.cart.push({
          id_obat: state.pos.selectedMedId,
          id_obat_satuan: state.pos.selectedSatuanId,
          qty: res.qty,
        });
      }
      state.pos.qtyRaw = "1";
      renderAll();
    });

    document.querySelectorAll("[data-cart-inc]").forEach((b) => {
      b.addEventListener("click", () => {
        state.pos.cart[Number(b.dataset.cartInc)].qty += 1;
        renderAll();
      });
    });

    document.querySelectorAll("[data-cart-dec]").forEach((b) => {
      b.addEventListener("click", () => {
        const idx = Number(b.dataset.cartDec);
        if (state.pos.cart[idx].qty > 1) {
          state.pos.cart[idx].qty -= 1;
        } else {
          state.pos.cart.splice(idx, 1);
        }
        renderAll();
      });
    });

    document.querySelectorAll("[data-cart-del]").forEach((b) => {
      b.addEventListener("click", () => {
        state.pos.cart.splice(Number(b.dataset.cartDel), 1);
        renderAll();
      });
    });

    document.getElementById("btn-clear-cart")?.addEventListener("click", () => {
      state.pos.cart = [];
      renderAll();
    });

    document.getElementById("pos-payment-method")?.addEventListener("change", (e) => {
      state.pos.paymentMethod = e.target.value;
      renderAll();
    });

    document.getElementById("pos-cash-input")?.addEventListener("input", (e) => {
      state.pos.cashRaw = e.target.value;
      const preview = previewFefoAllocation(state.pos.cart);
      const pay = computePayment(preview.subtotal, state.pos.paymentMethod, state.pos.cashRaw);

      const errEl = document.getElementById("pos-cash-error");
      if (errEl) {
        errEl.textContent = pay.cashError;
        errEl.classList.toggle("d-none", pay.cashValid);
      }
      e.target.classList.toggle("is-invalid", !pay.cashValid);

      const changeEl = document.getElementById("pos-change-display");
      if (changeEl) {
        changeEl.textContent = pay.cashValid ? formatRp(pay.kembalian) : "Belum mencukupi";
        changeEl.style.color = pay.cashValid ? "var(--sw-forest)" : "var(--sw-rose-text)";
      }

      const payBtn = document.getElementById("btn-open-checkout");
      if (payBtn) payBtn.disabled = preview.lines.length === 0 || preview.hasInsufficient || !pay.cashValid;
    });

    document.querySelectorAll("[data-cash-preset]").forEach((btn) => {
      btn.addEventListener("click", () => {
        state.pos.cashRaw = String(btn.dataset.cashPreset);
        renderAll();
      });
    });

    document.getElementById("btn-open-checkout")?.addEventListener("click", openCheckoutConfirmModal);
  }

  function bindStokEvents() {
    document.querySelectorAll("[data-zone-filter]").forEach((card) => {
      card.addEventListener("click", () => {
        const z = card.dataset.zoneFilter;
        state.zoneFilter = state.zoneFilter === z ? "ALL" : z;
        renderAll();
      });
    });

    document.getElementById("stock-search-input")?.addEventListener("input", (e) => {
      state.stockQuery = e.target.value;
      const cursorPos = e.target.selectionStart;
      renderAll();
      const input = document.getElementById("stock-search-input");
      if (input) {
        input.focus();
        input.setSelectionRange(cursorPos, cursorPos);
      }
    });

    document.getElementById("stock-rack-select")?.addEventListener("change", (e) => {
      state.stockRackFilter = e.target.value;
      renderAll();
    });

    document.getElementById("stock-zone-select")?.addEventListener("change", (e) => {
      state.zoneFilter = e.target.value;
      renderAll();
    });

    const resetFilter = () => {
      state.zoneFilter = "ALL";
      state.stockQuery = "";
      state.stockRackFilter = "ALL";
      renderAll();
    };
    document.getElementById("btn-reset-stock-filter")?.addEventListener("click", resetFilter);
    document.getElementById("btn-empty-reset-filter")?.addEventListener("click", resetFilter);

    document.querySelectorAll("[data-audit-batch]").forEach((btn) => {
      btn.addEventListener("click", () => openBatchAuditModal(btn.dataset.auditBatch));
    });

    document.getElementById("btn-export-csv")?.addEventListener("click", exportStockCSV);
  }

  function triggerCSVDownload(filename, headers, rows) {
    const csvContent = [headers.join(","), ...rows.map((r) => r.join(","))].join("\n");
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    link.click();
    URL.revokeObjectURL(url);
  }

  function exportStockCSV() {
    const canFinance = ROLES[state.role].canSeeFinance;
    const headers = ["Kode Obat", "Nama Obat", "Lokasi Rak", "Nomor Batch", "Tanggal ED", "Zona", "Stok Layak Jual", "Stok Karantina", "Satuan Dasar"];
    if (canFinance) headers.push("Harga Beli Dasar", "Nilai Aset");

    const rows = BATCHES.map((b) => {
      const med = getMedicineById(b.id_obat);
      const z = getBatchZone(b);
      const row = [med.kode_obat, `"${med.nama_obat}"`, med.lokasi_rak, b.nomor_batch, b.tanggal_kedaluwarsa, z.zone, b.stok_tersisa, b.stok_karantina, med.satuan_dasar];
      if (canFinance) row.push(b.harga_beli_dasar.toFixed(4), Math.round(b.stok_tersisa * b.harga_beli_dasar));
      return row;
    });

    triggerCSVDownload(`stok-batch-sumber-waras-${REF_DATE_STR}.csv`, headers, rows);
  }

  function exportLabaCSV() {
    if (!ROLES[state.role].canSeeFinance) return;
    const headers = ["Nomor Nota", "Waktu", "Kasir", "Metode", "Omzet", "HPP Batch", "Laba Kotor"];
    const rows = SALES_LEDGER.map((s) => [s.nomor_transaksi, s.waktu, `"${s.kasir}"`, s.metode, s.total_penjualan, s.total_hpp, s.total_penjualan - s.total_hpp]);
    triggerCSVDownload(`laba-kotor-sumber-waras-${REF_DATE_STR}.csv`, headers, rows);
  }

  function renderAll() {
    renderSidebarInto("desktop-sidebar-nav");
    renderSidebarInto("mobile-sidebar-nav");

    const mainEl = document.getElementById("main-content");
    if (!mainEl) return;

    const roleObj = ROLES[state.role];
    const allNavItems = NAV_GROUPS.flatMap((g) => g.items);
    const currentItem = allNavItems.find((i) => i.id === state.route) || allNavItems[0];

    if (!roleObj.allowedRoutes.includes(state.route)) {
      mainEl.innerHTML = renderForbiddenView(currentItem.label);
    } else if (state.route === "beranda") {
      mainEl.innerHTML = renderBeranda();
    } else if (state.route === "pos") {
      mainEl.innerHTML = renderPOS();
    } else if (state.route === "stok") {
      mainEl.innerHTML = renderStokPage();
    } else if (state.route === "laba") {
      mainEl.innerHTML = renderLaporanLaba();
    }

    bindMainEvents();
  }

  document.addEventListener("keydown", (e) => {
    if (state.route !== "pos") return;
    if (e.key === "F2") {
      e.preventDefault();
      document.getElementById("pos-search-input")?.focus();
    } else if (e.key === "F8") {
      e.preventDefault();
      const payBtn = document.getElementById("btn-open-checkout");
      if (payBtn && !payBtn.disabled) openCheckoutConfirmModal();
    }
  });

  document.addEventListener("DOMContentLoaded", () => {
    document.querySelectorAll("[data-brand-home]").forEach((el) => {
      el.addEventListener("click", (e) => {
        e.preventDefault();
        navigateTo("beranda");
      });
    });

    document.getElementById("btn-confirm-sale")?.addEventListener("click", executeCheckout);
    document.getElementById("btn-print-receipt")?.addEventListener("click", () => window.print());

    renderAll();
  });
})();
