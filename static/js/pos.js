(() => {
  const root = document.querySelector("[data-pos-app]");
  if (!(root instanceof HTMLElement)) {
    return;
  }

  const searchInput = root.querySelector("[data-pos-search]");
  const clearSearchBtn = root.querySelector("[data-pos-search-clear]");
  const catalogStatus = root.querySelector("[data-pos-catalog-status]");
  const catalogBody = root.querySelector("[data-pos-catalog-body]");
  const cartEmpty = root.querySelector("[data-pos-cart-empty]");
  const cartContent = root.querySelector("[data-pos-cart-content]");
  const cartList = root.querySelector("[data-pos-cart-list]");
  const clearCartBtn = root.querySelector("[data-pos-clear]");
  const totalEl = root.querySelector("[data-pos-total]");
  const previewStatus = root.querySelector("[data-pos-preview-status]");
  const errorBox = root.querySelector("[data-pos-error]");
  const form = root.querySelector("[data-pos-checkout-form]");
  const methodSelect = root.querySelector("[data-pos-method]");
  const cashGroup = root.querySelector("[data-pos-cash-group]");
  const paidInput = root.querySelector("[data-pos-paid]");
  const paidError = root.querySelector("[data-pos-paid-error]");
  const changeEl = root.querySelector("[data-pos-change]");
  const submitBtn = root.querySelector("[data-pos-submit]");

  const cart = new Map();
  let currentTotal = 0;
  let searchTimer = null;
  let previewAbort = null;

  const formatRupiah = (n) => "Rp" + Number(n || 0).toLocaleString("id-ID");

  const showError = (msg) => {
    if (!(errorBox instanceof HTMLElement)) return;
    errorBox.textContent = msg;
    errorBox.hidden = !msg;
  };

  const setPaidError = (msg) => {
    if (!(paidError instanceof HTMLElement) || !(paidInput instanceof HTMLInputElement)) return;
    paidError.textContent = msg;
    paidError.hidden = !msg;
    paidInput.classList.toggle("is-invalid", Boolean(msg));
  };

  const updatePaymentUI = () => {
    if (!(methodSelect instanceof HTMLSelectElement) || !(cashGroup instanceof HTMLElement)) return;
    const isTunai = methodSelect.value === "TUNAI";
    cashGroup.hidden = !isTunai;
    if (!isTunai) {
      setPaidError("");
      return;
    }
    const paid = Number(paidInput?.value || 0);
    const diff = paid - currentTotal;
    if (changeEl instanceof HTMLElement) {
      changeEl.textContent = diff >= 0 ? formatRupiah(diff) : formatRupiah(0);
    }
  };

  const renderCart = () => {
    const hasItems = cart.size > 0;
    if (cartEmpty instanceof HTMLElement) cartEmpty.hidden = hasItems;
    if (cartContent instanceof HTMLElement) cartContent.hidden = !hasItems;
    if (clearCartBtn instanceof HTMLButtonElement) clearCartBtn.hidden = !hasItems;

    if (!hasItems) {
      currentTotal = 0;
      if (totalEl instanceof HTMLElement) totalEl.textContent = formatRupiah(0);
      if (previewStatus instanceof HTMLElement) previewStatus.textContent = "";
      showError("");
      setPaidError("");
      updatePaymentUI();
      return;
    }

    if (!(cartList instanceof HTMLElement)) return;
    cartList.replaceChildren();
    let total = 0;

    cart.forEach((item) => {
      const subtotal = item.jumlah * item.harga;
      total += subtotal;

      const row = document.createElement("div");
      row.className = "sw-pos-cart-item";

      const head = document.createElement("div");
      head.className = "d-flex justify-content-between align-items-start gap-2";

      const info = document.createElement("div");
      const title = document.createElement("div");
      title.className = "fw-semibold";
      title.textContent = item.nama;
      const meta = document.createElement("div");
      meta.className = "sw-meta sw-num";
      meta.textContent = `${item.satuan} · ${formatRupiah(item.harga)}`;
      info.append(title, meta);

      const removeBtn = document.createElement("button");
      removeBtn.type = "button";
      removeBtn.className = "sw-icon-btn sw-icon-btn-sm sw-icon-btn-danger";
      removeBtn.setAttribute("aria-label", `Hapus ${item.nama}`);
      removeBtn.textContent = "×";
      removeBtn.addEventListener("click", () => {
        cart.delete(item.id);
        renderCart();
        requestPreview();
      });

      head.append(info, removeBtn);

      const controls = document.createElement("div");
      controls.className = "d-flex justify-content-between align-items-center gap-2 mt-2";

      const qtyWrap = document.createElement("div");
      qtyWrap.className = "d-flex align-items-center gap-1";

      const minusBtn = document.createElement("button");
      minusBtn.type = "button";
      minusBtn.className = "sw-btn sw-btn-secondary sw-btn-sm";
      minusBtn.setAttribute("aria-label", `Kurangi ${item.nama}`);
      minusBtn.textContent = "-";
      minusBtn.addEventListener("click", () => {
        if (item.jumlah <= 1) {
          cart.delete(item.id);
        } else {
          item.jumlah -= 1;
        }
        renderCart();
        requestPreview();
      });

      const qtyInput = document.createElement("input");
      qtyInput.type = "number";
      qtyInput.className = "form-control sw-num sw-pos-qty";
      qtyInput.min = "1";
      qtyInput.max = String(item.max);
      qtyInput.value = String(item.jumlah);
      qtyInput.setAttribute("aria-label", `Jumlah ${item.nama}`);
      qtyInput.addEventListener("change", () => {
        const val = parseInt(qtyInput.value, 10);
        if (!Number.isFinite(val) || val < 1) {
          item.jumlah = 1;
        } else if (val > item.max) {
          item.jumlah = item.max;
        } else {
          item.jumlah = val;
        }
        renderCart();
        requestPreview();
      });

      const plusBtn = document.createElement("button");
      plusBtn.type = "button";
      plusBtn.className = "sw-btn sw-btn-secondary sw-btn-sm";
      plusBtn.setAttribute("aria-label", `Tambah ${item.nama}`);
      plusBtn.textContent = "+";
      plusBtn.disabled = item.jumlah >= item.max;
      plusBtn.addEventListener("click", () => {
        if (item.jumlah < item.max) {
          item.jumlah += 1;
          renderCart();
          requestPreview();
        }
      });

      qtyWrap.append(minusBtn, qtyInput, plusBtn);

      const subEl = document.createElement("strong");
      subEl.className = "sw-num";
      subEl.textContent = formatRupiah(subtotal);

      controls.append(qtyWrap, subEl);
      row.append(head, controls);

      if (item.batches && item.batches.length > 0) {
        const batchLine = document.createElement("div");
        batchLine.className = "sw-meta sw-num mt-1";
        batchLine.textContent = "FEFO: " + item.batches.map((b) => `${b.nomor_batch} (${b.jumlah_dasar} ${item.dasar})`).join(", ");
        row.append(batchLine);
      }

      cartList.append(row);
    });

    currentTotal = total;
    if (totalEl instanceof HTMLElement) totalEl.textContent = formatRupiah(total);
    updatePaymentUI();
  };

  const requestPreview = async () => {
    if (cart.size === 0) return;
    if (previewAbort) previewAbort.abort();
    previewAbort = new AbortController();

    if (previewStatus instanceof HTMLElement) {
      previewStatus.textContent = "Memeriksa alokasi FEFO...";
    }

    const items = Array.from(cart.values()).map((it) => ({
      id_obat_satuan: it.id,
      jumlah: it.jumlah,
    }));

    try {
      const res = await fetch("/pos/preview", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ items }),
        signal: previewAbort.signal,
      });
      const data = await res.json();
      if (!res.ok) {
        showError(data.error || "Stok tidak mencukupi.");
        if (previewStatus instanceof HTMLElement) previewStatus.textContent = "";
        return;
      }
      showError("");
      if (Array.isArray(data.items)) {
        data.items.forEach((pItem) => {
          const existing = cart.get(pItem.id_obat_satuan);
          if (existing) {
            existing.batches = pItem.batches || [];
          }
        });
      }
      if (previewStatus instanceof HTMLElement) {
        previewStatus.textContent = "Alokasi FEFO siap diproses.";
      }
      renderCart();
    } catch (err) {
      if (err && err.name === "AbortError") return;
      if (previewStatus instanceof HTMLElement) previewStatus.textContent = "";
    }
  };

  const renderCatalogRows = (products) => {
    if (!(catalogBody instanceof HTMLElement)) return;
    catalogBody.replaceChildren();

    if (!products || products.length === 0) {
      const tr = document.createElement("tr");
      const td = document.createElement("td");
      td.colSpan = 5;
      td.className = "text-center py-4 sw-meta";
      td.textContent = "Tidak ada obat layak jual yang cocok dengan pencarian.";
      tr.append(td);
      catalogBody.append(tr);
      return;
    }

    products.forEach((p) => {
      const tr = document.createElement("tr");

      const tdName = document.createElement("td");
      const nameDiv = document.createElement("div");
      nameDiv.className = "fw-semibold";
      nameDiv.textContent = p.nama_obat;
      const metaDiv = document.createElement("div");
      metaDiv.className = "sw-meta";
      metaDiv.textContent = `${p.kode_obat} · ${p.nama_satuan} (${p.faktor_konversi} ${p.satuan_dasar})`;
      tdName.append(nameDiv, metaDiv);

      const tdRak = document.createElement("td");
      tdRak.className = "text-nowrap";
      tdRak.textContent = p.lokasi_rak;

      const tdStok = document.createElement("td");
      tdStok.className = "text-end sw-num text-nowrap";
      const stokDiv = document.createElement("div");
      stokDiv.textContent = `${p.max_paket} ${p.nama_satuan}`;
      tdStok.append(stokDiv);
      if (p.batch_terdekat) {
        const edDiv = document.createElement("div");
        edDiv.className = "sw-meta";
        edDiv.textContent = `ED ${p.batch_terdekat}`;
        tdStok.append(edDiv);
      }

      const tdHarga = document.createElement("td");
      tdHarga.className = "text-end sw-num fw-semibold text-nowrap";
      tdHarga.textContent = formatRupiah(p.harga_jual);

      const tdAction = document.createElement("td");
      tdAction.className = "text-end";
      const btn = document.createElement("button");
      btn.type = "button";
      btn.className = "sw-btn sw-btn-secondary sw-btn-sm";
      btn.setAttribute("data-pos-add", "");
      btn.dataset.id = String(p.id_obat_satuan);
      btn.dataset.nama = p.nama_obat;
      btn.dataset.satuan = p.nama_satuan;
      btn.dataset.dasar = p.satuan_dasar;
      btn.dataset.faktor = String(p.faktor_konversi);
      btn.dataset.harga = String(p.harga_jual);
      btn.dataset.max = String(p.max_paket);
      btn.textContent = "+ Tambah";
      tdAction.append(btn);

      tr.append(tdName, tdRak, tdStok, tdHarga, tdAction);
      catalogBody.append(tr);
    });
  };

  const fetchProducts = async (keyword) => {
    if (catalogStatus instanceof HTMLElement) {
      catalogStatus.textContent = "Mencari obat...";
      catalogStatus.classList.remove("sw-notice-rose");
      catalogStatus.hidden = false;
    }
    try {
      const res = await fetch(`/pos/produk?q=${encodeURIComponent(keyword)}`);
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data.error || "Gagal memuat obat.");
      }
      if (catalogStatus instanceof HTMLElement) catalogStatus.hidden = true;
      renderCatalogRows(data.products || []);
    } catch {
      if (catalogStatus instanceof HTMLElement) {
        catalogStatus.textContent = "Gagal mencari obat. Periksa koneksi lalu coba lagi.";
        catalogStatus.classList.add("sw-notice-rose");
        catalogStatus.hidden = false;
      }
    }
  };

  if (catalogBody instanceof HTMLElement) {
    catalogBody.addEventListener("click", (event) => {
      if (!(event.target instanceof Element)) return;
      const btn = event.target.closest("[data-pos-add]");
      if (!(btn instanceof HTMLButtonElement)) return;

      const id = parseInt(btn.dataset.id || "", 10);
      const max = parseInt(btn.dataset.max || "0", 10);
      if (!id || max <= 0) return;

      const existing = cart.get(id);
      if (existing) {
        if (existing.jumlah < max) {
          existing.jumlah += 1;
        } else {
          showError(`Jumlah ${existing.nama} sudah mencapai stok layak jual.`);
          return;
        }
      } else {
        cart.set(id, {
          id,
          nama: btn.dataset.nama || "",
          satuan: btn.dataset.satuan || "",
          dasar: btn.dataset.dasar || "",
          faktor: parseInt(btn.dataset.faktor || "1", 10),
          harga: parseInt(btn.dataset.harga || "0", 10),
          max,
          jumlah: 1,
          batches: [],
        });
      }
      showError("");
      renderCart();
      requestPreview();
    });
  }

  if (searchInput instanceof HTMLInputElement) {
    searchInput.addEventListener("input", () => {
      if (clearSearchBtn instanceof HTMLButtonElement) {
        clearSearchBtn.hidden = searchInput.value.trim() === "";
      }
      clearTimeout(searchTimer);
      searchTimer = setTimeout(() => fetchProducts(searchInput.value.trim()), 200);
    });
  }

  if (clearSearchBtn instanceof HTMLButtonElement && searchInput instanceof HTMLInputElement) {
    clearSearchBtn.addEventListener("click", () => {
      searchInput.value = "";
      clearSearchBtn.hidden = true;
      fetchProducts("");
      searchInput.focus();
    });
  }

  if (clearCartBtn instanceof HTMLButtonElement) {
    clearCartBtn.addEventListener("click", () => {
      cart.clear();
      renderCart();
    });
  }

  if (methodSelect instanceof HTMLSelectElement) {
    methodSelect.addEventListener("change", updatePaymentUI);
  }

  if (paidInput instanceof HTMLInputElement) {
    paidInput.addEventListener("input", () => {
      setPaidError("");
      updatePaymentUI();
    });
  }

  if (form instanceof HTMLFormElement) {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      if (cart.size === 0) {
        showError("Tambahkan minimal satu obat ke keranjang.");
        return;
      }

      const method = methodSelect instanceof HTMLSelectElement ? methodSelect.value : "TUNAI";
      let paid = currentTotal;
      if (method === "TUNAI") {
        paid = parseInt(paidInput?.value || "0", 10);
        if (!Number.isFinite(paid) || paid < currentTotal) {
          setPaidError("Jumlah bayar tunai kurang dari total belanja.");
          paidInput?.focus();
          return;
        }
      }

      if (submitBtn instanceof HTMLButtonElement) submitBtn.disabled = true;
      showError("");

      try {
        const res = await fetch("/pos/checkout", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            metode_pembayaran: method,
            jumlah_bayar: paid,
            items: Array.from(cart.values()).map((it) => ({
              id_obat_satuan: it.id,
              jumlah: it.jumlah,
            })),
          }),
        });
        const data = await res.json();
        if (!res.ok) {
          showError(data.error || "Checkout gagal diproses.");
          if (submitBtn instanceof HTMLButtonElement) submitBtn.disabled = false;
          return;
        }
        window.location.assign(data.redirect || `/pos/struk/${data.id_penjualan}`);
      } catch {
        showError("Tidak dapat terhubung ke server. Coba lagi.");
        if (submitBtn instanceof HTMLButtonElement) submitBtn.disabled = false;
      }
    });
  }

  document.addEventListener("keydown", (event) => {
    if (event.key === "F2" && searchInput instanceof HTMLInputElement) {
      event.preventDefault();
      searchInput.focus();
      searchInput.select();
    } else if (event.key === "F8" && form instanceof HTMLFormElement && cart.size > 0) {
      event.preventDefault();
      form.requestSubmit();
    } else if (event.key === "Escape" && document.activeElement === searchInput && searchInput instanceof HTMLInputElement) {
      if (searchInput.value !== "") {
        searchInput.value = "";
        if (clearSearchBtn instanceof HTMLButtonElement) clearSearchBtn.hidden = true;
        fetchProducts("");
      }
    }
  });
})();
