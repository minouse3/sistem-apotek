(() => {
  const menu = document.getElementById("pbf-mobile-menu");
  const openButton = document.querySelector('[data-bs-toggle="offcanvas"]');

  if (menu instanceof HTMLElement && openButton instanceof HTMLButtonElement) {
    menu.addEventListener("show.bs.offcanvas", () => {
      openButton.setAttribute("aria-expanded", "true");
    });
    menu.addEventListener("hidden.bs.offcanvas", () => {
      openButton.setAttribute("aria-expanded", "false");
    });
  }

  document.querySelectorAll("[data-pbf-submit]").forEach((form) => {
    form.addEventListener("submit", () => {
      const progress = form.querySelector("[data-pbf-progress]");
      if (progress instanceof HTMLElement) {
        progress.hidden = false;
      }
    });
  });
})();
